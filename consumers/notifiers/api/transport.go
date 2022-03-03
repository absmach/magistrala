// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
func MakeHandler(svc notifiers.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux := bone.New()

	mux.Post("/subscriptions", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_subscription")(createSubscriptionEndpoint(svc)),
		decodeCreate,
		encodeResponse,
		opts...,
	))

	mux.Get("/subscriptions/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_subscription")(viewSubscriptionEndpint(svc)),
		decodeSubscription,
		encodeResponse,
		opts...,
	))

	mux.Get("/subscriptions", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_subscriptions")(listSubscriptionsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.Delete("/subscriptions/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_subscription")(deleteSubscriptionEndpint(svc)),
		decodeSubscription,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("notifier"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := createSubReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeSubscription(_ context.Context, r *http.Request) (interface{}, error) {
	req := subReq{
		id:    bone.GetValue(r, "id"),
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	req := listSubsReq{token: apiutil.ExtractBearerToken(r)}
	vals := bone.GetQuery(r, topicKey)
	if len(vals) > 0 {
		req.topic = vals[0]
	}

	vals = bone.GetQuery(r, contactKey)
	if len(vals) > 0 {
		req.contact = vals[0]
	}

	offset, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return listSubsReq{}, err
	}
	req.offset = uint(offset)

	limit, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return listSubsReq{}, err
	}
	req.limit = uint(limit)

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(mainflux.Response); ok {
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
	switch {
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidContact,
		err == apiutil.ErrInvalidTopic,
		err == apiutil.ErrMissingID,
		errors.Contains(err, errors.ErrInvalidQueryParams):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)

	case errors.Contains(err, errors.ErrCreateEntity),
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
