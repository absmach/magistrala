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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc clients.Service, mux *bone.Mux, logger mflog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	mux.Post("/things", otelhttp.NewHandler(kithttp.NewServer(
		createClientEndpoint(svc),
		decodeCreateClientReq,
		api.EncodeResponse,
		opts...,
	), "create_thing"))

	mux.Post("/things/bulk", otelhttp.NewHandler(kithttp.NewServer(
		createClientsEndpoint(svc),
		decodeCreateClientsReq,
		api.EncodeResponse,
		opts...,
	), "create_things"))

	mux.Get("/things/:thingID", otelhttp.NewHandler(kithttp.NewServer(
		viewClientEndpoint(svc),
		decodeViewClient,
		api.EncodeResponse,
		opts...,
	), "view_thing"))

	mux.Get("/things", otelhttp.NewHandler(kithttp.NewServer(
		listClientsEndpoint(svc),
		decodeListClients,
		api.EncodeResponse,
		opts...,
	), "list_things"))

	mux.Get("/channels/:chanID/things", otelhttp.NewHandler(kithttp.NewServer(
		listMembersEndpoint(svc),
		decodeListMembersRequest,
		api.EncodeResponse,
		opts...,
	), "list_things_by_channel"))

	mux.Patch("/things/:thingID", otelhttp.NewHandler(kithttp.NewServer(
		updateClientEndpoint(svc),
		decodeUpdateClient,
		api.EncodeResponse,
		opts...,
	), "update_thing"))

	mux.Patch("/things/:thingID/tags", otelhttp.NewHandler(kithttp.NewServer(
		updateClientTagsEndpoint(svc),
		decodeUpdateClientTags,
		api.EncodeResponse,
		opts...,
	), "update_thing_tags"))

	mux.Patch("/things/:thingID/secret", otelhttp.NewHandler(kithttp.NewServer(
		updateClientSecretEndpoint(svc),
		decodeUpdateClientCredentials,
		api.EncodeResponse,
		opts...,
	), "update_thing_credentials"))

	mux.Patch("/things/:thingID/owner", otelhttp.NewHandler(kithttp.NewServer(
		updateClientOwnerEndpoint(svc),
		decodeUpdateClientOwner,
		api.EncodeResponse,
		opts...,
	), "update_thing_owner"))

	mux.Post("/things/:thingID/enable", otelhttp.NewHandler(kithttp.NewServer(
		enableClientEndpoint(svc),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	), "enable_thing"))

	mux.Post("/things/:thingID/disable", otelhttp.NewHandler(kithttp.NewServer(
		disableClientEndpoint(svc),
		decodeChangeClientStatus,
		api.EncodeResponse,
		opts...,
	), "disable_thing"))

	mux.GetFunc("/health", mainflux.Health("things", instanceID))
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
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateClientTags(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientTagsReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateClientCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientCredentialsReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateClientOwner(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateClientOwnerReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeCreateClientReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var c mfclients.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	req := createClientReq{
		client: c,
		token:  apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeCreateClientsReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	c := createClientsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&c.Clients); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
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
		},
		groupID: bone.GetValue(r, "chanID"),
	}
	return req, nil
}
