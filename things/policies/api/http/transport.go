// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

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
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/policies"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(csvc clients.Service, psvc policies.Service, mux *bone.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Post("/channels/:chanID/access", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("authorize"))(authorizeEndpoint(psvc)),
		decodeCanAccess,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/identify", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("identify"))(identifyEndpoint(csvc)),
		decodeIdentify,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/policies", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("connect"))(connectEndpoint(psvc)),
		decodeConnectThing,
		api.EncodeResponse,
		opts...,
	))

	mux.Put("/policies", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("update_policy"))(updatePolicyEndpoint(psvc)),
		decodeUpdatePolicy,
		api.EncodeResponse,
		opts...,
	))

	mux.Get("/policies", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("list_policies"))(listPoliciesEndpoint(psvc)),
		decodeListPolicies,
		api.EncodeResponse,
		opts...,
	))

	mux.Delete("/policies/:subject/:object", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("disconnect"))(disconnectEndpoint(psvc)),
		decodeDisconnectThing,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/connect", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("bulk_connect"))(connectThingsEndpoint(psvc)),
		decodeConnectList,
		api.EncodeResponse,
		opts...,
	))

	mux.Post("/disconnect", kithttp.NewServer(
		otelkit.EndpointMiddleware(otelkit.WithOperation("bulk_disconnect"))(disconnectThingsEndpoint(psvc)),
		decodeConnectList,
		api.EncodeResponse,
		opts...,
	))

	return mux

}

func decodeConnectThing(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := createPolicyReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeDisconnectThing(_ context.Context, r *http.Request) (interface{}, error) {
	req := createPolicyReq{
		token:   apiutil.ExtractBearerToken(r),
		Subject: bone.GetValue(r, api.SubjectKey),
		Object:  bone.GetValue(r, api.ObjectKey),
	}

	return req, nil
}

func decodeConnectList(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := createPoliciesReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeIdentify(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := identifyReq{secret: apiutil.ExtractThingKey(r)}

	return req, nil
}

func decodeCanAccess(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := authorizeReq{Object: bone.GetValue(r, "chanID")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdatePolicy(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.ErrUnsupportedContentType
	}
	req := policyReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListPolicies(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, err
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, err
	}
	c, err := apiutil.ReadStringQuery(r, api.ClientKey, "")
	if err != nil {
		return nil, err
	}
	g, err := apiutil.ReadStringQuery(r, api.GroupKey, "")
	if err != nil {
		return nil, err
	}
	a, err := apiutil.ReadStringQuery(r, api.ActionKey, "")
	if err != nil {
		return nil, err
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, err
	}

	req := listPoliciesReq{
		token:  apiutil.ExtractBearerToken(r),
		offset: o,
		limit:  l,
		client: c,
		group:  g,
		action: a,
		owner:  oid,
	}

	return req, nil
}
