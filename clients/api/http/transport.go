// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"
	"net/http"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/clients"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for clients and Groups API endpoints.
func MakeHandler(tsvc clients.Service, authn smqauthn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string, idp supermq.IDProvider) http.Handler {
	mux = clientsHandler(tsvc, authn, mux, logger, idp)

	mux.Get("/health", supermq.Health("clients", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
