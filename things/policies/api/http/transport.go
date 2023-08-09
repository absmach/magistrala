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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(csvc clients.Service, psvc policies.Service, mux *bone.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Post("/channels/:chanID/access", otelhttp.NewHandler(kithttp.NewServer(
		authorizeEndpoint(psvc),
		decodeCanAccess,
		api.EncodeResponse,
		opts...,
	), "authorize"))

	mux.Post("/identify", otelhttp.NewHandler(kithttp.NewServer(
		identifyEndpoint(csvc),
		decodeIdentify,
		api.EncodeResponse,
		opts...,
	), "identify"))

	mux.Post("/policies", otelhttp.NewHandler(kithttp.NewServer(
		connectEndpoint(psvc),
		decodeConnectThing,
		api.EncodeResponse,
		opts...,
	), "connect"))

	mux.Put("/policies", otelhttp.NewHandler(kithttp.NewServer(
		updatePolicyEndpoint(psvc),
		decodeUpdatePolicy,
		api.EncodeResponse,
		opts...,
	), "update_policy"))

	mux.Get("/policies", otelhttp.NewHandler(kithttp.NewServer(
		listPoliciesEndpoint(psvc),
		decodeListPolicies,
		api.EncodeResponse,
		opts...,
	), "list_policies"))

	mux.Delete("/policies/:subject/:object", otelhttp.NewHandler(kithttp.NewServer(
		disconnectEndpoint(psvc),
		decodeDisconnectThing,
		api.EncodeResponse,
		opts...,
	), "disconnect"))

	mux.Post("/connect", otelhttp.NewHandler(kithttp.NewServer(
		connectThingsEndpoint(psvc),
		decodeConnectList,
		api.EncodeResponse,
		opts...,
	), "bulk_connect"))

	mux.Post("/disconnect", otelhttp.NewHandler(kithttp.NewServer(
		disconnectThingsEndpoint(psvc),
		decodeConnectList,
		api.EncodeResponse,
		opts...,
	), "bulk_disconnect"))

	return mux

}

func decodeConnectThing(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createPolicyReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
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
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := createPoliciesReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeIdentify(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := identifyReq{secret: apiutil.ExtractThingKey(r)}

	return req, nil
}

func decodeCanAccess(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := authorizeReq{Object: bone.GetValue(r, "chanID")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdatePolicy(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := policyReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeListPolicies(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	c, err := apiutil.ReadStringQuery(r, api.ClientKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	g, err := apiutil.ReadStringQuery(r, api.GroupKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	a, err := apiutil.ReadStringQuery(r, api.ActionKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	oid, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
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
