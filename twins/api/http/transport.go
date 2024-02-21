// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
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
func MakeHandler(svc twins.Service, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()

	r.Route("/twins", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			addTwinEndpoint(svc),
			decodeTwinCreation,
			api.EncodeResponse,
			opts...,
		), "add_twin").ServeHTTP)
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listTwinsEndpoint(svc),
			decodeList,
			api.EncodeResponse,
			opts...,
		), "list_twins").ServeHTTP)
		r.Put("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			updateTwinEndpoint(svc),
			decodeTwinUpdate,
			api.EncodeResponse,
			opts...,
		), "update_twin").ServeHTTP)
		r.Get("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			viewTwinEndpoint(svc),
			decodeView,
			api.EncodeResponse,
			opts...,
		), "view_twin").ServeHTTP)
		r.Delete("/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
			removeTwinEndpoint(svc),
			decodeView,
			api.EncodeResponse,
			opts...,
		), "remove_twin").ServeHTTP)
	})
	r.Get("/states/{twinID}", otelhttp.NewHandler(kithttp.NewServer(
		listStatesEndpoint(svc),
		decodeListStates,
		api.EncodeResponse,
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
