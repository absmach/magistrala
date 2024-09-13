// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ auth.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    auth.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc auth.Service, logger *slog.Logger) auth.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Issue(ctx context.Context, token string, key auth.Key) (tkn auth.Token, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("key",
				slog.String("subject", key.Subject),
				slog.Any("type", key.Type),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Issue key failed", args...)
			return
		}
		lm.logger.Info("Issue key completed successfully", args...)
	}(time.Now())

	return lm.svc.Issue(ctx, token, key)
}

func (lm *loggingMiddleware) Revoke(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("key_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Revoke key failed", args...)
			return
		}
		lm.logger.Info("Revoke key completed successfully", args...)
	}(time.Now())

	return lm.svc.Revoke(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveKey(ctx context.Context, token, id string) (key auth.Key, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("key_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve key failed", args...)
			return
		}
		lm.logger.Info("Retrieve key completed successfully", args...)
	}(time.Now())

	return lm.svc.RetrieveKey(ctx, token, id)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, token string) (id auth.Key, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("key",
				slog.String("subject", id.Subject),
				slog.Any("type", id.Type),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Identify key failed", args...)
			return
		}
		lm.logger.Info("Identify key completed successfully", args...)
	}(time.Now())

	return lm.svc.Identify(ctx, token)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("object",
				slog.String("id", pr.Object),
				slog.String("type", pr.ObjectType),
			),
			slog.Group("subject",
				slog.String("id", pr.Subject),
				slog.String("kind", pr.SubjectKind),
				slog.String("type", pr.SubjectType),
			),
			slog.String("permission", pr.Permission),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Authorize failed", args...)
			return
		}
		lm.logger.Info("Authorize completed successfully", args...)
	}(time.Now())
	return lm.svc.Authorize(ctx, pr)
}

func (lm *loggingMiddleware) CreateDomain(ctx context.Context, token string, d auth.Domain) (do auth.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("domain",
				slog.String("id", d.ID),
				slog.String("name", d.Name),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Create domain failed", args...)
			return
		}
		lm.logger.Info("Create domain completed successfully", args...)
	}(time.Now())
	return lm.svc.CreateDomain(ctx, token, d)
}

func (lm *loggingMiddleware) RetrieveDomain(ctx context.Context, token, id string) (do auth.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve domain failed", args...)
			return
		}
		lm.logger.Info("Retrieve domain completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveDomain(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveDomainPermissions(ctx context.Context, token, id string) (permissions policies.Permissions, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve domain permissions failed", args...)
			return
		}
		lm.logger.Info("Retrieve domain permissions completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveDomainPermissions(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateDomain(ctx context.Context, token, id string, d auth.DomainReq) (do auth.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("domain",
				slog.String("id", id),
				slog.Any("name", d.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update domain failed", args...)
			return
		}
		lm.logger.Info("Update domain completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateDomain(ctx, token, id, d)
}

func (lm *loggingMiddleware) ChangeDomainStatus(ctx context.Context, token, id string, d auth.DomainReq) (do auth.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("domain",
				slog.String("id", id),
				slog.String("name", do.Name),
				slog.Any("status", d.Status),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Change domain status failed", args...)
			return
		}
		lm.logger.Info("Change domain status completed successfully", args...)
	}(time.Now())
	return lm.svc.ChangeDomainStatus(ctx, token, id, d)
}

func (lm *loggingMiddleware) ListDomains(ctx context.Context, token string, page auth.Page) (do auth.DomainsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("limit", page.Limit),
				slog.Uint64("offset", page.Offset),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List domains failed", args...)
			return
		}
		lm.logger.Info("List domains completed successfully", args...)
	}(time.Now())
	return lm.svc.ListDomains(ctx, token, page)
}

func (lm *loggingMiddleware) AssignUsers(ctx context.Context, token, id string, userIds []string, relation string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
			slog.String("relation", relation),
			slog.Any("user_ids", userIds),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Assign users to domain failed", args...)
			return
		}
		lm.logger.Info("Assign users to domain completed successfully", args...)
	}(time.Now())
	return lm.svc.AssignUsers(ctx, token, id, userIds, relation)
}

func (lm *loggingMiddleware) UnassignUser(ctx context.Context, token, id, userID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
			slog.Any("user_id", userID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Unassign user from domain failed", args...)
			return
		}
		lm.logger.Info("Unassign user from domain completed successfully", args...)
	}(time.Now())
	return lm.svc.UnassignUser(ctx, token, id, userID)
}

func (lm *loggingMiddleware) ListUserDomains(ctx context.Context, token, userID string, page auth.Page) (do auth.DomainsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", userID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List user domains failed", args...)
			return
		}
		lm.logger.Info("List user domains completed successfully", args...)
	}(time.Now())
	return lm.svc.ListUserDomains(ctx, token, userID, page)
}

func (lm *loggingMiddleware) DeleteUserPolicies(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete entity policies failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Delete entity policies completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteUserPolicies(ctx, id)
}
