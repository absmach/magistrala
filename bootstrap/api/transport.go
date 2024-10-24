// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	contentType     = "application/json"
	byteContentType = "application/octet-stream"
	offsetKey       = "offset"
	limitKey        = "limit"
	defOffset       = 0
	defLimit        = 10
)

var (
	fullMatch    = []string{"state", "external_id", "client_id", "client_key"}
	partialMatch = []string{"name"}
	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc bootstrap.Service, authn mgauthn.Authentication, reader bootstrap.ConfigReader, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()

	r.Route("/{domainID}/clients", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, true))

			r.Route("/configs", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addEndpoint(svc),
					decodeAddRequest,
					api.EncodeResponse,
					opts...), "add").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listEndpoint(svc),
					decodeListRequest,
					api.EncodeResponse,
					opts...), "list").ServeHTTP)

				r.Get("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					viewEndpoint(svc),
					decodeEntityRequest,
					api.EncodeResponse,
					opts...), "view").ServeHTTP)

				r.Put("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					updateEndpoint(svc),
					decodeUpdateRequest,
					api.EncodeResponse,
					opts...), "update").ServeHTTP)

				r.Delete("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					removeEndpoint(svc),
					decodeEntityRequest,
					api.EncodeResponse,
					opts...), "remove").ServeHTTP)

				r.Patch("/certs/{certID}", otelhttp.NewHandler(kithttp.NewServer(
					updateCertEndpoint(svc),
					decodeUpdateCertRequest,
					api.EncodeResponse,
					opts...), "update_cert").ServeHTTP)

				r.Put("/connections/{connID}", otelhttp.NewHandler(kithttp.NewServer(
					updateConnEndpoint(svc),
					decodeUpdateConnRequest,
					api.EncodeResponse,
					opts...), "update_connections").ServeHTTP)
			})
		})

		r.With(api.AuthenticateMiddleware(authn, true)).Put("/state/{clientID}", otelhttp.NewHandler(kithttp.NewServer(
			stateEndpoint(svc),
			decodeStateRequest,
			api.EncodeResponse,
			opts...), "update_state").ServeHTTP)
	})

	r.Route("/clients/bootstrap", func(r chi.Router) {
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, false),
			decodeBootstrapRequest,
			api.EncodeResponse,
			opts...), "bootstrap").ServeHTTP)
		r.Get("/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, false),
			decodeBootstrapRequest,
			api.EncodeResponse,
			opts...), "bootstrap").ServeHTTP)
		r.Get("/secure/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, true),
			decodeBootstrapRequest,
			encodeSecureRes,
			opts...), "bootstrap_secure").ServeHTTP)
	})

	r.Get("/health", magistrala.Health("bootstrap", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeAddRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateReq{
		id: chi.URLParam(r, "configID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateCertRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateCertReq{
		clientID: chi.URLParam(r, "certID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateConnRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateConnReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "connID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeListRequest(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidQueryParams)
	}

	req := listReq{
		filter: parseFilter(q),
		offset: o,
		limit:  l,
	}

	return req, nil
}

func decodeBootstrapRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := bootstrapReq{
		id:  chi.URLParam(r, "externalID"),
		key: apiutil.ExtractClientSecret(r),
	}

	return req, nil
}

func decodeStateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := changeStateReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "clientID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeEntityRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := entityReq{
		id: chi.URLParam(r, "configID"),
	}

	return req, nil
}

func encodeSecureRes(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", byteContentType)
	w.WriteHeader(http.StatusOK)
	if b, ok := response.([]byte); ok {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func parseFilter(values url.Values) bootstrap.Filter {
	ret := bootstrap.Filter{
		FullMatch:    make(map[string]string),
		PartialMatch: make(map[string]string),
	}
	for k := range values {
		if contains(fullMatch, k) {
			ret.FullMatch[k] = values.Get(k)
		}
		if contains(partialMatch, k) {
			ret.PartialMatch[k] = strings.ToLower(values.Get(k))
		}
	}

	return ret
}

func contains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}
