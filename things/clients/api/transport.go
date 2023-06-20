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
	"github.com/mainflux/mainflux/things/clients"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc clients.Service, mux *bone.Mux, logger mflog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux.Post("/things", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("create_thing"))(createClientEndpoint(svc)),
		decodeCreateClientReq,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/things/bulk", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("create_things"))(createClientsEndpoint(svc)),
		decodeCreateClientsReq,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/things/:thingID", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("view_thing"))(viewClientEndpoint(svc)),
		decodeViewClient,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/things", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_things"))(listClientsEndpoint(svc)),
		decodeListClients,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/channels/:thingID/things", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_things_by_channel"))(listMembersEndpoint(svc)),
		decodeListMembersRequest,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/things/:thingID", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_thing_name_and_metadata"))(updateClientEndpoint(svc)),
		decodeUpdateClient,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/things/:thingID/tags", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_thing_tags"))(updateClientTagsEndpoint(svc)),
		decodeUpdateClientTags,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/things/:thingID/secret", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_thing_secret"))(updateClientSecretEndpoint(svc)),
		decodeUpdateClientCredentials,
		api.EncodeResponse,
		opts...,
	))

	mux.Patch("/things/:thingID/owner", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_thing_owner"))(updateClientOwnerEndpoint(svc)),
		decodeUpdateClientOwner,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/things/:thingID/enable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("enable_thing"))(enableClientEndpoint(svc)),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/things/:thingID/disable", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("disable_thing"))(disableClientEndpoint(svc)),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("things"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func decodeViewClient(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
	}

	return req, nil
}

func decodeListClients(_ context.Context, r *http.Request) (interface{}, error) {
	var sid, oid string
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
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, err
	}
	visibility, err := apiutil.ReadStringQuery(r, api.VisibilityKey, api.MyVisibility)
	if err != nil {
		return nil, err
	}
	switch visibility {
	case api.MyVisibility:
		oid = api.MyVisibility
	case api.SharedVisibility:
		sid = api.MyVisibility
	case api.AllVisibility:
		sid = api.MyVisibility
		oid = api.MyVisibility
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
		tag:      t,
		sharedBy: sid,
		owner:    oid,
	}
	return req, nil
}

func decodeUpdateClient(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
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
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateClientCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := updateClientCredentialsReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
	}
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
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

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

func decodeCreateClientsReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	c := createClientsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&c.Clients); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return c, nil
}

func decodeChangeClientStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeClientStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
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
		},
		groupID: bone.GetValue(r, "thingID"),
	}
	return req, nil
}
