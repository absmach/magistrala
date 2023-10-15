// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mainflux/mainflux"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/users"
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
