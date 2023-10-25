// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	mainflux "github.com/absmach/magistrala"
	mflog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/users"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Users and Groups API endpoints.
func MakeHandler(cls users.Service, grps groups.Service, mux *chi.Mux, logger mflog.Logger, instanceID string) http.Handler {
	clientsHandler(cls, mux, logger)
	groupsHandler(grps, mux, logger)

	mux.Get("/health", mainflux.Health("users", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
