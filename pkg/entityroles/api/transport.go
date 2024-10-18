// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func RolesHandler(svc roles.Roles, authn mgauthn.Authentication, entityTypePrefixRootPath string, r *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	// RoleID - random string, So having roleName in URL make readable. But it have little overhead, it requires additional step to retrieve roleID in each service
	// http://localhost/things/thingID/roles/roleName

	r.Route(fmt.Sprintf("%s/{entityID}/roles", entityTypePrefixRootPath), func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn))

		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			CreateRoleEndpoint(svc),
			DecodeCreateRole,
			api.EncodeResponse,
			opts...,
		), "create_role").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			ListRolesEndpoint(svc),
			DecodeListRoles,
			api.EncodeResponse,
			opts...,
		), "list_roles").ServeHTTP)

		r.Route("/{roleName}", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				ViewRoleEndpoint(svc),
				DecodeViewRole,
				api.EncodeResponse,
				opts...,
			), "view_role").ServeHTTP)

			r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
				UpdateRoleEndpoint(svc),
				DecodeUpdateRole,
				api.EncodeResponse,
				opts...,
			), "update_role").ServeHTTP)

			r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
				DeleteRoleEndpoint(svc),
				DecodeDeleteRole,
				api.EncodeResponse,
				opts...,
			), "delete_role").ServeHTTP)

			r.Route("/actions", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					AddRoleActionsEndpoint(svc),
					DecodeAddRoleActions,
					api.EncodeResponse,
					opts...,
				), "add_role_actions").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					ListRoleActionsEndpoint(svc),
					DecodeListRoleActions,
					api.EncodeResponse,
					opts...,
				), "list_role_actions").ServeHTTP)

				r.Post("/delete", otelhttp.NewHandler(kithttp.NewServer(
					DeleteRoleActionsEndpoint(svc),
					DecodeDeleteRoleActions,
					api.EncodeResponse,
					opts...,
				), "delete_role_actions").ServeHTTP)

				r.Post("/delete-all", otelhttp.NewHandler(kithttp.NewServer(
					DeleteAllRoleActionsEndpoint(svc),
					DecodeDeleteAllRoleActions,
					api.EncodeResponse,
					opts...,
				), "delete_all_role_actions").ServeHTTP)
			})

			r.Route("/members", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					AddRoleMembersEndpoint(svc),
					DecodeAddRoleMembers,
					api.EncodeResponse,
					opts...,
				), "add_role_members").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					ListRoleMembersEndpoint(svc),
					DecodeListRoleMembers,
					api.EncodeResponse,
					opts...,
				), "list_role_members").ServeHTTP)

				r.Post("/delete", otelhttp.NewHandler(kithttp.NewServer(
					DeleteRoleMembersEndpoint(svc),
					DecodeDeleteRoleMembers,
					api.EncodeResponse,
					opts...,
				), "delete_role_members").ServeHTTP)

				r.Post("/delete-all", otelhttp.NewHandler(kithttp.NewServer(
					DeleteAllRoleMembersEndpoint(svc),
					DecodeDeleteAllRoleMembers,
					api.EncodeResponse,
					opts...,
				), "delete_all_role_members").ServeHTTP)
			})
		})

	})

	r.Get(fmt.Sprintf("%s/roles/available-actions", entityTypePrefixRootPath), otelhttp.NewHandler(kithttp.NewServer(
		ListAvailableActionsEndpoint(svc),
		DecodeListAvailableActions,
		api.EncodeResponse,
		opts...,
	), "list_available_actions").ServeHTTP)

	return r
}
