//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package cassandra

import (
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with version and metrics.
func MakeHandler() http.Handler {
	r := bone.New()
	r.GetFunc("/version", mainflux.Version("cassandra-writer"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}
