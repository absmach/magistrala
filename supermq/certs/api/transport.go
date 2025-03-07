// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/certs"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	revokeKey   = "revoked"
	defRevoke   = "false"
	defOffset   = 0
	defLimit    = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, authn smqauthn.Authentication, logger *slog.Logger, instanceID string, idp supermq.IDProvider) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))
		r.Use(api.RequestIDMiddleware(idp))

		r.Route("/{domainID}", func(r chi.Router) {
			r.Route("/certs", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					issueCert(svc),
					decodeCerts,
					api.EncodeResponse,
					opts...,
				), "issue").ServeHTTP)
				r.Get("/{certID}", otelhttp.NewHandler(kithttp.NewServer(
					viewCert(svc),
					decodeViewCert,
					api.EncodeResponse,
					opts...,
				), "view").ServeHTTP)
				r.Delete("/{certID}", otelhttp.NewHandler(kithttp.NewServer(
					revokeCert(svc),
					decodeRevokeCerts,
					api.EncodeResponse,
					opts...,
				), "revoke").ServeHTTP)
			})
			r.Get("/serials/{clientID}", otelhttp.NewHandler(kithttp.NewServer(
				listSerials(svc),
				decodeListCerts,
				api.EncodeResponse,
				opts...,
			), "list_serials").ServeHTTP)
		})
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", supermq.Health("certs", instanceID))

	return r
}

func decodeListCerts(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	rv, err := apiutil.ReadStringQuery(r, revokeKey, defRevoke)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listReq{
		clientID: chi.URLParam(r, "clientID"),
		pm: certs.PageMetadata{
			Offset:  o,
			Limit:   l,
			Revoked: rv,
		},
	}
	return req, nil
}

func decodeViewCert(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewReq{
		serialID: chi.URLParam(r, "certID"),
	}

	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addCertsReq{
		token:    apiutil.ExtractBearerToken(r),
		domainID: chi.URLParam(r, "domainID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	return req, nil
}

func decodeRevokeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	req := revokeReq{
		token:    apiutil.ExtractBearerToken(r),
		certID:   chi.URLParam(r, "certID"),
		domainID: chi.URLParam(r, "domainID"),
	}

	return req, nil
}
