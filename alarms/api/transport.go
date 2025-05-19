// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc alarms.Service, logger *slog.Logger, idp supermq.IDProvider, instanceID string, authn smqauthn.Authentication) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux := chi.NewRouter()

	mux.Route("/{domainID}/alarms", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, true))
			r.Use(api.RequestIDMiddleware(idp))

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listAlarmsEndpoint(svc),
				decodeListAlarmsReq,
				api.EncodeResponse,
				opts...,
			), "list_alarms").ServeHTTP)
			r.Route("/{alarmID}", func(r chi.Router) {
				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					viewAlarmEndpoint(svc),
					decodeAlarmReq,
					api.EncodeResponse,
					opts...,
				), "get_alarm").ServeHTTP)
				r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
					updateAlarmEndpoint(svc),
					decodeUpdateAlarmReq,
					api.EncodeResponse,
					opts...,
				), "update_alarm").ServeHTTP)
				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					deleteAlarmEndpoint(svc),
					decodeAlarmReq,
					api.EncodeResponse,
					opts...,
				), "delete_alarm").ServeHTTP)
			})
		})
	})

	mux.Get("/health", supermq.Health("alarms", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListAlarmsReq(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	domainID, err := apiutil.ReadStringQuery(r, "domain_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	channelID, err := apiutil.ReadStringQuery(r, "channel_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	clientID, err := apiutil.ReadStringQuery(r, "client_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	subtopic, err := apiutil.ReadStringQuery(r, "subtopic", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	ruleID, err := apiutil.ReadStringQuery(r, "rule_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, alarms.All)
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	status, err := alarms.ToStatus(s)
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	assigneeID, err := apiutil.ReadStringQuery(r, "assignee_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	serverity, err := apiutil.ReadNumQuery(r, "severity", uint64(math.MaxUint8))
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	updatedBy, err := apiutil.ReadStringQuery(r, "updated_by", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	assignedBy, err := apiutil.ReadStringQuery(r, "assigned_by", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	acknowledgedBy, err := apiutil.ReadStringQuery(r, "acknowledged_by", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	resolvedBy, err := apiutil.ReadStringQuery(r, "resolved_by", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	cfrom, err := apiutil.ReadStringQuery(r, "created_from", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	cto, err := apiutil.ReadStringQuery(r, "created_to", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	var createdFrom, createdTo time.Time
	if cfrom != "" {
		if createdFrom, err = time.Parse(time.RFC3339, cfrom); err != nil {
			return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
		}
	}
	if cto != "" {
		if createdTo, err = time.Parse(time.RFC3339, cto); err != nil {
			return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
		}
	}

	return listAlarmsReq{
		PageMetadata: alarms.PageMetadata{
			Offset:         offset,
			Limit:          limit,
			DomainID:       domainID,
			ChannelID:      channelID,
			ClientID:       clientID,
			Subtopic:       subtopic,
			RuleID:         ruleID,
			Status:         status,
			AssigneeID:     assigneeID,
			ResolvedBy:     resolvedBy,
			Severity:       uint8(serverity),
			UpdatedBy:      updatedBy,
			AcknowledgedBy: acknowledgedBy,
			AssignedBy:     assignedBy,
			CreatedFrom:    createdFrom,
			CreatedTo:      createdTo,
		},
	}, nil
}

func decodeAlarmReq(_ context.Context, r *http.Request) (interface{}, error) {
	return alarmReq{
		Alarm: alarms.Alarm{
			ID: chi.URLParam(r, "alarmID"),
		},
	}, nil
}

func decodeUpdateAlarmReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return alarmReq{}, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := alarmReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Alarm); err != nil {
		return alarmReq{}, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	req.Alarm.ID = chi.URLParam(r, "alarmID")

	return req, nil
}
