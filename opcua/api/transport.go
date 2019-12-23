// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/opcua"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType     = "application/json"
	serverParam     = "server"
	namespaceParam  = "namespace"
	identifierParam = "identifier"

	defOffset = 0
	defLimit  = 10
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc opcua.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Get("/browse", kithttp.NewServer(
		browseEndpoint(svc),
		decodeBrowse,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("opcua-adapter"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeBrowse(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := readStringQuery(r, serverParam)
	if err != nil {
		return nil, err
	}

	n, err := readStringQuery(r, namespaceParam)
	if err != nil {
		return nil, err
	}

	i, err := readStringQuery(r, identifierParam)
	if err != nil {
		return nil, err
	}

	req := browseReq{
		ServerURI:  s,
		Namespace:  n,
		Identifier: i,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case opcua.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case errInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidQueryParams
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
}
