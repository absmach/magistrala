// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
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

// RegisterUser instruments RegisterUser method with metrics.
func (ms *metricsMiddleware) RegisterUser(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_user").Add(1)
		ms.latency.With("method", "register_user").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RegisterUser(ctx, session, user, selfRegister)
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

// ViewUser instruments ViewUser method with metrics.
func (ms *metricsMiddleware) ViewUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_user").Add(1)
		ms.latency.With("method", "view_user").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewUser(ctx, session, id)
}

// ViewProfile instruments ViewProfile method with metrics.
func (ms *metricsMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewProfile(ctx, session)
}

// ViewUserByUserName instruments ViewUserByUserName method with metrics.
func (ms *metricsMiddleware) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_user_by_username").Add(1)
		ms.latency.With("method", "view_user_by_username").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewUserByUserName(ctx, session, userName)
}

// ListUsers instruments ListUsers method with metrics.
func (ms *metricsMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_users").Add(1)
		ms.latency.With("method", "list_users").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListUsers(ctx, session, pm)
}

// SearchUsers instruments SearchUsers method with metrics.
func (ms *metricsMiddleware) SearchUsers(ctx context.Context, pm users.Page) (mp users.UsersPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "search_users").Add(1)
		ms.latency.With("method", "search_users").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.SearchUsers(ctx, pm)
}

// UpdateUser instruments UpdateUser method with metrics.
func (ms *metricsMiddleware) UpdateUser(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_name_and_metadata").Add(1)
		ms.latency.With("method", "update_user_name_and_metadata").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUser(ctx, session, user)
}

// UpdateUserTags instruments UpdateUserTags method with metrics.
func (ms *metricsMiddleware) UpdateUserTags(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_tags").Add(1)
		ms.latency.With("method", "update_user_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUserTags(ctx, session, user)
}

// UpdateUserIdentity instruments UpdateUserIdentity method with metrics.
func (ms *metricsMiddleware) UpdateUserIdentity(ctx context.Context, session authn.Session, id, identity string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_identity").Add(1)
		ms.latency.With("method", "update_user_identity").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUserIdentity(ctx, session, id, identity)
}

// UpdateUserSecret instruments UpdateUserSecret method with metrics.
func (ms *metricsMiddleware) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_secret").Add(1)
		ms.latency.With("method", "update_user_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUserSecret(ctx, session, oldSecret, newSecret)
}

// UpdateUserNames instruments UpdateUserNames method with metrics.
func (ms *metricsMiddleware) UpdateUserNames(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_names").Add(1)
		ms.latency.With("method", "update_user_names").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUserNames(ctx, session, user)
}

// UpdateProfilePicture instruments UpdateProfilePicture method with metrics.
func (ms *metricsMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_profile_picture").Add(1)
		ms.latency.With("method", "update_profile_picture").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUser(ctx, session, user)
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

// UpdateUserRole instruments UpdateUserRole method with metrics.
func (ms *metricsMiddleware) UpdateUserRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user_role").Add(1)
		ms.latency.With("method", "update_user_role").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateUserRole(ctx, session, user)
}

// EnableUser instruments EnableUser method with metrics.
func (ms *metricsMiddleware) EnableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_user").Add(1)
		ms.latency.With("method", "enable_user").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableUser(ctx, session, id)
}

// DisableUser instruments DisableUser method with metrics.
func (ms *metricsMiddleware) DisableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_user").Add(1)
		ms.latency.With("method", "disable_user").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableUser(ctx, session, id)
}

// ListMembers instruments ListMembers method with metrics.
func (ms *metricsMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm users.Page) (mp users.MembersPage, err error) {
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
func (ms *metricsMiddleware) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "oauth_callback").Add(1)
		ms.latency.With("method", "oauth_callback").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.OAuthCallback(ctx, user)
}

// DeleteUser instruments DeleteUser method with metrics.
func (ms *metricsMiddleware) DeleteUser(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_user").Add(1)
		ms.latency.With("method", "delete_user").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteUser(ctx, session, id)
}

// OAuthAddUserPolicy instruments OAuthAddUserPolicy method with metrics.
func (ms *metricsMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_user_policy").Add(1)
		ms.latency.With("method", "add_user_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.OAuthAddUserPolicy(ctx, user)
}
