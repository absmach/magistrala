// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/twins"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
func MakeHandler(svc twins.Service, logger mglog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := chi.NewRouter()

	r.Route("/twins", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			addTwinEndpoint(svc),
			decodeTwinCreation,
			encodeResponse,
			opts...,
		), "add_twin").ServeHTTP)
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listTwinsEndpoint(svc),
			decodeList,
			encodeResponse,
			opts...,
		), "list_twins").ServeHTTP)
		r.Put("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			updateTwinEndpoint(svc),
			decodeTwinUpdate,
			encodeResponse,
			opts...,
		), "update_twin").ServeHTTP)
		r.Get("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			viewTwinEndpoint(svc),
			decodeView,
			encodeResponse,
			opts...,
		), "view_twin").ServeHTTP)
		r.Delete("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			removeTwinEndpoint(svc),
			decodeView,
			encodeResponse,
			opts...,
		), "remove_twin").ServeHTTP)
	})
	r.Get("/states/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
		listStatesEndpoint(svc),
		decodeListStates,
		encodeResponse,
		opts...,
	), "list_states").ServeHTTP)

	r.Get("/health", magistrala.Health("twins", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeTwinCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addTwinReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeTwinUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateTwinReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "twinID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewTwinReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "twinID"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
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
	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listStatesReq{
		token:  apiutil.ExtractBearerToken(r),
		limit:  l,
		offset: o,
		id:     chi.URLParam(r, "twinID"),
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(magistrala.Response); ok {
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
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrNameSize),
		errors.Contains(err, apiutil.ErrLimitSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, svcerr.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, svcerr.ErrConflict):
		w.WriteHeader(http.StatusConflict)

	case errors.Contains(err, svcerr.ErrCreateEntity),
		errors.Contains(err, svcerr.ErrUpdateEntity),
		errors.Contains(err, svcerr.ErrViewEntity),
		errors.Contains(err, svcerr.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
