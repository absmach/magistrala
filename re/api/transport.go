// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	idKey            = "ruleID"
	inputChannelKey  = "input_channel"
	outputChannelKey = "output_channel"
	statusKey        = "status"
)

// MakeHandler creates an HTTP handler for the service endpoints.
func MakeHandler(svc re.Service, authn mgauthn.Authentication, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux := chi.NewRouter()
	mux.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))
		r.Route("/{domainID}/rules", func(r chi.Router) {
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				addRuleEndpoint(svc),
				decodeAddRuleRequest,
				api.EncodeResponse,
				opts...,
			), "create_rule").ServeHTTP)

			r.Get("/{ruleID}", otelhttp.NewHandler(kithttp.NewServer(
				viewRuleEndpoint(svc),
				decodeViewRuleRequest,
				api.EncodeResponse,
				opts...,
			), "view_rule").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listRulesEndpoint(svc),
				decodeListRulesRequest,
				api.EncodeResponse,
				opts...,
			), "list_rules").ServeHTTP)

			r.Put("/{ruleID}", otelhttp.NewHandler(kithttp.NewServer(
				updateRuleEndpoint(svc),
				decodeUpdateRuleRequest,
				api.EncodeResponse,
				opts...,
			), "update_rule").ServeHTTP)

			r.Delete("/{ruleID}", otelhttp.NewHandler(kithttp.NewServer(
				deleteRuleEndpoint(svc),
				decodeDeleteRuleRequest,
				api.EncodeResponse,
				opts...,
			), "delete_rule").ServeHTTP)

			r.Put("/{ruleID}/enable", otelhttp.NewHandler(kithttp.NewServer(
				enableRuleEndpoint(svc),
				decodeUpdateRuleStatusRequest,
				api.EncodeResponse,
				opts...,
			), "enable_rule").ServeHTTP)

			r.Put("/{ruleID}/disable", otelhttp.NewHandler(kithttp.NewServer(
				disableRuleEndpoint(svc),
				decodeUpdateRuleStatusRequest,
				api.EncodeResponse,
				opts...,
			), "disable_rule").ServeHTTP)
		})
	})

	mux.Get("/health", supermq.Health("rule_engine", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeAddRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, errors.Wrap(err, apiutil.ErrValidation)
	}
	return addRuleReq{Rule: rule}, nil
}

func decodeViewRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, idKey)
	return viewRuleReq{id: id}, nil
}

func decodeUpdateRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, err
	}
	return updateRuleReq{Rule: rule}, nil
}

func decodeUpdateRuleStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := updateRuleStatusReq{
		id: chi.URLParam(r, idKey),
	}
	return req, nil
}

func decodeListRulesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	ic, err := apiutil.ReadStringQuery(r, inputChannelKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	oc, err := apiutil.ReadStringQuery(r, outputChannelKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefStatus)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := re.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	return listRulesReq{
		PageMeta: re.PageMeta{
			Offset:        offset,
			Limit:         limit,
			InputChannel:  ic,
			OutputChannel: oc,
			Status:        st,
		},
	}, nil
}

func decodeDeleteRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, idKey)

	return deleteRuleReq{id: id}, nil
}
