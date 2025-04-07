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

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq"
	sapi "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc alarms.Service, logger *slog.Logger, idp supermq.IDProvider, instanceID string, authn smqauthn.Authentication) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, sapi.EncodeError)),
	}

	mux := chi.NewRouter()
	mux.Use(sapi.AuthenticateMiddleware(authn, true))
	mux.Use(sapi.RequestIDMiddleware(idp))
	mux.Route("/{domainID}/alarms", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			createAlarmEndpoint(svc),
			decodeCreateAlarmReq,
			sapi.EncodeResponse,
			opts...,
		), "create_alarm").ServeHTTP)
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listAlarmsEndpoint(svc),
			decodeListAlarmsReq,
			sapi.EncodeResponse,
			opts...,
		), "list_alarms").ServeHTTP)
		r.Route("/{alarmID}", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				viewAlarmEndpoint(svc),
				decodeAlarmReq,
				sapi.EncodeResponse,
				opts...,
			), "get_alarm").ServeHTTP)
			r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
				updateAlarmEndpoint(svc),
				decodeUpdateAlarmReq,
				sapi.EncodeResponse,
				opts...,
			), "update_alarm").ServeHTTP)
			r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
				deleteAlarmEndpoint(svc),
				decodeAlarmReq,
				sapi.EncodeResponse,
				opts...,
			), "delete_alarm").ServeHTTP)
		})
	})

	return mux
}

func decodeCreateAlarmReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), sapi.ContentType) {
		return createAlarmReq{}, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req createAlarmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return createAlarmReq{}, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeListAlarmsReq(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, sapi.OffsetKey, sapi.DefOffset)
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, sapi.LimitKey, sapi.DefLimit)
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
	ruleID, err := apiutil.ReadStringQuery(r, "rule_id", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	s, err := apiutil.ReadStringQuery(r, sapi.StatusKey, alarms.AllStatus.String())
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
	resolvedBy, err := apiutil.ReadStringQuery(r, "resolved_by", "")
	if err != nil {
		return listAlarmsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	return listAlarmsReq{
		PageMetadata: alarms.PageMetadata{
			Offset:     offset,
			Limit:      limit,
			DomainID:   domainID,
			ChannelID:  channelID,
			RuleID:     ruleID,
			Status:     status,
			AssigneeID: assigneeID,
			ResolvedBy: resolvedBy,
			Severity:   uint8(serverity),
			UpdatedBy:  updatedBy,
		},
	}, nil
}

func decodeAlarmReq(_ context.Context, r *http.Request) (interface{}, error) {
	alarmID, err := apiutil.ReadStringQuery(r, "alarm_id", "")
	if err != nil {
		return entityReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	return entityReq{
		ID: alarmID,
	}, nil
}

func decodeUpdateAlarmReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), sapi.ContentType) {
		return createAlarmReq{}, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createAlarmReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Alarm); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	alarmID, err := apiutil.ReadStringQuery(r, "alarm_id", "")
	if err != nil {
		return createAlarmReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	req.Alarm.ID = alarmID

	return createAlarmReq{
		Alarm: req.Alarm,
	}, nil
}
