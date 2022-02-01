// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/internal/httputil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	defOffset   = 0
	defLimit    = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/certs", kithttp.NewServer(
		issueCert(svc),
		decodeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/certs/:certId", kithttp.NewServer(
		viewCert(svc),
		decodeViewCert,
		encodeResponse,
		opts...,
	))

	r.Delete("/certs/:certId", kithttp.NewServer(
		revokeCert(svc),
		decodeRevokeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/serials/:thingId", kithttp.NewServer(
		listSerials(svc),
		decodeListCerts,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("certs"))

	return r
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

func decodeListCerts(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := httputil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}
	o, err := httputil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}
	req := listReq{
		token:   r.Header.Get("Authorization"),
		thingID: bone.GetValue(r, "thingId"),
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeViewCert(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewReq{
		token:    r.Header.Get("Authorization"),
		serialID: bone.GetValue(r, "certId"),
	}

	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.ErrUnsupportedContentType
	}

	req := addCertsReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeRevokeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	req := revokeReq{
		token:  r.Header.Get("Authorization"),
		certID: bone.GetValue(r, "certId"),
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case errors.ErrUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case io.EOF, errors.ErrMalformedEntity,
		errors.ErrInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case errors.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	default:
		switch err.(type) {
		case *json.SyntaxError:
			w.WriteHeader(http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
