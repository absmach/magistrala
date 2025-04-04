// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq"
	sapi "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func MakeHandler(svc alarms.Service, logger *slog.Logger, idp supermq.IDProvider, instanceID string, authn smqauthn.Authentication) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, sapi.EncodeError)),
	}

	mux := chi.NewRouter()
	mux.Group(func(r chi.Router) {
		r.Use(sapi.AuthenticateMiddleware(authn, true))
		r.Use(sapi.RequestIDMiddleware(idp))
		r.Route("/rules", func(r chi.Router) {
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				createRuleEndpoint(svc),
				decodeCreateRuleReq,
				sapi.EncodeResponse,
				opts...,
			), "create_client").ServeHTTP)
		})
	})

	return mux
}

func decodeCreateRuleReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), sapi.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req createRuleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}
