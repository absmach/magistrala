// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/logger"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc auth.Service, r *bone.Mux, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	dr := bone.New()

	dr.Post("", otelhttp.NewHandler(kithttp.NewServer(
		createDomainEndpoint(svc),
		decodeCreateDomainRequest,
		api.EncodeResponse,
		opts...,
	), "create_domain"))

	dr.Get("/:domainID", otelhttp.NewHandler(kithttp.NewServer(
		retrieveDomainEndpoint(svc),
		decodeRetrieveDomainRequest,
		api.EncodeResponse,
		opts...,
	), "view_domain"))

	dr.Patch("/:domainID", otelhttp.NewHandler(kithttp.NewServer(
		updateDomainEndpoint(svc),
		decodeUpdateDomainRequest,
		api.EncodeResponse,
		opts...,
	), "update_domain"))

	dr.Get("", otelhttp.NewHandler(kithttp.NewServer(
		listDomainsEndpoint(svc),
		decodeListDomainRequest,
		api.EncodeResponse,
		opts...,
	), "list_domains"))

	dr.Post("/:domainID/enable", otelhttp.NewHandler(kithttp.NewServer(
		enableDomainEndpoint(svc),
		decodeEnableDomainRequest,
		api.EncodeResponse,
		opts...,
	), "enable_domain"))

	dr.Post("/:domainID/disable", otelhttp.NewHandler(kithttp.NewServer(
		disableDomainEndpoint(svc),
		decodeDisableDomainRequest,
		api.EncodeResponse,
		opts...,
	), "disable_domain"))

	dr.Post("/:domainID/users/assign", otelhttp.NewHandler(kithttp.NewServer(
		assignDomainUsersEndpoint(svc),
		decodeAssignUsersRequest,
		api.EncodeResponse,
		opts...,
	), "assign_domain_users"))

	dr.Post("/:domainID/users/unassign", otelhttp.NewHandler(kithttp.NewServer(
		unassignDomainUsersEndpoint(svc),
		decodeUnassignUsersRequest,
		api.EncodeResponse,
		opts...,
	), "unassign_domain_users"))

	r.SubRoute("/domains", dr)

	r.Get("/users/:userID/domains", otelhttp.NewHandler(kithttp.NewServer(
		listUserDomainsEndpoint(svc),
		decodeListUserDomainsRequest,
		api.EncodeResponse,
		opts...,
	), "list_domains_by_user_id"))

	return r
}
