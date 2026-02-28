// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala/provision"
	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc provision.Service, authn smqauthn.AuthNMiddleware, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()

	r.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.WithOptions(smqauthn.WithDomainCheck(true)).Middleware())
		r.Route("/mapping", func(r chi.Router) {
			r.Post("/", kithttp.NewServer(
				doProvision(svc),
				decodeProvisionRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)
			r.Get("/", kithttp.NewServer(
				getMapping(svc),
				decodeMappingRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)
		})
		r.Post("/cert", kithttp.NewServer(
			issueCert(svc),
			decodeCertRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", supermq.Health("provision", instanceID))

	return r
}

func decodeProvisionRequest(_ context.Context, r *http.Request) (any, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := provisionReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeMappingRequest(_ context.Context, r *http.Request) (any, error) {
	return nil, nil
}

func decodeCertRequest(_ context.Context, r *http.Request) (any, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := certReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}
