// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/consumers/notifiers"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	topicKey    = "topic"
	contactKey  = "contact"
	defOffset   = 0
	defLimit    = 20
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc notifiers.Service, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux := chi.NewRouter()

	mux.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			createSubscriptionEndpoint(svc),
			decodeCreate,
			api.EncodeResponse,
			opts...,
		), "create").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listSubscriptionsEndpoint(svc),
			decodeList,
			api.EncodeResponse,
			opts...,
		), "list").ServeHTTP)

		r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
			deleteSubscriptionEndpint(svc),
			decodeSubscription,
			api.EncodeResponse,
			opts...,
		), "delete").ServeHTTP)

		r.Get("/{subID}", otelhttp.NewHandler(kithttp.NewServer(
			viewSubscriptionEndpint(svc),
			decodeSubscription,
			api.EncodeResponse,
			opts...,
		), "view").ServeHTTP)

		r.Delete("/{subID}", otelhttp.NewHandler(kithttp.NewServer(
			deleteSubscriptionEndpint(svc),
			decodeSubscription,
			api.EncodeResponse,
			opts...,
		), "delete").ServeHTTP)
	})

	mux.Get("/health", magistrala.Health("notifier", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createSubReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeSubscription(_ context.Context, r *http.Request) (interface{}, error) {
	req := subReq{
		id:    chi.URLParam(r, "subID"),
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	req := listSubsReq{token: apiutil.ExtractBearerToken(r)}
	vals := r.URL.Query()[topicKey]
	if len(vals) > 0 {
		req.topic = vals[0]
	}

	vals = r.URL.Query()[contactKey]
	if len(vals) > 0 {
		req.contact = vals[0]
	}

	offset, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return listSubsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	req.offset = uint(offset)

	limit, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return listSubsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	req.limit = uint(limit)

	return req, nil
}
