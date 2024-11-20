// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	roleManagerHttp "github.com/absmach/magistrala/pkg/roles/rolemanager/api"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc domains.Service, authn authn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	d := roleManagerHttp.NewDecoder("domainID")
	mux.Route("/domains", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, false))
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				createDomainEndpoint(svc),
				decodeCreateDomainRequest,
				api.EncodeResponse,
				opts...,
			), "create_domain").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listDomainsEndpoint(svc),
				decodeListDomainRequest,
				api.EncodeResponse,
				opts...,
			), "list_domains").ServeHTTP)

			roleManagerHttp.EntityAvailableActionsRouter(svc, d, r, opts)
		})

		r.Route("/{domainID}", func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, true))
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				retrieveDomainEndpoint(svc),
				decodeRetrieveDomainRequest,
				api.EncodeResponse,
				opts...,
			), "view_domain").ServeHTTP)

			r.Patch("/", otelhttp.NewHandler(kithttp.NewServer(
				updateDomainEndpoint(svc),
				decodeUpdateDomainRequest,
				api.EncodeResponse,
				opts...,
			), "update_domain").ServeHTTP)

			r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
				enableDomainEndpoint(svc),
				decodeEnableDomainRequest,
				api.EncodeResponse,
				opts...,
			), "enable_domain").ServeHTTP)

			r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
				disableDomainEndpoint(svc),
				decodeDisableDomainRequest,
				api.EncodeResponse,
				opts...,
			), "disable_domain").ServeHTTP)

			r.Post("/freeze", otelhttp.NewHandler(kithttp.NewServer(
				freezeDomainEndpoint(svc),
				decodeFreezeDomainRequest,
				api.EncodeResponse,
				opts...,
			), "freeze_domain").ServeHTTP)
			roleManagerHttp.EntityRoleMangerRouter(svc, d, r, opts)
		})
	})

	mux.Get("/health", magistrala.Health("auth", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
