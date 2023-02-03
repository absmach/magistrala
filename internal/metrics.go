// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func MakeMetrics(namespace, subsystem string) (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, []string{"method"})
	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, []string{"method"})

	return counter, latency
}
