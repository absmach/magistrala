// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/api"
	gapi "github.com/absmach/magistrala/internal/groups/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func groupsHandler(svc groups.Service, r *chi.Mux, logger *slog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	r.Route("/channels", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.CreateGroupEndpoint(svc, auth.NewChannelKind),
			gapi.DecodeGroupCreate,
			api.EncodeResponse,
			opts...,
		), "create_channel").ServeHTTP)

		r.Get("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ViewGroupEndpoint(svc),
			gapi.DecodeGroupRequest,
			api.EncodeResponse,
			opts...,
		), "view_channel").ServeHTTP)

		r.Delete("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.DeleteGroupEndpoint(svc),
			gapi.DecodeGroupRequest,
			api.EncodeResponse,
			opts...,
		), "delete_channel").ServeHTTP)

		r.Get("/{groupID}/permissions", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ViewGroupPermsEndpoint(svc),
			gapi.DecodeGroupPermsRequest,
			api.EncodeResponse,
			opts...,
		), "view_channel_permissions").ServeHTTP)

		r.Put("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.UpdateGroupEndpoint(svc),
			gapi.DecodeGroupUpdate,
			api.EncodeResponse,
			opts...,
		), "update_channel").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListGroupsEndpoint(svc, "channels", "users"),
			gapi.DecodeListGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "list_channels").ServeHTTP)

		r.Post("/{groupID}/enable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.EnableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "enable_channel").ServeHTTP)

		r.Post("/{groupID}/disable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.DisableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "disable_channel").ServeHTTP)

		// Request to add users to a channel
		// This endpoint can be used alternative to /channels/{groupID}/members
		r.Post("/{groupID}/users/assign", otelhttp.NewHandler(kithttp.NewServer(
			assignUsersEndpoint(svc),
			decodeAssignUsersRequest,
			api.EncodeResponse,
			opts...,
		), "assign_users").ServeHTTP)

		// Request to remove users from a channel
		// This endpoint can be used alternative to /channels/{groupID}/members
		r.Post("/{groupID}/users/unassign", otelhttp.NewHandler(kithttp.NewServer(
			unassignUsersEndpoint(svc),
			decodeUnassignUsersRequest,
			api.EncodeResponse,
			opts...,
		), "unassign_users").ServeHTTP)

		// Request to add user_groups to a channel
		// This endpoint can be used alternative to /channels/{groupID}/members
		r.Post("/{groupID}/groups/assign", otelhttp.NewHandler(kithttp.NewServer(
			assignUserGroupsEndpoint(svc),
			decodeAssignUserGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "assign_groups").ServeHTTP)

		// Request to remove user_groups from a channel
		// This endpoint can be used alternative to /channels/{groupID}/members
		r.Post("/{groupID}/groups/unassign", otelhttp.NewHandler(kithttp.NewServer(
			unassignUserGroupsEndpoint(svc),
			decodeUnassignUserGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "unassign_groups").ServeHTTP)

		r.Post("/{groupID}/things/{thingID}/connect", otelhttp.NewHandler(kithttp.NewServer(
			connectChannelThingEndpoint(svc),
			decodeConnectChannelThingRequest,
			api.EncodeResponse,
			opts...,
		), "connect_channel_thing").ServeHTTP)

		r.Post("/{groupID}/things/{thingID}/disconnect", otelhttp.NewHandler(kithttp.NewServer(
			disconnectChannelThingEndpoint(svc),
			decodeDisconnectChannelThingRequest,
			api.EncodeResponse,
			opts...,
		), "disconnect_channel_thing").ServeHTTP)
	})

	// Connect channel and thing
	r.Post("/connect", otelhttp.NewHandler(kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnectRequest,
		api.EncodeResponse,
		opts...,
	), "connect").ServeHTTP)

	// Disconnect channel and thing
	r.Post("/disconnect", otelhttp.NewHandler(kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeDisconnectRequest,
		api.EncodeResponse,
		opts...,
	), "disconnect").ServeHTTP)

	return r
}

func decodeAssignUsersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := assignUsersRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUnassignUsersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := unassignUsersRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeAssignUserGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := assignUserGroupsRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUnassignUserGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := unassignUserGroupsRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeConnectChannelThingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := connectChannelThingRequest{
		token:     apiutil.ExtractBearerToken(r),
		ThingID:   chi.URLParam(r, "thingID"),
		ChannelID: chi.URLParam(r, "groupID"),
	}

	return req, nil
}

func decodeDisconnectChannelThingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := disconnectChannelThingRequest{
		token:     apiutil.ExtractBearerToken(r),
		ThingID:   chi.URLParam(r, "thingID"),
		ChannelID: chi.URLParam(r, "groupID"),
	}

	return req, nil
}

func decodeConnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := connectChannelThingRequest{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := disconnectChannelThingRequest{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}
