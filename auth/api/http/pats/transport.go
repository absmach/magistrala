// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package pats

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
)

const (
	contentType = "application/json"
	defInterval = "30d"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Route("/pats", func(r chi.Router) {
		r.Post("/", kithttp.NewServer(
			createPATEndpoint(svc),
			decodeCreatePATRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Get("/{id}", kithttp.NewServer(
			(retrievePATEndpoint(svc)),
			decodeRetrievePATRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/name", kithttp.NewServer(
			(updatePATNameEndpoint(svc)),
			decodeUpdatePATNameRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/description", kithttp.NewServer(
			(updatePATDescriptionEndpoint(svc)),
			decodeUpdatePATDescriptionRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Get("/", kithttp.NewServer(
			(listPATSEndpoint(svc)),
			decodeListPATSRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/{id}", kithttp.NewServer(
			(deletePATEndpoint(svc)),
			decodeDeletePATRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/secret/reset", kithttp.NewServer(
			(resetPATSecretEndpoint(svc)),
			decodeResetPATSecretRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/secret/revoke", kithttp.NewServer(
			(revokePATSecretEndpoint(svc)),
			decodeRevokePATSecretRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/scope/add", kithttp.NewServer(
			(addPATScopeEntryEndpoint(svc)),
			decodeAddPATScopeEntryRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}/scope/remove", kithttp.NewServer(
			(removePATScopeEntryEndpoint(svc)),
			decodeRemovePATScopeEntryRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/{id}/scope", kithttp.NewServer(
			(clearPATAllScopeEntryEndpoint(svc)),
			decodeClearPATAllScopeEntryRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Post("/authorize", kithttp.NewServer(
			(authorizePATEndpoint(svc)),
			decodeAuthorizePATRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)
	})
	return mux
}

func decodeCreatePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createPatReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeRetrievePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := retrievePatReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	return req, nil
}

func decodeUpdatePATNameRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updatePatNameReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeUpdatePATDescriptionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updatePatDescriptionReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeListPATSRequest(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := listPatsReq{
		token:  apiutil.ExtractBearerToken(r),
		limit:  l,
		offset: o,
	}
	return req, nil
}

func decodeDeletePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return deletePatReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}, nil
}

func decodeResetPATSecretRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := resetPatSecretReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeRevokePATSecretRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return revokePatSecretReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}, nil
}

func decodeAddPATScopeEntryRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := addPatScopeEntryReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeRemovePATScopeEntryRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removePatScopeEntryReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeClearPATAllScopeEntryRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	return clearAllScopeEntryReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "id"),
	}, nil
}

func decodeAuthorizePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := authorizePATReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}
