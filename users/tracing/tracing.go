// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	users "github.com/absmach/magistrala/users"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ users.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    users.Service
}

// New returns a new group service with tracing capabilities.
func New(svc users.Service, tracer trace.Tracer) users.Service {
	return &tracingMiddleware{tracer, svc}
}

// RegisterUser traces the "RegisterUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) RegisterUser(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_user", trace.WithAttributes(attribute.String("identity", user.Identity)))
	defer span.End()

	return tm.svc.RegisterUser(ctx, session, user, selfRegister)
}

// IssueToken traces the "IssueToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_issue_token", trace.WithAttributes(attribute.String("identity", identity)))
	defer span.End()

	return tm.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken traces the "RefreshToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_refresh_token", trace.WithAttributes(attribute.String("refresh_token", refreshToken)))
	defer span.End()

	return tm.svc.RefreshToken(ctx, session, refreshToken, domainID)
}

// ViewUser traces the "ViewUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ViewUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.ViewUser(ctx, session, id)
}

// ListUsers traces the "ListUsers" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_users", trace.WithAttributes(
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
	ctx, span := tm.tracer.Start(ctx, "svc_search_users", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))
	defer span.End()

	return tm.svc.SearchUsers(ctx, pm)
}

// UpdateUser traces the "UpdateUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUser(ctx context.Context, session authn.Session, cli users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_name_and_metadata", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.String("first_name", cli.FirstName),
		attribute.String("last_name", cli.LastName),
	))
	defer span.End()

	return tm.svc.UpdateUser(ctx, session, cli)
}

// UpdateUserTags traces the "UpdateUserTags" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUserTags(ctx context.Context, session authn.Session, cli users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateUserTags(ctx, session, cli)
}

// UpdateUserIdentity traces the "UpdateUserIdentity" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUserIdentity(ctx context.Context, session authn.Session, id, identity string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_identity", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("identity", identity),
	))
	defer span.End()

	return tm.svc.UpdateUserIdentity(ctx, session, id, identity)
}

// UpdateUserSecret traces the "UpdateUserSecret" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_secret")
	defer span.End()

	return tm.svc.UpdateUserSecret(ctx, session, oldSecret, newSecret)
}

// UpdateUserFullName traces the "UpdateUserFullName" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUserNames(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_names", trace.WithAttributes(
		attribute.String("id", user.ID),
		attribute.String("first_name", user.FirstName),
		attribute.String("last_name", user.LastName),
		attribute.String("user_name", user.Credentials.UserName),
	))
	defer span.End()

	return tm.svc.UpdateUserNames(ctx, session, user)
}

// UpdateProfilePicture traces the "UpdateProfilePicture" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, usr users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_profile_picture", trace.WithAttributes(attribute.String("id", usr.ID)))
	defer span.End()

	return tm.svc.UpdateUser(ctx, session, usr)
}

// GenerateResetToken traces the "GenerateResetToken" operation of the wrapped users.Service.
func (tm *tracingMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_generate_reset_token", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("host", host),
	))
	defer span.End()

	return tm.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret traces the "ResetSecret" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_reset_secret")
	defer span.End()

	return tm.svc.ResetSecret(ctx, session, secret)
}

// SendPasswordReset traces the "SendPasswordReset" operation of the wrapped users.Service.
func (tm *tracingMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_send_password_reset", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("user", user),
	))
	defer span.End()

	return tm.svc.SendPasswordReset(ctx, host, email, user, token)
}

// ViewProfile traces the "ViewProfile" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_profile")
	defer span.End()

	return tm.svc.ViewProfile(ctx, session)
}

func (tm *tracingMiddleware) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_user_by_username", trace.WithAttributes(attribute.String("username", userName)))
	defer span.End()

	return tm.svc.ViewUserByUserName(ctx, session, userName)
}

// UpdateUserRole traces the "UpdateUserRole" operation of the wrapped users.Service.
func (tm *tracingMiddleware) UpdateUserRole(ctx context.Context, session authn.Session, cli users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user_role", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateUserRole(ctx, session, cli)
}

// EnableUser traces the "EnableUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) EnableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableUser(ctx, session, id)
}

// DisableUser traces the "DisableUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) DisableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableUser(ctx, session, id)
}

// ListMembers traces the "ListMembers" operation of the wrapped users.Service.
func (tm *tracingMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm users.Page) (users.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_members", trace.WithAttributes(attribute.String("object_kind", objectKind)), trace.WithAttributes(attribute.String("object_id", objectID)))
	defer span.End()

	return tm.svc.ListMembers(ctx, session, objectKind, objectID, pm)
}

// Identify traces the "Identify" operation of the wrapped users.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, session authn.Session) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("user_id", session.UserID)))
	defer span.End()

	return tm.svc.Identify(ctx, session)
}

// OAuthCallback traces the "OAuthCallback" operation of the wrapped users.Service.
func (tm *tracingMiddleware) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_oauth_callback", trace.WithAttributes(
		attribute.String("user_id", user.ID),
	))
	defer span.End()

	return tm.svc.OAuthCallback(ctx, user)
}

// DeleteUser traces the "DeleteUser" operation of the wrapped users.Service.
func (tm *tracingMiddleware) DeleteUser(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_delete_user", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DeleteUser(ctx, session, id)
}

// OAuthAddUserPolicy traces the "OAuthAddUserPolicy" operation of the wrapped users.Service.
func (tm *tracingMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	ctx, span := tm.tracer.Start(ctx, "svc_add_user_policy", trace.WithAttributes(
		attribute.String("id", user.ID),
	))
	defer span.End()

	return tm.svc.OAuthAddUserPolicy(ctx, user)
}
