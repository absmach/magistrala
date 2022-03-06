// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	defOffset   = 0
	defLimit    = 10
)

var (
	fullMatch    = []string{"state", "external_id", "mainflux_id", "mainflux_key"}
	partialMatch = []string{"name"}
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc bootstrap.Service, reader bootstrap.ConfigReader, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	r := bone.New()

	r.Post("/things/configs", kithttp.NewServer(
		addEndpoint(svc),
		decodeAddRequest,
		encodeResponse,
		opts...))

	r.Get("/things/configs/:id", kithttp.NewServer(
		viewEndpoint(svc),
		decodeEntityRequest,
		encodeResponse,
		opts...))

	r.Put("/things/configs/:id", kithttp.NewServer(
		updateEndpoint(svc),
		decodeUpdateRequest,
		encodeResponse,
		opts...))

	r.Patch("/things/configs/certs/:id", kithttp.NewServer(
		updateCertEndpoint(svc),
		decodeUpdateCertRequest,
		encodeResponse,
		opts...))

	r.Put("/things/configs/connections/:id", kithttp.NewServer(
		updateConnEndpoint(svc),
		decodeUpdateConnRequest,
		encodeResponse,
		opts...))

	r.Get("/things/configs", kithttp.NewServer(
		listEndpoint(svc),
		decodeListRequest,
		encodeResponse,
		opts...))

	r.Get("/things/bootstrap/:external_id", kithttp.NewServer(
		bootstrapEndpoint(svc, reader, false),
		decodeBootstrapRequest,
		encodeResponse,
		opts...))

	r.Get("/things/bootstrap/secure/:external_id", kithttp.NewServer(
		bootstrapEndpoint(svc, reader, true),
		decodeBootstrapRequest,
		encodeSecureRes,
		opts...))

	r.Put("/things/state/:id", kithttp.NewServer(
		stateEndpoint(svc),
		decodeStateRequest,
		encodeResponse,
		opts...))

	r.Delete("/things/configs/:id", kithttp.NewServer(
		removeEndpoint(svc),
		decodeEntityRequest,
		encodeResponse,
		opts...))

	r.GetFunc("/health", mainflux.Health("bootstrap"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeAddRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := addReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := updateReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateCertRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := updateCertReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateConnRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := updateConnReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListRequest(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errors.ErrInvalidQueryParams
	}

	req := listReq{
		token:  apiutil.ExtractBearerToken(r),
		filter: parseFilter(q),
		offset: o,
		limit:  l,
	}

	return req, nil
}

func decodeBootstrapRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := bootstrapReq{
		id:  bone.GetValue(r, "external_id"),
		key: apiutil.ExtractThingKey(r),
	}

	return req, nil
}

func decodeStateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := changeStateReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeEntityRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := entityReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
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

func encodeSecureRes(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if b, ok := response.([]byte); ok {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken,
		err == apiutil.ErrBearerKey:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errors.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID,
		err == apiutil.ErrBootstrapState,
		err == apiutil.ErrLimitSize:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, bootstrap.ErrExternalKey),
		errors.Contains(err, bootstrap.ErrExternalKeySecure),
		errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, bootstrap.ErrThings):
		w.WriteHeader(http.StatusServiceUnavailable)

	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrViewEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
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
