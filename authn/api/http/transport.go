// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const contentType = "application/json"

var errUnsupportedContentType = errors.New("unsupported content type")

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc authn.Service, tracer opentracing.Tracer) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()

	mux.Post("/keys", kithttp.NewServer(
		kitot.TraceServer(tracer, "issue")(issueEndpoint(svc)),
		decodeIssue,
		encodeResponse,
		opts...,
	))

	mux.Get("/keys/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "retrieve")(retrieveEndpoint(svc)),
		decodeKeyReq,
		encodeResponse,
		opts...,
	))

	mux.Delete("/keys/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke")(revokeEndpoint(svc)),
		decodeKeyReq,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/version", mainflux.Version("auth"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeIssue(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}
	req := issueKeyReq{
		token: r.Header.Get("Authorization"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(authn.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	req := keyReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
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
	switch {
	case errors.Contains(err, authn.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, authn.ErrUnauthorizedAccess):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, authn.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, authn.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, io.EOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, io.ErrUnexpectedEOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	errorVal, ok := err.(errors.Error)
	if ok {
		if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
