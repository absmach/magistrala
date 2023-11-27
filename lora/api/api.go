// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(instanceID string) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", magistrala.Health("lora-adapter", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}
