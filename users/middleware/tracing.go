// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/tracing"
	users "github.com/absmach/supermq/users"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ users.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    users.Service
}

// NewTracing returns a new users service with tracing capabilities.
func NewTracing(svc users.Service, tracer trace.Tracer) users.Service {
	return &tracingMiddleware{tracer, svc}
}

// Register traces the "Register" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Register(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_register_user", trace.WithAttributes(attribute.String("email", user.Email)))
	defer span.End()

	return tm.svc.Register(ctx, session, user, selfRegister)
}

// SendVerification traces the "SendVerification" operation of the wrapped users.Service.
func (tm *tracingMiddleware) SendVerification(ctx context.Context, session authn.Session) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_send_verification")
	defer span.End()

	return tm.svc.SendVerification(ctx, session)
}

// VerifyEmail traces the "VerifyEmail" operation of the wrapped users.Service.
func (tm *tracingMiddleware) VerifyEmail(ctx context.Context, verificationToken string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_verify_email")
	defer span.End()
	return tm.svc.VerifyEmail(ctx, verificationToken)
}

// IssueToken traces the "IssueToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) IssueToken(ctx context.Context, username, secret, description string) (*grpcTokenV1.Token, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_issue_token", trace.WithAttributes(attribute.String("username", username)))
	defer span.End()

	return tm.svc.IssueToken(ctx, username, secret, description)
}

// RefreshToken traces the "RefreshToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_refresh_token", trace.WithAttributes(attribute.String("refresh_token", refreshToken)))
	defer span.End()

	return tm.svc.RefreshToken(ctx, session, refreshToken)
}

// RevokeRefreshToken traces the "RevokeRefreshToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) RevokeRefreshToken(ctx context.Context, session authn.Session, tokenID string) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_revoke_refresh_token")
	defer span.End()

	return tm.svc.RevokeRefreshToken(ctx, session, tokenID)
}

// ListActiveRefreshTokens traces the "ListActiveRefreshTokens" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ListActiveRefreshTokens(ctx context.Context, session authn.Session) (*grpcTokenV1.ListUserRefreshTokensRes, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_list_active_refresh_tokens")
	defer span.End()

	return tm.svc.ListActiveRefreshTokens(ctx, session)
}

// View traces the "View" operation of the wrapped users.Service.
func (tm *tracingMiddleware) View(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_view_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.View(ctx, session, id)
}

// ListUsers traces the "ListUsers" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_list_users", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))

	defer span.End()

	return tm.svc.ListUsers(ctx, session, pm)
}

// SearchUsers traces the "SearchUsers" operation of the wrapped users.Service.
func (tm *tracingMiddleware) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_search_users", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))
	defer span.End()

	return tm.svc.SearchUsers(ctx, pm)
}

// Update traces the "Update" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Update(ctx context.Context, session authn.Session, id string, user users.UserReq) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_user", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.Update(ctx, session, id, user)
}

// UpdateTags traces the "UpdateTags" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateTags(ctx context.Context, session authn.Session, id string, user users.UserReq) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_user_tags", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.UpdateTags(ctx, session, id, user)
}

// UpdateEmail traces the "UpdateEmail" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateEmail(ctx context.Context, session authn.Session, id, email string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_user_email", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("email", email),
	))
	defer span.End()

	return tm.svc.UpdateEmail(ctx, session, id, email)
}

// UpdateSecret traces the "UpdateSecret" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_user_secret")
	defer span.End()

	return tm.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

// UpdateUsername traces the "UpdateUsername" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_usernames", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("username", username),
	))
	defer span.End()

	return tm.svc.UpdateUsername(ctx, session, id, username)
}

// UpdateProfilePicture traces the "UpdateProfilePicture" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, id string, usr users.UserReq) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_profile_picture", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.UpdateProfilePicture(ctx, session, id, usr)
}

// SendPasswordReset traces the "SendPasswordReset" operation of the wrapped users.Service.
func (tm *tracingMiddleware) SendPasswordReset(ctx context.Context, email string) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_send_password_reset", trace.WithAttributes(
		attribute.String("email", email),
	))
	defer span.End()

	return tm.svc.SendPasswordReset(ctx, email)
}

// ResetSecret traces the "ResetSecret" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_reset_secret")
	defer span.End()

	return tm.svc.ResetSecret(ctx, session, secret)
}

// ViewProfile traces the "ViewProfile" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_view_profile")
	defer span.End()

	return tm.svc.ViewProfile(ctx, session)
}

// UpdateRole traces the "UpdateRole" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateRole(ctx context.Context, session authn.Session, cli users.User) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_update_user_role", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateRole(ctx, session, cli)
}

// Enable traces the "Enable" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Enable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_enable_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.Enable(ctx, session, id)
}

// Disable traces the "Disable" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Disable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_disable_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.Disable(ctx, session, id)
}

// Identify traces the "Identify" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, session authn.Session) (string, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_identify", trace.WithAttributes(attribute.String("user_id", session.UserID)))
	defer span.End()

	return tm.svc.Identify(ctx, session)
}

// OAuthCallback traces the "OAuthCallback" operation of the wrapped users.Service.
func (tm *tracingMiddleware) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_oauth_callback", trace.WithAttributes(
		attribute.String("user_id", user.ID),
	))
	defer span.End()

	return tm.svc.OAuthCallback(ctx, user)
}

// Delete traces the "Delete" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_delete_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.Delete(ctx, session, id)
}

// OAuthAddUserPolicy traces the "OAuthAddUserPolicy" operation of the wrapped users.Service.
func (tm *tracingMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "svc_add_user_policy", trace.WithAttributes(
		attribute.String("id", user.ID),
	))
	defer span.End()

	return tm.svc.OAuthAddUserPolicy(ctx, user)
}
