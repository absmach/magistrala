// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/users"
)

var _ users.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    users.Service
}

// LoggingMiddleware adds logging facilities to the users service.
func LoggingMiddleware(svc users.Service, logger *slog.Logger) users.Service {
	return &loggingMiddleware{logger, svc}
}

// RegisterUser logs the user request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RegisterUser(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (u users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", u.ID),
				slog.String("user_name", u.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Register user failed", args...)
			return
		}
		lm.logger.Info("Register user completed successfully", args...)
	}(time.Now())
	return lm.svc.RegisterUser(ctx, session, user, selfRegister)
}

// IssueToken logs the issue_token request. It logs the user identity type and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (t *magistrala.Token, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", domainID),
		}
		if t.AccessType != "" {
			args = append(args, slog.String("access_type", t.AccessType))
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Issue token failed", args...)
			return
		}
		lm.logger.Info("Issue token completed successfully", args...)
	}(time.Now())
	return lm.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken logs the refresh_token request. It logs the refreshtoken, token type and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (t *magistrala.Token, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", domainID),
		}
		if t.AccessType != "" {
			args = append(args, slog.String("access_type", t.AccessType))
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Refresh token failed", args...)
			return
		}
		lm.logger.Info("Refresh token completed successfully", args...)
	}(time.Now())
	return lm.svc.RefreshToken(ctx, session, refreshToken, domainID)
}

// ViewUser logs the view_user request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewUser(ctx context.Context, session authn.Session, id string) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", id),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View user failed", args...)
			return
		}
		lm.logger.Info("View user completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewUser(ctx, session, id)
}

// ViewProfile logs the view_profile request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewProfile(ctx context.Context, session authn.Session) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", c.ID),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View profile failed", args...)
			return
		}
		lm.logger.Info("View profile completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewProfile(ctx, session)
}

// ViewUserByUserName logs the view_user_by_username request. It logs the user name and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (u users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_name", userName),
			slog.Group("user",
				slog.String("id", u.ID),
				slog.String("user_name", u.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View user by username failed", args...)
			return
		}
		lm.logger.Info("View user by username completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewUserByUserName(ctx, session, userName)
}

// ListUsers logs the list_users request. It logs the page metadata and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (cp users.UsersPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List users failed", args...)
			return
		}
		lm.logger.Info("List users completed successfully", args...)
	}(time.Now())
	return lm.svc.ListUsers(ctx, session, pm)
}

// SearchUsers logs the search_users request. It logs the page metadata and the time it took to complete the request.
func (lm *loggingMiddleware) SearchUsers(ctx context.Context, cp users.Page) (mp users.UsersPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("limit", cp.Limit),
				slog.Uint64("offset", cp.Offset),
				slog.Uint64("total", mp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Search users failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Search users completed successfully", args...)
	}(time.Now())
	return lm.svc.SearchUsers(ctx, cp)
}

// UpdateUser logs the update_user request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUser(ctx context.Context, session authn.Session, user users.User) (u users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", u.ID),
				slog.String("user_name", u.Credentials.UserName),
				slog.Any("metadata", u.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user failed", args...)
			return
		}
		lm.logger.Info("Update user completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUser(ctx, session, user)
}

// UpdateUserTags logs the update_user_tags request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUserTags(ctx context.Context, session authn.Session, user users.User) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", c.ID),
				slog.String("user_name", c.Credentials.UserName),
				slog.Any("tags", c.Tags),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user tags failed", args...)
			return
		}
		lm.logger.Info("Update user tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUserTags(ctx, session, user)
}

// UpdateUserIdentity logs the update_identity request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUserIdentity(ctx context.Context, session authn.Session, id, identity string) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", c.ID),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user identity failed", args...)
			return
		}
		lm.logger.Info("Update user identity completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUserIdentity(ctx, session, id, identity)
}

// UpdateUserSecret logs the update_user_secret request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", c.ID),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user secret failed", args...)
			return
		}
		lm.logger.Info("Update user secret completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUserSecret(ctx, session, oldSecret, newSecret)
}

