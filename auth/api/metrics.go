// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/auth"
)

var _ auth.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     auth.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc auth.Service, counter metrics.Counter, latency metrics.Histogram) auth.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_objects").Add(1)
		ms.latency.With("method", "list_objects").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListObjects(ctx, pr, nextPageToken, limit)
}

func (ms *metricsMiddleware) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_all_objects").Add(1)
		ms.latency.With("method", "list_all_objects").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListAllObjects(ctx, pr)
}

func (ms *metricsMiddleware) CountObjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "count_objects").Add(1)
		ms.latency.With("method", "count_objects").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CountObjects(ctx, pr)
}

func (ms *metricsMiddleware) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_subjects").Add(1)
		ms.latency.With("method", "list_subjects").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListSubjects(ctx, pr, nextPageToken, limit)
}

func (ms *metricsMiddleware) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_all_subjects").Add(1)
		ms.latency.With("method", "list_all_subjects").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListAllSubjects(ctx, pr)
}

func (ms *metricsMiddleware) CountSubjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "count_subjects").Add(1)
		ms.latency.With("method", "count_subjects").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CountSubjects(ctx, pr)
}

func (ms *metricsMiddleware) Issue(ctx context.Context, token string, key auth.Key) (auth.Token, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_key").Add(1)
		ms.latency.With("method", "issue_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Issue(ctx, token, key)
}

func (ms *metricsMiddleware) Revoke(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_key").Add(1)
		ms.latency.With("method", "revoke_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Revoke(ctx, token, id)
}

func (ms *metricsMiddleware) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve_key").Add(1)
		ms.latency.With("method", "retrieve_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RetrieveKey(ctx, token, id)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, token string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, token)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, pr auth.PolicyReq) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Authorize(ctx, pr)
}

func (ms *metricsMiddleware) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_policy").Add(1)
		ms.latency.With("method", "add_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.AddPolicy(ctx, pr)
}

func (ms *metricsMiddleware) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_policy_bulk").Add(1)
		ms.latency.With("method", "create_policy_bulk").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddPolicies(ctx, token, object, subjectIDs, relations)
}

func (ms *metricsMiddleware) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_policy").Add(1)
		ms.latency.With("method", "delete_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeletePolicy(ctx, pr)
}

func (ms *metricsMiddleware) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_policies").Add(1)
		ms.latency.With("method", "delete_policies").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeletePolicies(ctx, token, object, subjectIDs, relations)
}
