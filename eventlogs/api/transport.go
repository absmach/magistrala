// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	operationKey = "operation"
	fromKey      = "from"
	toKey        = "to"
)

// MakeHandler returns a HTTP API handler with health check and metrics.
func MakeHandler(svc eventlogs.Service, logger mglog.Logger, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux := chi.NewRouter()

	mux.Get("/events/{id}/{type}", otelhttp.NewHandler(kithttp.NewServer(
		listEventsEndpoint(svc),
		decodeListEventsReq,
		api.EncodeResponse,
		opts...,
	), "list_events").ServeHTTP)

	mux.Get("/health", magistrala.Health(svcName, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListEventsReq(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	operation, err := apiutil.ReadStringQuery(r, operationKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	from, err := apiutil.ReadNumQuery[int64](r, fromKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	if from == 0 {
		from = time.Now().Add(-24 * time.Hour).UnixNano()
	}
	to, err := apiutil.ReadNumQuery[int64](r, toKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	if to == 0 {
		to = time.Now().UnixNano()
	}

	req := listEventsReq{
		token: apiutil.ExtractBearerToken(r),
		page: eventlogs.Page{
			Offset:     offset,
			Limit:      limit,
			ID:         chi.URLParam(r, "id"),
			EntityType: chi.URLParam(r, "type"),
			Operation:  operation,
			From:       time.Unix(0, from),
			To:         time.Unix(0, to),
		},
	}

	return req, nil
}
