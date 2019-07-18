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
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const contentType = "application/json"

var errUnsupportedContentType = errors.New("unsupported content type")

// MakeHandler returns a HTTP handler for auth API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc things.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/identify", kithttp.NewServer(
		kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
		decodeIdentify,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/:chanId/access", kithttp.NewServer(
		kitot.TraceServer(tracer, "can_access")(canAccessEndpoint(svc)),
		decodeCanAccess,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/:chanId/access-by-id", kithttp.NewServer(
		kitot.TraceServer(tracer, "can_access_by_id")(canAccessByIDEndpoint(svc)),
		decodeCanAccessByID,
		encodeResponse,
		opts...,
	))

	return r
}

func decodeIdentify(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := identifyReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeCanAccess(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := canAccessReq{
		chanID: bone.GetValue(r, "chanId"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeCanAccessByID(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := canAccessByIDReq{
		chanID: bone.GetValue(r, "chanId"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
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
	case things.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
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
