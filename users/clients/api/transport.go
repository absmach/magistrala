// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc clients.Service, mux *bone.Mux, logger mflog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux.Post("/users", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("register_client"))(registrationEndpoint(svc)),
		decodeCreateClientReq,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/users/profile", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("view_profile"))(viewProfileEndpoint(svc)),
		decodeViewProfile,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/users/:id", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("view_client"))(viewClientEndpoint(svc)),
		decodeViewClient,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/users", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_clients"))(listClientsEndpoint(svc)),
		decodeListClients,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/members", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_members"))(listMembersEndpoint(svc)),
		decodeListMembersRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/users/secret", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_client_secret"))(updateClientSecretEndpoint(svc)),
		decodeUpdateClientSecret,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/users/:id", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_client_name_and_metadata"))(updateClientEndpoint(svc)),
		decodeUpdateClient,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/users/:id/tags", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_client_tags"))(updateClientTagsEndpoint(svc)),
		decodeUpdateClientTags,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/users/:id/identity", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_client_identity"))(updateClientIdentityEndpoint(svc)),
		decodeUpdateClientIdentity,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/password/reset-request", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("password_reset_req"))(passwordResetRequestEndpoint(svc)),
		decodePasswordResetRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Put("/password/reset", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("password_reset"))(passwordResetEndpoint(svc)),
		decodePasswordReset,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/users/:id/owner", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_client_owner"))(updateClientOwnerEndpoint(svc)),
		decodeUpdateClientOwner,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/users/tokens/issue", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("issue_token"))(issueTokenEndpoint(svc)),
		decodeCredentials,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/users/tokens/refresh", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("refresh_token"))(refreshTokenEndpoint(svc)),
		decodeRefreshToken,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/users/:id/enable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("enable_client"))(enableClientEndpoint(svc)),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/users/:id/disable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("disable_client"))(disableClientEndpoint(svc)),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("users", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeViewClient(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeViewProfile(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewProfileReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeListClients(_ context.Context, r *http.Request) (interface{}, error) {
	var sharedID, ownerID string
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefClientStatus)
	if err != nil {
		return nil, err
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, err
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, err
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, err
	}
	i, err := apiutil.ReadStringQuery(r, api.IdentityKey, "")
	if err != nil {
		return nil, err
	}
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, err
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}
	visibility, err := apiutil.ReadStringQuery(r, api.VisibilityKey, "")
	if err != nil {
		return nil, err
	}
	switch visibility {
	case api.MyVisibility:
		ownerID = api.MyVisibility
	case api.SharedVisibility:
		sharedID = api.MyVisibility
	case api.AllVisibility:
		sharedID = api.MyVisibility
		ownerID = api.MyVisibility
	}
	if oid != "" {
		ownerID = oid
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return nil, err
	}
	req := listClientsReq{
		token:    apiutil.ExtractBearerToken(r),
		status:   st,
		offset:   o,
		limit:    l,
		metadata: m,
		name:     n,
		identity: i,
		tag:      t,
		sharedBy: sharedID,
		owner:    ownerID,
	}
	return req, nil
}

func decodeUpdateClient(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateClientTags(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientTagsReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateClientIdentity(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientIdentityReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateClientSecret(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientSecretReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.Host = r.Header.Get("Referer")
	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateClientOwner(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientOwnerReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := loginClientReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRefreshToken(_ context.Context, r *http.Request) (interface{}, error) {
	req := tokenReq{RefreshToken: apiutil.ExtractBearerToken(r)}

	return req, nil
}
func decodeCreateClientReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var c mfclients.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	req := createClientReq{
		client: c,
		token:  apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeChangeClientStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeClientStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeListMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefClientStatus)
	if err != nil {
		return nil, err
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, err
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, err
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, err
	}
	n, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, err
	}
	i, err := apiutil.ReadStringQuery(r, api.IdentityKey, "")
	if err != nil {
		return nil, err
	}
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, err
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return nil, err
	}
	req := listMembersReq{
		token: apiutil.ExtractBearerToken(r),
		Page: mfclients.Page{
			Status:   st,
			Offset:   o,
			Limit:    l,
			Metadata: m,
			Identity: i,
			Name:     n,
			Tag:      t,
			Owner:    oid,
		},
		groupID: bone.GetValue(r, "id"),
	}
	return req, nil
}
