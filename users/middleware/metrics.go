// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
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
func (ms *metricsMiddleware) RegisterClient(ctx context.Context, session authn.Session, client mgclients.Client, selfRegister bool) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_client").Add(1)
		ms.latency.With("method", "register_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RegisterClient(ctx, session, client, selfRegister)
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
func (ms *metricsMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (token *magistrala.Token, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "refresh_token").Add(1)
		ms.latency.With("method", "refresh_token").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RefreshToken(ctx, session, refreshToken, domainID)
}

// ViewClient instruments ViewClient method with metrics.
func (ms *metricsMiddleware) ViewClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewClient(ctx, session, id)
}

// ViewProfile instruments ViewProfile method with metrics.
func (ms *metricsMiddleware) ViewProfile(ctx context.Context, session authn.Session) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewProfile(ctx, session)
}

// ListClients instruments ListClients method with metrics.
func (ms *metricsMiddleware) ListClients(ctx context.Context, session authn.Session, pm mgclients.Page) (mgclients.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListClients(ctx, session, pm)
}

// SearchUsers instruments SearchClients method with metrics.
func (ms *metricsMiddleware) SearchUsers(ctx context.Context, pm mgclients.Page) (mp mgclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "search_users").Add(1)
		ms.latency.With("method", "search_users").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.SearchUsers(ctx, pm)
}

// UpdateClient instruments UpdateClient method with metrics.
func (ms *metricsMiddleware) UpdateClient(ctx context.Context, session authn.Session, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_name_and_metadata").Add(1)
		ms.latency.With("method", "update_client_name_and_metadata").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClient(ctx, session, client)
}

// UpdateClientTags instruments UpdateClientTags method with metrics.
func (ms *metricsMiddleware) UpdateClientTags(ctx context.Context, session authn.Session, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_tags").Add(1)
		ms.latency.With("method", "update_client_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientTags(ctx, session, client)
}

// UpdateClientIdentity instruments UpdateClientIdentity method with metrics.
func (ms *metricsMiddleware) UpdateClientIdentity(ctx context.Context, session authn.Session, id, identity string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_identity").Add(1)
		ms.latency.With("method", "update_client_identity").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientIdentity(ctx, session, id, identity)
}

// UpdateClientSecret instruments UpdateClientSecret method with metrics.
func (ms *metricsMiddleware) UpdateClientSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_secret").Add(1)
		ms.latency.With("method", "update_client_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
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
func (ms *metricsMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reset_secret").Add(1)
		ms.latency.With("method", "reset_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ResetSecret(ctx, session, secret)
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
func (ms *metricsMiddleware) UpdateClientRole(ctx context.Context, session authn.Session, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_role").Add(1)
		ms.latency.With("method", "update_client_role").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateClientRole(ctx, session, client)
}

// EnableClient instruments EnableClient method with metrics.
func (ms *metricsMiddleware) EnableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_client").Add(1)
		ms.latency.With("method", "enable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableClient(ctx, session, id)
}

// DisableClient instruments DisableClient method with metrics.
func (ms *metricsMiddleware) DisableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_client").Add(1)
		ms.latency.With("method", "disable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableClient(ctx, session, id)
}

// ListMembers instruments ListMembers method with metrics.
func (ms *metricsMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm mgclients.Page) (mp mgclients.MembersPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_members").Add(1)
		ms.latency.With("method", "list_members").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListMembers(ctx, session, objectKind, objectID, pm)
}

// Identify instruments Identify method with metrics.
func (ms *metricsMiddleware) Identify(ctx context.Context, session authn.Session) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Identify(ctx, session)
}

// OAuthCallback instruments OAuthCallback method with metrics.
func (ms *metricsMiddleware) OAuthCallback(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "oauth_callback").Add(1)
		ms.latency.With("method", "oauth_callback").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.OAuthCallback(ctx, client)
}

// DeleteClient instruments DeleteClient method with metrics.
func (ms *metricsMiddleware) DeleteClient(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_client").Add(1)
		ms.latency.With("method", "delete_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteClient(ctx, session, id)
}

// OAuthAddClientPolicy instruments OAuthAddClientPolicy method with metrics.
func (ms *metricsMiddleware) OAuthAddClientPolicy(ctx context.Context, client mgclients.Client) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_client_policy").Add(1)
		ms.latency.With("method", "add_client_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.OAuthAddClientPolicy(ctx, client)
}
