// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/users"
	"github.com/go-kit/kit/metrics"
)

var _ users.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     users.Service
}

// MetricsMiddleware instruments policies service by tracking request count and latency.
func MetricsMiddleware(svc users.Service, counter metrics.Counter, latency metrics.Histogram) users.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// RegisterClient instruments RegisterClient method with metrics.
func (ms *metricsMiddleware) RegisterClient(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_client").Add(1)
		ms.latency.With("method", "register_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RegisterClient(ctx, token, client)
}

// IssueToken instruments IssueToken method with metrics.
func (ms *metricsMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_token").Add(1)
		ms.latency.With("method", "issue_token").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken instruments RefreshToken method with metrics.
func (ms *metricsMiddleware) RefreshToken(ctx context.Context, refreshToken, domainID string) (token *magistrala.Token, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "refresh_token").Add(1)
		ms.latency.With("method", "refresh_token").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RefreshToken(ctx, refreshToken, domainID)
}

// ViewClient instruments ViewClient method with metrics.
func (ms *metricsMiddleware) ViewClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewClient(ctx, token, id)
}

// ViewProfile instruments ViewProfile method with metrics.
func (ms *metricsMiddleware) ViewProfile(ctx context.Context, token string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewProfile(ctx, token)
}

// ListClients instruments ListClients method with metrics.
func (ms *metricsMiddleware) ListClients(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListClients(ctx, token, pm)
}

// SearchUsers instruments SearchClients method with metrics.
func (ms *metricsMiddleware) SearchUsers(ctx context.Context, token string, pm mgclients.Page) (mp mgclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "search_users").Add(1)
		ms.latency.With("method", "search_users").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.SearchUsers(ctx, token, pm)
}

// UpdateClient instruments UpdateClient method with metrics.
func (ms *metricsMiddleware) UpdateClient(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_name_and_metadata").Add(1)
		ms.latency.With("method", "update_client_name_and_metadata").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClient(ctx, token, client)
}

// UpdateClientTags instruments UpdateClientTags method with metrics.
func (ms *metricsMiddleware) UpdateClientTags(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_tags").Add(1)
		ms.latency.With("method", "update_client_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientTags(ctx, token, client)
}

// UpdateClientIdentity instruments UpdateClientIdentity method with metrics.
func (ms *metricsMiddleware) UpdateClientIdentity(ctx context.Context, token, id, identity string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_identity").Add(1)
		ms.latency.With("method", "update_client_identity").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientIdentity(ctx, token, id, identity)
}

// UpdateClientSecret instruments UpdateClientSecret method with metrics.
func (ms *metricsMiddleware) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_secret").Add(1)
		ms.latency.With("method", "update_client_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientSecret(ctx, token, oldSecret, newSecret)
}

// GenerateResetToken instruments GenerateResetToken method with metrics.
func (ms *metricsMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "generate_reset_token").Add(1)
		ms.latency.With("method", "generate_reset_token").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret instruments ResetSecret method with metrics.
func (ms *metricsMiddleware) ResetSecret(ctx context.Context, token, secret string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reset_secret").Add(1)
		ms.latency.With("method", "reset_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ResetSecret(ctx, token, secret)
}

// SendPasswordReset instruments SendPasswordReset method with metrics.
func (ms *metricsMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_password_reset").Add(1)
		ms.latency.With("method", "send_password_reset").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.SendPasswordReset(ctx, host, email, user, token)
}

// UpdateClientRole instruments UpdateClientRole method with metrics.
func (ms *metricsMiddleware) UpdateClientRole(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_role").Add(1)
		ms.latency.With("method", "update_client_role").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientRole(ctx, token, client)
}

// EnableClient instruments EnableClient method with metrics.
func (ms *metricsMiddleware) EnableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_client").Add(1)
		ms.latency.With("method", "enable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableClient(ctx, token, id)
}

// DisableClient instruments DisableClient method with metrics.
func (ms *metricsMiddleware) DisableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_client").Add(1)
		ms.latency.With("method", "disable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableClient(ctx, token, id)
}

// Identify instruments Identify method with metrics.
func (ms *metricsMiddleware) Identify(ctx context.Context, token string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Identify(ctx, token)
}

func (ms *metricsMiddleware) OAuthCallback(ctx context.Context, client mgclients.Client) (*magistrala.Token, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "oauth_callback").Add(1)
		ms.latency.With("method", "oauth_callback").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.OAuthCallback(ctx, client)
}

// DeleteClient instruments DeleteClient method with metrics.
func (ms *metricsMiddleware) DeleteClient(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_client").Add(1)
		ms.latency.With("method", "delete_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteClient(ctx, token, id)
}