// UpdateUserNames logs the update_user_names request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUserNames(ctx context.Context, session authn.Session, user users.User) (u users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", u.ID),
				slog.String("first_name", u.FirstName),
				slog.String("last_name", u.LastName),
				slog.String("user_name", u.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user name failed", args...)
			return
		}
		lm.logger.Info("Update user name completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUserNames(ctx, session, user)
}

// UpdateProfilePicture logs the update_profile_picture request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (u users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", u.ID),
				slog.String("user_name", u.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update profile picture failed", args...)
			return
		}
		lm.logger.Info("Update profile picture completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUser(ctx, session, user)
}

// GenerateResetToken logs the generate_reset_token request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) GenerateResetToken(ctx context.Context, email, host string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("host", host),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Generate reset token failed", args...)
			return
		}
		lm.logger.Info("Generate reset token completed successfully", args...)
	}(time.Now())
	return lm.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret logs the reset_secret request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Reset secret failed", args...)
			return
		}
		lm.logger.Info("Reset secret completed successfully", args...)
	}(time.Now())
	return lm.svc.ResetSecret(ctx, session, secret)
}

// SendPasswordReset logs the send_password_reset request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("host", host),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Send password reset failed", args...)
			return
		}
		lm.logger.Info("Send password reset completed successfully", args...)
	}(time.Now())
	return lm.svc.SendPasswordReset(ctx, host, email, user, token)
}

// UpdateUserRole logs the update_user_role request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateUserRole(ctx context.Context, session authn.Session, user users.User) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", c.ID),
				slog.String("user_name", c.Credentials.UserName),
				slog.String("role", user.Role.String()),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update user role failed", args...)
			return
		}
		lm.logger.Info("Update user role completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateUserRole(ctx, session, user)
}

// EnableUser logs the enable_user request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableUser(ctx context.Context, session authn.Session, id string) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", id),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Enable user failed", args...)
			return
		}
		lm.logger.Info("Enable user completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableUser(ctx, session, id)
}

// DisableUser logs the disable_user request. It logs the user id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DisableUser(ctx context.Context, session authn.Session, id string) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("user",
				slog.String("id", id),
				slog.String("user_name", c.Credentials.UserName),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disable user failed", args...)
			return
		}
		lm.logger.Info("Disable user completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableUser(ctx, session, id)
}

// ListMembers logs the list_members request. It logs the group id, and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, cp users.Page) (mp users.MembersPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("object",
				slog.String("kind", objectKind),
				slog.String("id", objectID),
			),
			slog.Group("page",
				slog.Uint64("limit", cp.Limit),
				slog.Uint64("offset", cp.Offset),
				slog.Uint64("total", mp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List members failed", args...)
			return
		}
		lm.logger.Info("List members completed successfully", args...)
	}(time.Now())
	return lm.svc.ListMembers(ctx, session, objectKind, objectID, cp)
}

// Identify logs the identify request. It logs the time it took to complete the request.
func (lm *loggingMiddleware) Identify(ctx context.Context, session authn.Session) (id string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Identify user failed", args...)
			return
		}
		lm.logger.Info("Identify user completed successfully", args...)
	}(time.Now())
	return lm.svc.Identify(ctx, session)
}

func (lm *loggingMiddleware) OAuthCallback(ctx context.Context, user users.User) (c users.User, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", user.ID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("OAuth callback failed", args...)
			return
		}
		lm.logger.Info("OAuth callback completed successfully", args...)
	}(time.Now())
	return lm.svc.OAuthCallback(ctx, user)
}

// DeleteUser logs the delete_user request. It logs the user id and token and the time it took to complete the request.
func (lm *loggingMiddleware) DeleteUser(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete user failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Delete user completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteUser(ctx, session, id)
}

// OAuthAddUserPolicy logs the add_user_policy request. It logs the user id and the time it took to complete the request.
func (lm *loggingMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", user.ID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Add user policy failed", args...)
			return
		}
		lm.logger.Info("Add user policy completed successfully", args...)
	}(time.Now())
	return lm.svc.OAuthAddUserPolicy(ctx, user)
}
