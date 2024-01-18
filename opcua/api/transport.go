// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/opcua"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType     = "application/json"
	serverParam     = "server"
	namespaceParam  = "namespace"
	identifierParam = "identifier"
	defNamespace    = "ns=0" // Standard root namespace
	defIdentifier   = "i=84" // Standard root identifier
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc opcua.Service, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := chi.NewRouter()

	r.Get("/browse", kithttp.NewServer(
		browseEndpoint(svc),
		decodeBrowse,
		encodeResponse,
		opts...,
	).ServeHTTP)

	r.Get("/health", magistrala.Health("opcua-adapter", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeBrowse(_ context.Context, r *http.Request) (interface{}, error) {
	s, err := apiutil.ReadStringQuery(r, serverParam, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	n, err := apiutil.ReadStringQuery(r, namespaceParam, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	i, err := apiutil.ReadStringQuery(r, identifierParam, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	if n == "" || i == "" {
		n = defNamespace
		i = defIdentifier
	}

	req := browseReq{
		ServerURI:  s,
		Namespace:  n,
		Identifier: i,
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

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID:
		w.WriteHeader(http.StatusBadRequest)

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
