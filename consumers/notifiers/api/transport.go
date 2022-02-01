// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

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
	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/internal/httputil"
	"github.com/mainflux/mainflux/pkg/errors"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const contentType = "application/json"

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc notifiers.Service, tracer opentracing.Tracer) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
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
	var req createSubReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeSubscription(_ context.Context, r *http.Request) (interface{}, error) {
	req := subReq{
		id:    bone.GetValue(r, "id"),
		token: r.Header.Get("Authorization"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	req := listSubsReq{
		token: r.Header.Get("Authorization"),
	}
	vals := bone.GetQuery(r, "topic")
	if len(vals) > 0 {
		req.topic = vals[0]
	}

	vals = bone.GetQuery(r, "contact")
	if len(vals) > 0 {
		req.contact = vals[0]
	}

	offset, err := httputil.ReadUintQuery(r, "offset", 0)
	if err != nil {
		return listSubsReq{}, err
	}
	req.offset = uint(offset)

	limit, err := httputil.ReadUintQuery(r, "limit", 20)
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
	switch errorVal := err.(type) {
	case errors.Error:
		w.Header().Set("Content-Type", contentType)
		switch {
		case errors.Contains(errorVal, errors.ErrMalformedEntity),
			errors.Contains(errorVal, errInvalidContact),
			errors.Contains(errorVal, errInvalidTopic),
			errors.Contains(errorVal, errors.ErrInvalidQueryParams):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, errors.ErrNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Contains(errorVal, errors.ErrAuthentication):
			w.WriteHeader(http.StatusUnauthorized)
		case errors.Contains(errorVal, errors.ErrConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Contains(errorVal, errors.ErrUnsupportedContentType):
			w.WriteHeader(http.StatusUnsupportedMediaType)
		case errors.Contains(errorVal, io.ErrUnexpectedEOF):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, io.EOF):
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		if errorVal.Msg() != "" {
			if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
