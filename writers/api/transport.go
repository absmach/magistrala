// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with version and metrics.
func MakeHandler(svcName string) http.Handler {
	r := bone.New()
	r.GetFunc("/version", mainflux.Version(svcName))
	r.Handle("/metrics", promhttp.Handler())

	return r
}
