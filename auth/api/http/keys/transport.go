// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
)

const contentType = "application/json"

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	mux.Route("/keys", func(r chi.Router) {
		r.Post("/", kithttp.NewServer(
			issueEndpoint(svc),
			decodeIssue,
			encodeResponse,
			opts...,
		).ServeHTTP)

		r.Get("/{id}", kithttp.NewServer(
			(retrieveEndpoint(svc)),
			decodeKeyReq,
			encodeResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/{id}", kithttp.NewServer(
			(revokeEndpoint(svc)),
			decodeKeyReq,
			encodeResponse,
			opts...,
		).ServeHTTP)
	})
	return mux
}

func decodeIssue(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	req := issueKeyReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeKeyReq(_ context.Context, r *http.Request) (interface{}, error) {
	req := keyReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
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
	switch {
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID,
		err == apiutil.ErrInvalidAPIKey:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
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
