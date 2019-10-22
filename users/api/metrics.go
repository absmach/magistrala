// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/users"
)

var _ users.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     users.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc users.Service, counter metrics.Counter, latency metrics.Histogram) users.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Register(ctx context.Context, user users.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "register").Add(1)
		ms.latency.With("method", "register").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Register(ctx, user)
}

func (ms *metricsMiddleware) Login(ctx context.Context, user users.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "login").Add(1)
		ms.latency.With("method", "login").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Login(ctx, user)
}

func (ms *metricsMiddleware) Identify(key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identity").Add(1)
		ms.latency.With("method", "identity").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(key)
}

func (ms *metricsMiddleware) UserInfo(ctx context.Context, key string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "user_info").Add(1)
		ms.latency.With("method", "user_info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UserInfo(ctx, key)
}

func (ms *metricsMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "generate_reset_token").Add(1)
		ms.latency.With("method", "generate_reset_token").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GenerateResetToken(ctx, email, host)
}

func (ms *metricsMiddleware) ChangePassword(ctx context.Context, email, password, oldPassword string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "change_password").Add(1)
		ms.latency.With("method", "change_password").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ChangePassword(ctx, email, password, oldPassword)
}

func (ms *metricsMiddleware) ResetPassword(ctx context.Context, email, password string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reset_password").Add(1)
		ms.latency.With("method", "reset_password").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ResetPassword(ctx, email, password)
}

func (ms *metricsMiddleware) SendPasswordReset(ctx context.Context, host, email, token string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_password_reset").Add(1)
		ms.latency.With("method", "send_password_reset").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SendPasswordReset(ctx, host, email, token)
}
