// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
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
func MakeHandler(svc certs.Service, logger logger.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/certs", otelhttp.NewHandler(kithttp.NewServer(
		issueCert(svc),
		decodeCerts,
		encodeResponse,
		opts...,
	), "issue"))

	r.Get("/certs/:certID", otelhttp.NewHandler(kithttp.NewServer(
		viewCert(svc),
		decodeViewCert,
		encodeResponse,
		opts...,
	), "view"))

	r.Delete("/certs/:certID", otelhttp.NewHandler(kithttp.NewServer(
		revokeCert(svc),
		decodeRevokeCerts,
		encodeResponse,
		opts...,
	), "revoke"))

	r.Get("/serials/:thingID", otelhttp.NewHandler(kithttp.NewServer(
		listSerials(svc),
		decodeListCerts,
		encodeResponse,
		opts...,
	), "list_serials"))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", magistrala.Health("certs", instanceID))

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
	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, "thingID"),
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeViewCert(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewReq{
		token:    apiutil.ExtractBearerToken(r),
		serialID: bone.GetValue(r, "certID"),
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
		certID: bone.GetValue(r, "certID"),
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrMissingCertData),
		errors.Contains(err, apiutil.ErrInvalidCertData),
		errors.Contains(err, apiutil.ErrLimitSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrViewEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
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
