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
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	roleManagerHttp "github.com/absmach/supermq/pkg/roles/rolemanager/api"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	ruleIdKey       = "ruleID"
	inputChannelKey = "input_channel"
)

// MakeHandler creates an HTTP handler for the service endpoints.
func MakeHandler(svc re.Service, authn smqauthn.AuthNMiddleware, mux *chi.Mux, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Group(func(r chi.Router) {
		r.Use(authn.WithOptions(smqauthn.WithDomainCheck(true)).Middleware())
		r.Route("/{domainID}", func(r chi.Router) {
			r.Route("/rules", func(r chi.Router) {
				d := roleManagerHttp.NewDecoder("ruleID")

				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addRuleEndpoint(svc),
					decodeAddRuleRequest,
					api.EncodeResponse,
					opts...,
				), "create_rule").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listRulesEndpoint(svc),
					decodeListRulesRequest,
					api.EncodeResponse,
					opts...,
				), "list_rules").ServeHTTP)

				r = roleManagerHttp.EntityAvailableActionsRouter(svc, d, r, opts)

				r.Route("/{ruleID}", func(r chi.Router) {
					r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
						viewRuleEndpoint(svc),
						decodeViewRuleRequest,
						api.EncodeResponse,
						opts...,
					), "view_rule").ServeHTTP)

					r.Patch("/", otelhttp.NewHandler(kithttp.NewServer(
						updateRuleEndpoint(svc),
						decodeUpdateRuleRequest,
						api.EncodeResponse,
						opts...,
					), "update_rule").ServeHTTP)

					r.Patch("/tags", otelhttp.NewHandler(kithttp.NewServer(
						updateRuleTagsEndpoint(svc),
						decodeUpdateRuleTags,
						api.EncodeResponse,
						opts...,
					), "update_rule_tags").ServeHTTP)

					r.Patch("/schedule", otelhttp.NewHandler(kithttp.NewServer(
						updateRuleScheduleEndpoint(svc),
						decodeUpdateRuleScheduleRequest,
						api.EncodeResponse,
						opts...,
					), "update_rule_scheduler").ServeHTTP)

					r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
						deleteRuleEndpoint(svc),
						decodeDeleteRuleRequest,
						api.EncodeResponse,
						opts...,
					), "delete_rule").ServeHTTP)

					r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
						enableRuleEndpoint(svc),
						decodeUpdateRuleStatusRequest,
						api.EncodeResponse,
						opts...,
					), "enable_rule").ServeHTTP)

					r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
						disableRuleEndpoint(svc),
						decodeUpdateRuleStatusRequest,
						api.EncodeResponse,
						opts...,
					), "disable_rule").ServeHTTP)

					roleManagerHttp.EntityRoleMangerRouter(svc, d, r, opts)
				})
			})
		})
	})

	mux.Get("/health", supermq.Health("rule_engine", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeAddRuleRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return addRuleReq{Rule: rule}, nil
}

func decodeViewRuleRequest(_ context.Context, r *http.Request) (any, error) {
	id := chi.URLParam(r, ruleIdKey)
	return viewRuleReq{id: id}, nil
}

func decodeUpdateRuleRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	rule.ID = chi.URLParam(r, ruleIdKey)

	return updateRuleReq{Rule: rule}, nil
}

func decodeUpdateRuleTags(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateRuleTagsReq{
		id: chi.URLParam(r, ruleIdKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateRuleScheduleRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateRuleScheduleReq{
		id: chi.URLParam(r, ruleIdKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateRuleStatusRequest(_ context.Context, r *http.Request) (any, error) {
	req := updateRuleStatusReq{
		id: chi.URLParam(r, ruleIdKey),
	}

	return req, nil
}

func decodeListRulesRequest(_ context.Context, r *http.Request) (any, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	ic, err := apiutil.ReadStringQuery(r, inputChannelKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefStatus)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, "desc")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	order, err := apiutil.ReadStringQuery(r, api.OrderKey, api.DefOrder)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := re.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	tag, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	return listRulesReq{
		PageMeta: re.PageMeta{
			Offset:       offset,
			Limit:        limit,
			Name:         name,
			InputChannel: ic,
			Status:       st,
			Dir:          dir,
			Order:        order,
			Tag:          tag,
		},
	}, nil
}

func decodeDeleteRuleRequest(_ context.Context, r *http.Request) (any, error) {
	id := chi.URLParam(r, ruleIdKey)

	return deleteRuleReq{id: id}, nil
}
