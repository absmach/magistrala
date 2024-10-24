// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/clients"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for clients and Groups API endpoints.
func MakeHandler(tsvc clients.Service, authn mgauthn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string) http.Handler {
	mux = clientsHandler(tsvc, authn, mux, logger)

	mux.Get("/health", magistrala.Health("clients", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
