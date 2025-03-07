// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

// MakeMetrics returns an instance of Prometheus implementations for metrics.
// It returns a request counter and a request latency summary.
//
//	counter, latency := metrics.MakeMetrics("demo-service", "api")
func MakeMetrics(namespace, subsystem string) (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, []string{"method"})
	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace:  namespace,
		Subsystem:  subsystem,
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		Name:       "request_latency_microseconds",
		Help:       "Total duration of requests in microseconds.",
	}, []string{"method"})

	return counter, latency
}
