// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/absmach/magistrala"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/oauth2"
	"github.com/absmach/magistrala/users"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Users and Groups API endpoints.
func MakeHandler(cls users.Service, authn mgauthn.Authentication, tokensvc grpcTokenV1.TokenServiceClient, selfRegister bool, mux *chi.Mux, logger *slog.Logger, instanceID string, pr *regexp.Regexp, providers ...oauth2.Provider) http.Handler {
	mux = usersHandler(cls, authn, tokensvc, selfRegister, mux, logger, pr, providers...)

	mux.Get("/health", magistrala.Health("users", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
