// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with health check and metrics.
func MakeHandler(svcName, instanceID string) http.Handler {
	r := bone.New()
	r.GetFunc("/health", magistrala.Health(svcName, instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}
