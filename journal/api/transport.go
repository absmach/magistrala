// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/journal"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	operationKey  = "operation"
	fromKey       = "from"
	toKey         = "to"
	attributesKey = "with_attributes"
	metadataKey   = "with_metadata"
	entityIDKey   = "id"
	entityTypeKey = "entity_type"
)

// MakeHandler returns a HTTP API handler with health check and metrics.
func MakeHandler(svc journal.Service, authn smqauthn.Authentication, logger *slog.Logger, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux := chi.NewRouter()
	idp := uuid.New()
	mux.Use(api.RequestIDMiddleware(idp))

	mux.With(api.AuthenticateMiddleware(authn, false)).Get("/journal/user/{userID}", otelhttp.NewHandler(kithttp.NewServer(
		retrieveJournalsEndpoint(svc),
		decodeRetrieveUserJournalReq,
		api.EncodeResponse,
		opts...,
	), "list_user_journals").ServeHTTP)

	mux.Route("/{domainID}/journal", func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))

		r.Get("/{entityType}/{entityID}", otelhttp.NewHandler(kithttp.NewServer(
			retrieveJournalsEndpoint(svc),
			decodeRetrieveEntityJournalReq,
			api.EncodeResponse,
			opts...,
		), "list__entity_journals").ServeHTTP)

		r.Get("/client/{clientID}/telemetry", otelhttp.NewHandler(kithttp.NewServer(
			retrieveClientTelemetryEndpoint(svc),
			decodeRetrieveClientTelemetryReq,
			api.EncodeResponse,
			opts...,
		), "view_client_telemetry").ServeHTTP)
	})

	mux.Get("/health", supermq.Health(svcName, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeRetrieveEntityJournalReq(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := decodePageQuery(r)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	entityType, err := journal.ToEntityType(chi.URLParam(r, "entityType"))
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	page.EntityID = chi.URLParam(r, "entityID")
	page.EntityType = entityType

	if entityType == journal.ChannelEntity {
		page.Operation = strings.ReplaceAll(page.Operation, "channel", "group")
	}

	req := retrieveJournalsReq{
		token: apiutil.ExtractBearerToken(r),
		page:  page,
	}

	return req, nil
}

func decodeRetrieveUserJournalReq(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := decodePageQuery(r)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	page.EntityID = chi.URLParam(r, "userID")
	page.EntityType = journal.UserEntity

	req := retrieveJournalsReq{
		token: apiutil.ExtractBearerToken(r),
		page:  page,
	}

	return req, nil
}

func decodePageQuery(r *http.Request) (journal.Page, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	operation, err := apiutil.ReadStringQuery(r, operationKey, "")
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	from, err := apiutil.ReadNumQuery[int64](r, fromKey, 0)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	if from > math.MaxInt32 {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidTimeFormat)
	}
	var fromTime time.Time
	if from != 0 {
		fromTime = time.Unix(from, 0)
	}
	to, err := apiutil.ReadNumQuery[int64](r, toKey, 0)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	if to > math.MaxInt32 {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidTimeFormat)
	}
	var toTime time.Time
	if to != 0 {
		toTime = time.Unix(to, 0)
	}
	attributes, err := apiutil.ReadBoolQuery(r, attributesKey, false)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	metadata, err := apiutil.ReadBoolQuery(r, metadataKey, false)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DescDir)
	if err != nil {
		return journal.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	return journal.Page{
		Offset:         offset,
		Limit:          limit,
		Operation:      operation,
		From:           fromTime,
		To:             toTime,
		WithAttributes: attributes,
		WithMetadata:   metadata,
		Direction:      dir,
	}, nil
}

func decodeRetrieveClientTelemetryReq(_ context.Context, r *http.Request) (interface{}, error) {
	req := retrieveClientTelemetryReq{
		clientID: chi.URLParam(r, "clientID"),
	}

	return req, nil
}
