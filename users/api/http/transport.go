//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const contentType = "application/json"

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	logger                    log.Logger
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, tracer opentracing.Tracer, l log.Logger) http.Handler {
	logger = l

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()

	mux.Post("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "register")(registrationEndpoint(svc)),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	mux.Post("/tokens", kithttp.NewServer(
		kitot.TraceServer(tracer, "login")(loginEndpoint(svc)),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/version", mainflux.Version("users"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		logger.Warn("Invalid or missing content type.")
		return nil, errUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		logger.Warn(fmt.Sprintf("Failed to decode user credentials: %s", err))
		return nil, err
	}

	return userReq{user}, nil
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
	case users.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case users.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case users.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case io.ErrUnexpectedEOF:
		w.WriteHeader(http.StatusBadRequest)
	case io.EOF:
		w.WriteHeader(http.StatusBadRequest)
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
