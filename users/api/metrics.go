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

func (ms *metricsMiddleware) Register(ctx context.Context, user users.User) (string, error) {
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

func (ms *metricsMiddleware) ViewUser(ctx context.Context, token, id string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_user").Add(1)
		ms.latency.With("method", "view_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewUser(ctx, token, id)
}

func (ms *metricsMiddleware) ViewProfile(ctx context.Context, token string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewProfile(ctx, token)
}

func (ms *metricsMiddleware) ListUsers(ctx context.Context, token string, offset, limit uint64, email string, um users.Metadata) (users.UserPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_users").Add(1)
		ms.latency.With("method", "list_users").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListUsers(ctx, token, offset, limit, email, um)
}

func (ms *metricsMiddleware) UpdateUser(ctx context.Context, token string, u users.User) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user").Add(1)
		ms.latency.With("method", "update_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateUser(ctx, token, u)
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

func (ms *metricsMiddleware) CreateGroup(ctx context.Context, token string, group users.Group) (users.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_group").Add(1)
		ms.latency.With("method", "create_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateGroup(ctx, token, group)
}

func (ms *metricsMiddleware) ListGroups(ctx context.Context, token, id string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups").Add(1)
		ms.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListGroups(ctx, token, id, offset, limit, um)
}

func (ms *metricsMiddleware) ListMembers(ctx context.Context, token, id string, offset, limit uint64, um users.Metadata) (users.UserPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_members").Add(1)
		ms.latency.With("method", "list_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListMembers(ctx, token, id, offset, limit, um)
}

func (ms *metricsMiddleware) RemoveGroup(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_group").Add(1)
		ms.latency.With("method", "remove_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveGroup(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, group users.Group) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateGroup(ctx, token, group)
}

func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, name string) (users.Group, error) {

	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroup(ctx, token, name)
}

func (ms *metricsMiddleware) Assign(ctx context.Context, token, userID, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign").Add(1)
		ms.latency.With("method", "assign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Assign(ctx, token, userID, groupID)
}

func (ms *metricsMiddleware) Unassign(ctx context.Context, token, userID, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign").Add(1)
		ms.latency.With("method", "unassign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Unassign(ctx, token, userID, groupID)
}

func (ms *metricsMiddleware) ListMemberships(ctx context.Context, token, id string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_memberships").Add(1)
		ms.latency.With("method", "list_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListMemberships(ctx, token, id, offset, limit, um)
}
