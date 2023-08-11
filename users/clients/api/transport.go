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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc clients.Service, mux *bone.Mux, logger mflog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux.Post("/users", otelhttp.NewHandler(kithttp.NewServer(
		registrationEndpoint(svc),
		decodeCreateClientReq,
		api.EncodeResponse,
		opts...,
	), "register_client"))

	mux.Get("/users/profile", otelhttp.NewHandler(kithttp.NewServer(
		viewProfileEndpoint(svc),
		decodeViewProfile,
		api.EncodeResponse,
		opts...,
	), "view_profile"))

	mux.Get("/users/:id", otelhttp.NewHandler(kithttp.NewServer(
		viewClientEndpoint(svc),
		decodeViewClient,
		api.EncodeResponse,
		opts...,
	), "view_client"))

	mux.Get("/users", otelhttp.NewHandler(kithttp.NewServer(
		listClientsEndpoint(svc),
		decodeListClients,
		api.EncodeResponse,
		opts...,
	), "list_clients"))

	mux.Get("/groups/:groupID/members", otelhttp.NewHandler(kithttp.NewServer(
		listMembersEndpoint(svc),
		decodeListMembersRequest,
		api.EncodeResponse,
		opts...,
	), "list_members"))

	mux.Patch("/users/secret", otelhttp.NewHandler(kithttp.NewServer(
		updateClientSecretEndpoint(svc),
		decodeUpdateClientSecret,
		api.EncodeResponse,
		opts...,
	), "update_client_secret"))

	mux.Patch("/users/:id", otelhttp.NewHandler(kithttp.NewServer(
		updateClientEndpoint(svc),
		decodeUpdateClient,
		api.EncodeResponse,
		opts...,
	), "update_client"))

	mux.Patch("/users/:id/tags", otelhttp.NewHandler(kithttp.NewServer(
		updateClientTagsEndpoint(svc),
		decodeUpdateClientTags,
		api.EncodeResponse,
		opts...,
	), "update_client_tags"))

	mux.Patch("/users/:id/identity", otelhttp.NewHandler(kithttp.NewServer(
		updateClientIdentityEndpoint(svc),
		decodeUpdateClientIdentity,
		api.EncodeResponse,
		opts...,
	), "update_client_identity"))

	mux.Post("/password/reset-request", otelhttp.NewHandler(kithttp.NewServer(
		passwordResetRequestEndpoint(svc),
		decodePasswordResetRequest,
		api.EncodeResponse,
		opts...,
	), "password_reset_req"))

	mux.Put("/password/reset", otelhttp.NewHandler(kithttp.NewServer(
		passwordResetEndpoint(svc),
		decodePasswordReset,
		api.EncodeResponse,
		opts...,
	), "password_reset"))

	mux.Patch("/users/:id/owner", otelhttp.NewHandler(kithttp.NewServer(
		updateClientOwnerEndpoint(svc),
		decodeUpdateClientOwner,
		api.EncodeResponse,
		opts...,
	), "update_client_owner"))

	mux.Post("/users/tokens/issue", otelhttp.NewHandler(kithttp.NewServer(
		issueTokenEndpoint(svc),
		decodeCredentials,
		api.EncodeResponse,
		opts...,
	), "issue_token"))

	mux.Post("/users/tokens/refresh", otelhttp.NewHandler(kithttp.NewServer(
		refreshTokenEndpoint(svc),
		decodeRefreshToken,
		api.EncodeResponse,
		opts...,
	), "refresh_token"))

	mux.Post("/users/:id/enable", otelhttp.NewHandler(kithttp.NewServer(
		enableClientEndpoint(svc),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	), "enable_client"))

	mux.Post("/users/:id/disable", otelhttp.NewHandler(kithttp.NewServer(
		disableClientEndpoint(svc),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	), "disable_client"))

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
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	n, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	i, err := apiutil.ReadStringQuery(r, api.IdentityKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}
	visibility, err := apiutil.ReadStringQuery(r, api.VisibilityKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
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
		return nil, errors.Wrap(apiutil.ErrValidation, err)
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
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateClientTags(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientTagsReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateClientIdentity(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientIdentityReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateClientSecret(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientSecretReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	req.Host = r.Header.Get("Referer")
	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateClientOwner(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientOwnerReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := loginClientReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeRefreshToken(_ context.Context, r *http.Request) (interface{}, error) {
	req := tokenReq{RefreshToken: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeCreateClientReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var c mfclients.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
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
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	n, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	i, err := apiutil.ReadStringQuery(r, api.IdentityKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
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
		groupID: bone.GetValue(r, "groupID"),
	}
	return req, nil
}
