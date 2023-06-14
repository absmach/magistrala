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
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things/groups"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc groups.Service, mux *bone.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Post("/channels", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("create_channel"))(createGroupEndpoint(svc)),
		decodeGroupCreate,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/channels/bulk", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("create_channels"))(createGroupsEndpoint(svc)),
		decodeGroupsCreate,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/channels/:chanID", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("view_channel"))(viewGroupEndpoint(svc)),
		decodeGroupRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Put("/channels/:chanID", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_channel"))(updateGroupEndpoint(svc)),
		decodeGroupUpdate,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/things/:thingID/channels", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_channels_by_thing"))(listMembershipsEndpoint(svc)),
		decodeListMembershipRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/channels", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_channels"))(listGroupsEndpoint(svc)),
		decodeListGroupsRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/channels/:chanID/enable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("enable_channel"))(enableGroupEndpoint(svc)),
		decodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/channels/:chanID/disable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("disable_channel"))(disableGroupEndpoint(svc)),
		decodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	))
	return mux
}

func decodeListMembershipRequest(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return nil, err
	}
	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, err
	}
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, err
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, err
	}
	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
	if err != nil {
		return nil, err
	}
	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, err
	}
	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, err
	}
	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
	if err != nil {
		return nil, err
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return nil, err
	}
	req := listMembershipReq{
		token:    apiutil.ExtractBearerToken(r),
		clientID: bone.GetValue(r, "thingID"),
		GroupsPage: mfgroups.GroupsPage{
			Level: level,
			ID:    parentID,
			Page: mfgroups.Page{
				Offset:   offset,
				Limit:    limit,
				OwnerID:  ownerID,
				Name:     name,
				Metadata: meta,
				Status:   st,
			},
			Direction: dir,
		},
	}
	return req, nil

}

func decodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return nil, err
	}
	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, err
	}
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, err
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, err
	}
	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
	if err != nil {
		return nil, err
	}
	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, err
	}
	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, err
	}
	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
	if err != nil {
		return nil, err
	}
	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
	if err != nil {
		return nil, err
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return nil, err
	}
	req := listGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		tree:  tree,
		GroupsPage: mfgroups.GroupsPage{
			Level: level,
			ID:    parentID,
			Page: mfgroups.Page{
				Offset:   offset,
				Limit:    limit,
				OwnerID:  ownerID,
				Name:     name,
				Metadata: meta,
				Status:   st,
			},
			Direction: dir,
		},
	}
	return req, nil
}

func decodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	var g mfgroups.Group
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	req := createGroupReq{
		Group: g,
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeGroupsCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := createGroupsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req.Groups); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateGroupReq{
		id:    bone.GetValue(r, "chanID"),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := groupReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "chanID"),
	}
	return req, nil
}

func decodeChangeGroupStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeGroupStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "chanID"),
	}

	return req, nil
}
