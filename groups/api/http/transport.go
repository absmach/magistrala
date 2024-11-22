// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"log/slog"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	roleManagerHttp "github.com/absmach/magistrala/pkg/roles/rolemanager/api"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for Groups API endpoints.
func MakeHandler(svc groups.Service, authn authn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	d := roleManagerHttp.NewDecoder("groupID")

	mux.Route("/{domainID}/groups", func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			CreateGroupEndpoint(svc),
			DecodeGroupCreate,
			api.EncodeResponse,
			opts...,
		), "create_group").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			ListGroupsEndpoint(svc),
			DecodeListGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "list_groups").ServeHTTP)
		r = roleManagerHttp.EntityAvailableActionsRouter(svc, d, r, opts)

		r.Route("/{groupID}", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				ViewGroupEndpoint(svc),
				DecodeGroupRequest,
				api.EncodeResponse,
				opts...,
			), "view_group").ServeHTTP)

			r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
				UpdateGroupEndpoint(svc),
				DecodeGroupUpdate,
				api.EncodeResponse,
				opts...,
			), "update_group").ServeHTTP)

			r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
				DeleteGroupEndpoint(svc),
				DecodeGroupRequest,
				api.EncodeResponse,
				opts...,
			), "delete_group").ServeHTTP)

			r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
				EnableGroupEndpoint(svc),
				DecodeChangeGroupStatusRequest,
				api.EncodeResponse,
				opts...,
			), "enable_group").ServeHTTP)

			r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
				DisableGroupEndpoint(svc),
				DecodeChangeGroupStatusRequest,
				api.EncodeResponse,
				opts...,
			), "disable_group").ServeHTTP)

			r = roleManagerHttp.EntityRoleMangerRouter(svc, d, r, opts)

			r.Get("/hierarchy", otelhttp.NewHandler(kithttp.NewServer(
				retrieveGroupHierarchyEndpoint(svc),
				decodeRetrieveGroupHierarchy,
				api.EncodeResponse,
				opts...,
			), "retrieve_group_hierarchy").ServeHTTP)

			r.Route("/parent", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addParentGroupEndpoint(svc),
					decodeAddParentGroupRequest,
					api.EncodeResponse,
					opts...,
				), "add_parent_group").ServeHTTP)

				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					removeParentGroupEndpoint(svc),
					decodeRemoveParentGroupRequest,
					api.EncodeResponse,
					opts...,
				), "remove_parent_group").ServeHTTP)
			})

			r.Route("/children", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addChildrenGroupsEndpoint(svc),
					decodeAddChildrenGroupsRequest,
					api.EncodeResponse,
					opts...,
				), "add_children_groups").ServeHTTP)

				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					removeChildrenGroupsEndpoint(svc),
					decodeRemoveChildrenGroupsRequest,
					api.EncodeResponse,
					opts...,
				), "remove_children_groups").ServeHTTP)

				r.Delete("/all", otelhttp.NewHandler(kithttp.NewServer(
					removeAllChildrenGroupsEndpoint(svc),
					decodeRemoveAllChildrenGroupsRequest,
					api.EncodeResponse,
					opts...,
				), "remove_all_children_groups").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listChildrenGroupsEndpoint(svc),
					decodeListChildrenGroupsRequest,
					api.EncodeResponse,
					opts...,
				), "list_children_groups").ServeHTTP)
			})
		})
	})

	return mux
}
