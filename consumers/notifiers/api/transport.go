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
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
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
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux := chi.NewRouter()

	mux.Route("/subscriptions", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			createSubscriptionEndpoint(svc),
			decodeCreate,
			encodeResponse,
			opts...,
		), "create").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listSubscriptionsEndpoint(svc),
			decodeList,
			encodeResponse,
			opts...,
		), "list").ServeHTTP)

		r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
			deleteSubscriptionEndpint(svc),
			decodeSubscription,
			encodeResponse,
			opts...,
		), "delete").ServeHTTP)

		r.Get("/{subID}", otelhttp.NewHandler(kithttp.NewServer(
			viewSubscriptionEndpint(svc),
			decodeSubscription,
			encodeResponse,
			opts...,
		), "view").ServeHTTP)

		r.Delete("/{subID}", otelhttp.NewHandler(kithttp.NewServer(
			deleteSubscriptionEndpint(svc),
			decodeSubscription,
			encodeResponse,
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

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(magistrala.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", contentType)
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
	case errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrInvalidContact),
		errors.Contains(err, apiutil.ErrInvalidTopic),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrInvalidQueryParams):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, svcerr.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, svcerr.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)

	case errors.Contains(err, svcerr.ErrCreateEntity),
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
