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

var (
	fullMatch    = []string{"state", "external_id", "thing_id", "thing_key"}
	partialMatch = []string{"name"}
	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc bootstrap.Service, reader bootstrap.ConfigReader, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := chi.NewRouter()

	r.Route("/things", func(r chi.Router) {
		r.Route("/configs", func(r chi.Router) {
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				addEndpoint(svc),
				decodeAddRequest,
				encodeResponse,
				opts...), "add").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listEndpoint(svc),
				decodeListRequest,
				encodeResponse,
				opts...), "list").ServeHTTP)

			r.Get("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
				viewEndpoint(svc),
				decodeEntityRequest,
				encodeResponse,
				opts...), "view").ServeHTTP)

			r.Put("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
				updateEndpoint(svc),
				decodeUpdateRequest,
				encodeResponse,
				opts...), "update").ServeHTTP)

			r.Delete("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
				removeEndpoint(svc),
				decodeEntityRequest,
				encodeResponse,
				opts...), "remove").ServeHTTP)

			r.Patch("/certs/{certID}", otelhttp.NewHandler(kithttp.NewServer(
				updateCertEndpoint(svc),
				decodeUpdateCertRequest,
				encodeResponse,
				opts...), "update_cert").ServeHTTP)

			r.Put("/connections/{connID}", otelhttp.NewHandler(kithttp.NewServer(
				updateConnEndpoint(svc),
				decodeUpdateConnRequest,
				encodeResponse,
				opts...), "update_connections").ServeHTTP)
		})

		r.Route("/bootstrap", func(r chi.Router) {
			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				bootstrapEndpoint(svc, reader, false),
				decodeBootstrapRequest,
				encodeResponse,
				opts...), "bootstrap").ServeHTTP)
			r.Get("/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
				bootstrapEndpoint(svc, reader, false),
				decodeBootstrapRequest,
				encodeResponse,
				opts...), "bootstrap").ServeHTTP)
			r.Get("/secure/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
				bootstrapEndpoint(svc, reader, true),
				decodeBootstrapRequest,
				encodeSecureRes,
				opts...), "bootstrap_secure").ServeHTTP)
		})

		r.Put("/state/{thingID}", otelhttp.NewHandler(kithttp.NewServer(
			stateEndpoint(svc),
			decodeStateRequest,
			encodeResponse,
			opts...), "update_state").ServeHTTP)
	})
	r.Get("/health", magistrala.Health("bootstrap", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeAddRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addReq{token: apiutil.ExtractBearerToken(r)}
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
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "configID"),
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
		token:   apiutil.ExtractBearerToken(r),
		thingID: chi.URLParam(r, "certID"),
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
		token:  apiutil.ExtractBearerToken(r),
		filter: parseFilter(q),
		offset: o,
		limit:  l,
	}

	return req, nil
}

func decodeBootstrapRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := bootstrapReq{
		id:  chi.URLParam(r, "externalID"),
		key: apiutil.ExtractThingKey(r),
	}

	return req, nil
}

func decodeStateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := changeStateReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "thingID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeEntityRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := entityReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "configID"),
	}

	return req, nil
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
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken),
		errors.Contains(err, apiutil.ErrBearerKey):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrBootstrapState),
		errors.Contains(err, apiutil.ErrLimitSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, svcerr.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, bootstrap.ErrExternalKey),
		errors.Contains(err, bootstrap.ErrExternalKeySecure),
		errors.Contains(err, svcerr.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, svcerr.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, bootstrap.ErrThings):
		w.WriteHeader(http.StatusServiceUnavailable)

	case errors.Contains(err, svcerr.ErrCreateEntity),
		errors.Contains(err, svcerr.ErrUpdateEntity),
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
