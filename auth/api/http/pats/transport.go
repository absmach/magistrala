// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package pats

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
)

const (
	contentType = "application/json"
	defInterval = "30d"
	patPrefix   = "pat_"
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

		r.Get("/", kithttp.NewServer(
			listPATSEndpoint(svc),
			decodeListPATSRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/", kithttp.NewServer(
			clearAllPATEndpoint(svc),
			decodeClearAllPATRequest,
			api.EncodeResponse,
			opts...,
		).ServeHTTP)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", kithttp.NewServer(
				retrievePATEndpoint(svc),
				decodeRetrievePATRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)

			r.Patch("/name", kithttp.NewServer(
				updatePATNameEndpoint(svc),
				decodeUpdatePATNameRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)

			r.Patch("/description", kithttp.NewServer(
				updatePATDescriptionEndpoint(svc),
				decodeUpdatePATDescriptionRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)

			r.Delete("/", kithttp.NewServer(
				deletePATEndpoint(svc),
				decodeDeletePATRequest,
				api.EncodeResponse,
				opts...,
			).ServeHTTP)

			r.Route("/secret", func(r chi.Router) {
				r.Patch("/reset", kithttp.NewServer(
					resetPATSecretEndpoint(svc),
					decodeResetPATSecretRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)

				r.Patch("/revoke", kithttp.NewServer(
					revokePATSecretEndpoint(svc),
					decodeRevokePATSecretRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)
			})

			r.Route("/scope", func(r chi.Router) {
				r.Patch("/add", kithttp.NewServer(
					addScopeEndpoint(svc),
					decodeAddScopeRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)

				r.Get("/", kithttp.NewServer(
					listScopesEndpoint(svc),
					decodeListScopeRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)

				r.Patch("/remove", kithttp.NewServer(
					removeScopeEndpoint(svc),
					decodeRemoveScopeRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)

				r.Delete("/", kithttp.NewServer(
					clearAllScopeEndpoint(svc),
					decodeClearAllScopeRequest,
					api.EncodeResponse,
					opts...,
				).ServeHTTP)
			})
		})
	})
	return mux
}

func decodeCreatePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := createPatReq{token: token}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeRetrievePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}

	req := retrievePatReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}
	return req, nil
}

func decodeUpdatePATNameRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := updatePatNameReq{
		token: token,
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

	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := updatePatDescriptionReq{
		token: token,
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
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := listPatsReq{
		token:  token,
		limit:  l,
		offset: o,
	}
	return req, nil
}

func decodeDeletePATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	return deletePatReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}, nil
}

func decodeResetPATSecretRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := resetPatSecretReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeRevokePATSecretRequest(_ context.Context, r *http.Request) (interface{}, error) {
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	return revokePatSecretReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}, nil
}

func decodeClearAllPATRequest(_ context.Context, r *http.Request) (interface{}, error) {
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}

	return clearAllPATReq{
		token: token,
	}, nil
}

func decodeAddScopeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}

	req := addScopeReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScopeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}
	req := listScopesReq{
		token:  token,
		limit:  l,
		offset: o,
		patID:  chi.URLParam(r, "id"),
	}
	return req, nil
}

func decodeRemoveScopeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}

	req := removeScopeReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeClearAllScopeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	token := apiutil.ExtractBearerToken(r)
	if strings.HasPrefix(token, patPrefix) {
		return nil, apiutil.ErrUnsupportedTokenType
	}

	return clearAllScopeReq{
		token: token,
		id:    chi.URLParam(r, "id"),
	}, nil
}
