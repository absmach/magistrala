// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/groups"
	groupsAPI "github.com/mainflux/mainflux/internal/groups/api"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	nameKey     = "name"
	orderKey    = "order"
	dirKey      = "dir"
	metadataKey = "metadata"
	connKey     = "connected"

	defOffset = 0
	defLimit  = 10
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc things.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_thing")(createThingEndpoint(svc)),
		decodeThingCreation,
		encodeResponse,
		opts...,
	))

	r.Post("/things/bulk", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_things")(createThingsEndpoint(svc)),
		decodeThingsCreation,
		encodeResponse,
		opts...,
	))

	r.Patch("/things/:id/key", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_key")(updateKeyEndpoint(svc)),
		decodeKeyUpdate,
		encodeResponse,
		opts...,
	))

	r.Put("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_thing")(updateThingEndpoint(svc)),
		decodeThingUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_thing")(removeThingEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_thing")(viewThingEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_channels_by_thing")(listChannelsByThingEndpoint(svc)),
		decodeListByConnection,
		encodeResponse,
		opts...,
	))

	r.Get("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things")(listThingsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Post("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_channel")(createChannelEndpoint(svc)),
		decodeChannelCreation,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/bulk", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_channels")(createChannelsEndpoint(svc)),
		decodeChannelsCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_channel")(updateChannelEndpoint(svc)),
		decodeChannelUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_channel")(removeChannelEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_channel")(viewChannelEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_channel")(listThingsByChannelEndpoint(svc)),
		decodeListByConnection,
		encodeResponse,
		opts...,
	))

	r.Get("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_channels")(listChannelsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:chanId/things/:thingId", kithttp.NewServer(
		kitot.TraceServer(tracer, "connect")(connectEndpoint(svc)),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.Post("/connect", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_connections")(createConnectionsEndpoint(svc)),
		decodeCreateConnections,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:chanId/things/:thingId", kithttp.NewServer(
		kitot.TraceServer(tracer, "disconnect")(disconnectEndpoint(svc)),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:memberID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_memberships")(groupsAPI.ListMembership(svc)),
		groupsAPI.DecodeListMemberGroupRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "add_group")(groupsAPI.CreateGroupEndpoint(svc)),
		groupsAPI.DecodeGroupCreate,
		encodeResponse,
		opts...,
	))

	r.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(groupsAPI.ListGroupsEndpoint(svc)),
		groupsAPI.DecodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	r.Delete("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_group")(groupsAPI.DeleteGroupEndpoint(svc)),
		groupsAPI.DecodeGroupRequest,
		encodeResponse,
		opts...,
	))

	r.Put("/groups/:groupID/things/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign")(groupsAPI.AssignEndpoint(svc)),
		groupsAPI.DecodeMemberGroupRequest,
		encodeResponse,
		opts...,
	))

	r.Delete("/groups/:groupID/things/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign")(groupsAPI.UnassignEndpoint(svc)),
		groupsAPI.DecodeMemberGroupRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:groupID/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things")(groupsAPI.ListMembersEndpoint(svc)),
		groupsAPI.DecodeListMemberGroupRequest,
		encodeResponse,
		opts...,
	))

	r.Put("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(groupsAPI.UpdateGroupEndpoint(svc)),
		groupsAPI.DecodeGroupUpdate,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:groupID/children", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_children_groups")(groupsAPI.ListGroupChildrenEndpoint(svc)),
		groupsAPI.DecodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:groupID/parents", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_parent_groups")(groupsAPI.ListGroupParentsEndpoint(svc)),
		groupsAPI.DecodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group")(groupsAPI.ViewGroupEndpoint(svc)),
		groupsAPI.DecodeGroupRequest,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("things"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeThingCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createThingReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeThingsCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createThingsReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req.Things); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeThingUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := updateThingReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeKeyUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := updateKeyReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeChannelCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createChannelReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeChannelsCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createChannelsReq{token: r.Header.Get("Authorization")}

	if err := json.NewDecoder(r.Body).Decode(&req.Channels); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeChannelUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := updateChannelReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewResourceReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := readUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	n, err := readStringQuery(r, nameKey)
	if err != nil {
		return nil, err
	}

	or, err := readStringQuery(r, orderKey)
	if err != nil {
		return nil, err
	}

	d, err := readStringQuery(r, dirKey)
	if err != nil {
		return nil, err
	}

	m, err := readMetadataQuery(r, metadataKey)
	if err != nil {
		return nil, err
	}

	req := listResourcesReq{
		token: r.Header.Get("Authorization"),
		pageMetadata: things.PageMetadata{
			Offset:   o,
			Limit:    l,
			Name:     n,
			Order:    or,
			Dir:      d,
			Metadata: m,
		},
	}

	return req, nil
}

func decodeListByConnection(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := readUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	c, err := readBoolQuery(r, connKey)
	if err != nil {
		return nil, err
	}

	or, err := readStringQuery(r, orderKey)
	if err != nil {
		return nil, err
	}

	d, err := readStringQuery(r, dirKey)
	if err != nil {
		return nil, err
	}

	req := listByConnectionReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "id"),
		pageMetadata: things.PageMetadata{
			Offset:    o,
			Limit:     l,
			Connected: c,
			Order:     or,
			Dir:       d,
		},
	}

	return req, nil
}

func decodeConnection(_ context.Context, r *http.Request) (interface{}, error) {
	req := connectionReq{
		token:   r.Header.Get("Authorization"),
		chanID:  bone.GetValue(r, "chanId"),
		thingID: bone.GetValue(r, "thingId"),
	}

	return req, nil
}

func decodeCreateConnections(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createConnectionsReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch errorVal := err.(type) {
	case errors.Error:
		w.Header().Set("Content-Type", contentType)
		switch {
		case errors.Contains(errorVal, things.ErrUnauthorizedAccess),
			errors.Contains(errorVal, things.ErrEntityConnected):
			w.WriteHeader(http.StatusUnauthorized)

		case errors.Contains(errorVal, errInvalidQueryParams):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, errUnsupportedContentType):
			w.WriteHeader(http.StatusUnsupportedMediaType)

		case errors.Contains(errorVal, things.ErrMalformedEntity):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, things.ErrNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Contains(errorVal, things.ErrConflict):
			w.WriteHeader(http.StatusConflict)

		case errors.Contains(errorVal, things.ErrScanMetadata),
			errors.Contains(errorVal, things.ErrSelectEntity):
			w.WriteHeader(http.StatusUnprocessableEntity)

		case errors.Contains(errorVal, things.ErrCreateEntity),
			errors.Contains(errorVal, things.ErrUpdateEntity),
			errors.Contains(errorVal, things.ErrViewEntity),
			errors.Contains(errorVal, things.ErrRemoveEntity),
			errors.Contains(errorVal, things.ErrConnect),
			errors.Contains(errorVal, things.ErrDisconnect),
			errors.Contains(errorVal, groups.ErrCreateGroup):
			w.WriteHeader(http.StatusBadRequest)

		case errors.Contains(errorVal, io.ErrUnexpectedEOF),
			errors.Contains(errorVal, io.EOF):
			w.WriteHeader(http.StatusBadRequest)

		case errors.Contains(errorVal, things.ErrCreateUUID):
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		if errorVal.Msg() != "" {
			if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errInvalidQueryParams
	}

	return val, nil
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidQueryParams
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
}

func readMetadataQuery(r *http.Request, key string) (map[string]interface{}, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return nil, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(errInvalidQueryParams, err)
	}

	return m, nil
}

func readBoolQuery(r *http.Request, key string) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return true, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return true, nil
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return true, errInvalidQueryParams
	}

	return b, nil
}
