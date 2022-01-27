// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/httputil"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType     = "application/json"
	serverParam     = "server"
	namespaceParam  = "namespace"
	identifierParam = "identifier"
	defOffset       = 0
	defLimit        = 10
	defNamespace    = "ns=0" // Standard root namespace
	defIdentifier   = "i=84" // Standard root identifier
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

	r.GetFunc("/health", mainflux.Health("opcua-adapter"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeBrowse(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := httputil.ReadStringQuery(r, serverParam, "")
	if err != nil {
		return nil, err
	}

	n, err := httputil.ReadStringQuery(r, namespaceParam, "")
	if err != nil {
		return nil, err
	}

	i, err := httputil.ReadStringQuery(r, identifierParam, "")
	if err != nil {
		return nil, err
	}

	if n == "" || i == "" {
		n = defNamespace
		i = defIdentifier
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
	case errors.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case errors.ErrInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
