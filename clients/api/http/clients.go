// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	roleManagerHttp "github.com/absmach/magistrala/pkg/roles/rolemanager/api"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func clientsHandler(svc clients.Service, authn mgauthn.Authentication, r *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	d := roleManagerHttp.NewDecoder("clientID")

	r.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))

		r.Route("/{domainID}/clients", func(r chi.Router) {
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				createClientEndpoint(svc),
				decodeCreateClientReq,
				api.EncodeResponse,
				opts...,
			), "create_client").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listClientsEndpoint(svc),
				decodeListClients,
				api.EncodeResponse,
				opts...,
			), "list_clients").ServeHTTP)

			r.Post("/bulk", otelhttp.NewHandler(kithttp.NewServer(
				createClientsEndpoint(svc),
				decodeCreateClientsReq,
				api.EncodeResponse,
				opts...,
			), "create_clients").ServeHTTP)
			r = roleManagerHttp.EntityAvailableActionsRouter(svc, d, r, opts)

			r.Route("/{clientID}", func(r chi.Router) {
				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					viewClientEndpoint(svc),
					decodeViewClient,
					api.EncodeResponse,
					opts...,
				), "view_client").ServeHTTP)

				r.Patch("/", otelhttp.NewHandler(kithttp.NewServer(
					updateClientEndpoint(svc),
					decodeUpdateClient,
					api.EncodeResponse,
					opts...,
				), "update_client").ServeHTTP)

				r.Patch("/tags", otelhttp.NewHandler(kithttp.NewServer(
					updateClientTagsEndpoint(svc),
					decodeUpdateClientTags,
					api.EncodeResponse,
					opts...,
				), "update_client_tags").ServeHTTP)

				r.Patch("/secret", otelhttp.NewHandler(kithttp.NewServer(
					updateClientSecretEndpoint(svc),
					decodeUpdateClientCredentials,
					api.EncodeResponse,
					opts...,
				), "update_client_credentials").ServeHTTP)

				r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
					enableClientEndpoint(svc),
					decodeChangeClientStatus,
					api.EncodeResponse,
					opts...,
				), "enable_client").ServeHTTP)

				r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
					disableClientEndpoint(svc),
					decodeChangeClientStatus,
					api.EncodeResponse,
					opts...,
				), "disable_client").ServeHTTP)

				r.Post("/parent", otelhttp.NewHandler(kithttp.NewServer(
					setClientParentGroupEndpoint(svc),
					decodeSetClientParentGroupStatus,
					api.EncodeResponse,
					opts...,
				), "set_client_parent_group").ServeHTTP)

				r.Delete("/parent", otelhttp.NewHandler(kithttp.NewServer(
					removeClientParentGroupEndpoint(svc),
					decodeRemoveClientParentGroupStatus,
					api.EncodeResponse,
					opts...,
				), "remove_client_parent_group").ServeHTTP)

				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					deleteClientEndpoint(svc),
					decodeDeleteClientReq,
					api.EncodeResponse,
					opts...,
				), "delete_client").ServeHTTP)
				roleManagerHttp.EntityRoleMangerRouter(svc, d, r, opts)
			})
		})

		r.Get("/{domainID}/users/{userID}/clients", otelhttp.NewHandler(kithttp.NewServer(
			listClientsEndpoint(svc),
			decodeListClients,
			api.EncodeResponse,
			opts...,
		), "list_user_clients").ServeHTTP)
	})
	return r
}
