//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offset      = "offset"
	limit       = "limit"
	name        = "name"

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
		kitot.TraceServer(tracer, "add_thing")(addThingEndpoint(svc)),
		decodeThingCreation,
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

	r.Delete("/channels/:chanId/things/:thingId", kithttp.NewServer(
		kitot.TraceServer(tracer, "disconnect")(disconnectEndpoint(svc)),
		decodeConnection,
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

	req := addThingReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	return req, nil
}

func decodeChannelCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := createChannelReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
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
		return nil, err
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
	o, err := readUintQuery(r, offset, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limit, defLimit)
	if err != nil {
		return nil, err
	}

	n, err := readStringQuery(r, name)
	if err != nil {
		return nil, err
	}

	req := listResourcesReq{
		token:  r.Header.Get("Authorization"),
		offset: o,
		limit:  l,
		name:   n,
	}

	return req, nil
}

func decodeListByConnection(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := readUintQuery(r, offset, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limit, defLimit)
	if err != nil {
		return nil, err
	}

	req := listByConnectionReq{
		token:  r.Header.Get("Authorization"),
		id:     bone.GetValue(r, "id"),
		offset: o,
		limit:  l,
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
	w.Header().Set("Content-Type", contentType)

	switch err {
	case things.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case things.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case things.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case things.ErrConflict:
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case io.ErrUnexpectedEOF:
		w.WriteHeader(http.StatusBadRequest)
	case io.EOF:
		w.WriteHeader(http.StatusBadRequest)
	default:
		switch err.(type) {
		case *json.SyntaxError:
			w.WriteHeader(http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
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
