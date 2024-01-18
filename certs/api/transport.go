// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	defOffset   = 0
	defLimit    = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := chi.NewRouter()

	r.Route("/certs", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			issueCert(svc),
			decodeCerts,
			encodeResponse,
			opts...,
		), "issue").ServeHTTP)
		r.Get("/{certID}", otelhttp.NewHandler(kithttp.NewServer(
			viewCert(svc),
			decodeViewCert,
			encodeResponse,
			opts...,
		), "view").ServeHTTP)
		r.Delete("/{certID}", otelhttp.NewHandler(kithttp.NewServer(
			revokeCert(svc),
			decodeRevokeCerts,
			encodeResponse,
			opts...,
		), "revoke").ServeHTTP)
	})
	r.Get("/serials/{thingID}", otelhttp.NewHandler(kithttp.NewServer(
		listSerials(svc),
		decodeListCerts,
		encodeResponse,
		opts...,
	), "list_serials").ServeHTTP)

	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", magistrala.Health("certs", instanceID))

	return r
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(magistrala.Response); ok {
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

func decodeListCerts(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: chi.URLParam(r, "thingID"),
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeViewCert(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewReq{
		token:    apiutil.ExtractBearerToken(r),
		serialID: chi.URLParam(r, "certID"),
	}

	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addCertsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	return req, nil
}

func decodeRevokeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	req := revokeReq{
		token:  apiutil.ExtractBearerToken(r),
		certID: chi.URLParam(r, "certID"),
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrMissingCertData),
		errors.Contains(err, apiutil.ErrInvalidCertData),
		errors.Contains(err, apiutil.ErrLimitSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, svcerr.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, svcerr.ErrCreateEntity),
		errors.Contains(err, svcerr.ErrViewEntity),
		errors.Contains(err, svcerr.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
