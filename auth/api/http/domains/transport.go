// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"log/slog"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc auth.Service, mux *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux.Route("/domains", func(r chi.Router) {
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

		r.Route("/{domainID}", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				retrieveDomainEndpoint(svc),
				decodeRetrieveDomainRequest,
				api.EncodeResponse,
				opts...,
			), "view_domain").ServeHTTP)

			r.Get("/permissions", otelhttp.NewHandler(kithttp.NewServer(
				retrieveDomainPermissionsEndpoint(svc),
				decodeRetrieveDomainPermissionsRequest,
				api.EncodeResponse,
				opts...,
			), "view_domain_permissions").ServeHTTP)

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

			r.Route("/users", func(r chi.Router) {
				r.Post("/assign", otelhttp.NewHandler(kithttp.NewServer(
					assignDomainUsersEndpoint(svc),
					decodeAssignUsersRequest,
					api.EncodeResponse,
					opts...,
				), "assign_domain_users").ServeHTTP)

				r.Post("/unassign", otelhttp.NewHandler(kithttp.NewServer(
					unassignDomainUsersEndpoint(svc),
					decodeUnassignUsersRequest,
					api.EncodeResponse,
					opts...,
				), "unassign_domain_users").ServeHTTP)
			})
		})
	})

	return mux
}
