// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"
	"net/http"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	roleManagerHttp "github.com/absmach/supermq/pkg/roles/rolemanager/api"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for Domains and Invitations API endpoints.
func MakeHandler(svc domains.Service, authn authn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string, idp supermq.IDProvider) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	d := roleManagerHttp.NewDecoder("domainID")
	mux.Route("/domains", func(r chi.Router) {
		r.Use(api.RequestIDMiddleware(idp))

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

		r.Route("/{domainID}/invitations", func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, true))
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				sendInvitationEndpoint(svc),
				decodeSendInvitationReq,
				api.EncodeResponse,
				opts...,
			), "send_invitation").ServeHTTP)
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listDomainInvitationsEndpoint(svc),
				decodeListInvitationsReq,
				api.EncodeResponse,
				opts...,
			), "list_domain_invitations").ServeHTTP)
			r.Route("/{userID}", func(r chi.Router) {
				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					viewInvitationEndpoint(svc),
					decodeInvitationReq,
					api.EncodeResponse,
					opts...,
				), "view_invitation").ServeHTTP)
				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					deleteInvitationEndpoint(svc),
					decodeInvitationReq,
					api.EncodeResponse,
					opts...,
				), "delete_invitation").ServeHTTP)
			})
		})
	})

	mux.Route("/invitations", func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, false))
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listUserInvitationsEndpoint(svc),
			decodeListInvitationsReq,
			api.EncodeResponse,
			opts...,
		), "list_user_invitations").ServeHTTP)
		r.Post("/accept", otelhttp.NewHandler(kithttp.NewServer(
			acceptInvitationEndpoint(svc),
			decodeAcceptInvitationReq,
			api.EncodeResponse,
			opts...,
		), "accept_invitation").ServeHTTP)
		r.Post("/reject", otelhttp.NewHandler(kithttp.NewServer(
			rejectInvitationEndpoint(svc),
			decodeAcceptInvitationReq,
			api.EncodeResponse,
			opts...,
		), "reject_invitation").ServeHTTP)
	})

	mux.Get("/health", supermq.Health("domains", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
