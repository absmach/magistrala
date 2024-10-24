// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func EntityRoleMangerRouter(svc roles.RoleManager, d Decoder, r chi.Router, opts []kithttp.ServerOption) chi.Router {
	r.Route("/roles", func(r chi.Router) {

		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			CreateRoleEndpoint(svc),
			d.DecodeCreateRole,
			api.EncodeResponse,
			opts...,
		), "create_role").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			ListRolesEndpoint(svc),
			d.DecodeListRoles,
			api.EncodeResponse,
			opts...,
		), "list_roles").ServeHTTP)

		r.Route("/{roleName}", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				ViewRoleEndpoint(svc),
				d.DecodeViewRole,
				api.EncodeResponse,
				opts...,
			), "view_role").ServeHTTP)

			r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
				UpdateRoleEndpoint(svc),
				d.DecodeUpdateRole,
				api.EncodeResponse,
				opts...,
			), "update_role").ServeHTTP)

			r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
				DeleteRoleEndpoint(svc),
				d.DecodeDeleteRole,
				api.EncodeResponse,
				opts...,
			), "delete_role").ServeHTTP)

			r.Route("/actions", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					AddRoleActionsEndpoint(svc),
					d.DecodeAddRoleActions,
					api.EncodeResponse,
					opts...,
				), "add_role_actions").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					ListRoleActionsEndpoint(svc),
					d.DecodeListRoleActions,
					api.EncodeResponse,
					opts...,
				), "list_role_actions").ServeHTTP)

				r.Post("/delete", otelhttp.NewHandler(kithttp.NewServer(
					DeleteRoleActionsEndpoint(svc),
					d.DecodeDeleteRoleActions,
					api.EncodeResponse,
					opts...,
				), "delete_role_actions").ServeHTTP)

				r.Post("/delete-all", otelhttp.NewHandler(kithttp.NewServer(
					DeleteAllRoleActionsEndpoint(svc),
					d.DecodeDeleteAllRoleActions,
					api.EncodeResponse,
					opts...,
				), "delete_all_role_actions").ServeHTTP)
			})

			r.Route("/members", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					AddRoleMembersEndpoint(svc),
					d.DecodeAddRoleMembers,
					api.EncodeResponse,
					opts...,
				), "add_role_members").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					ListRoleMembersEndpoint(svc),
					d.DecodeListRoleMembers,
					api.EncodeResponse,
					opts...,
				), "list_role_members").ServeHTTP)

				r.Post("/delete", otelhttp.NewHandler(kithttp.NewServer(
					DeleteRoleMembersEndpoint(svc),
					d.DecodeDeleteRoleMembers,
					api.EncodeResponse,
					opts...,
				), "delete_role_members").ServeHTTP)

				r.Post("/delete-all", otelhttp.NewHandler(kithttp.NewServer(
					DeleteAllRoleMembersEndpoint(svc),
					d.DecodeDeleteAllRoleMembers,
					api.EncodeResponse,
					opts...,
				), "delete_all_role_members").ServeHTTP)
			})
		})

	})

	return r
}

func EntityAvailableActionsRouter(svc roles.RoleManager, d Decoder, r chi.Router, opts []kithttp.ServerOption) chi.Router {
	r.Get("/roles/available-actions", otelhttp.NewHandler(kithttp.NewServer(
		ListAvailableActionsEndpoint(svc),
		d.DecodeListAvailableActions,
		api.EncodeResponse,
		opts...,
	), "list_available_actions").ServeHTTP)

	return r
}
