// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/api/http/keys"
	"github.com/absmach/magistrala/auth/api/http/policies"
	"github.com/absmach/magistrala/logger"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, logger logger.Logger, instanceID string) http.Handler {
	mux := bone.New()

	mux = keys.MakeHandler(svc, mux, logger)
	mux = policies.MakeHandler(svc, mux, logger)

	mux.GetFunc("/health", magistrala.Health("auth", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
