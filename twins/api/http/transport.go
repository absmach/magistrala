// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/twins"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	nameKey     = "name"
	metadataKey = "metadata"
	defLimit    = 10
	defOffset   = 0
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc twins.Service, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/twins", kithttp.NewServer(
		kitot.TraceServer(tracer, "add_twin")(addTwinEndpoint(svc)),
		decodeTwinCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/twins/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_twin")(updateTwinEndpoint(svc)),
		decodeTwinUpdate,
		encodeResponse,
		opts...,
	))

	r.Get("/twins/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_twin")(viewTwinEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Delete("/twins/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_twin")(removeTwinEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/twins", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_twins")(listTwinsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Get("/states/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_states")(listStatesEndpoint(svc)),
		decodeListStates,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("twins"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeTwinCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := addTwinReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeTwinUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := updateTwinReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewTwinReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:    apiutil.ExtractBearerToken(r),
		limit:    l,
		offset:   o,
		name:     n,
		metadata: m,
	}

	return req, nil
}

func decodeListStates(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	req := listStatesReq{
		token:  apiutil.ExtractBearerToken(r),
		limit:  l,
		offset: o,
		id:     bone.GetValue(r, "id"),
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
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrInvalidQueryParams):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrLimitSize:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)

	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrViewEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
