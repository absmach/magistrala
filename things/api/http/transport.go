// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	chHttp "github.com/absmach/magistrala/internal/channels/api/http"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/channels"
	"github.com/absmach/magistrala/things"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Things and Groups API endpoints.
func MakeHandler(tsvc things.Service, csvc channels.Service, authn mgauthn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string) http.Handler {
	mux = clientsHandler(tsvc, mux, authn, logger)
	mux = chHttp.MakeHandler(csvc, authn, mux, logger, instanceID)

	mux.Get("/health", magistrala.Health("things", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
