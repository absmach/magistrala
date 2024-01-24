// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/auth"
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

func (lm *loggingMiddleware) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Int64("limit", int64(limit)),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List objects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List objects completed successfully", args...)
	}(time.Now())

	return lm.svc.ListObjects(ctx, pr, nextPageToken, limit)
}

func (lm *loggingMiddleware) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("policy_request",
				slog.String("object_type", pr.ObjectType),
				slog.String("subject_id", pr.Subject),
				slog.String("subject_type", pr.SubjectType),
				slog.String("permission", pr.Permission),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List all objects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List all objects completed successfully", args...)
	}(time.Now())

	return lm.svc.ListAllObjects(ctx, pr)
}

func (lm *loggingMiddleware) CountObjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Count objects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Count objects completed successfully", args...)
	}(time.Now())
	return lm.svc.CountObjects(ctx, pr)
}

func (lm *loggingMiddleware) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List subjects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List subjects completed successfully", args...)
	}(time.Now())

	return lm.svc.ListSubjects(ctx, pr, nextPageToken, limit)
}

func (lm *loggingMiddleware) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("policy_request",
				slog.String("sybject_type", pr.SubjectType),
				slog.String("object_id", pr.Object),
				slog.String("object_type", pr.ObjectType),
				slog.String("permission", pr.Permission),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List all subjects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List all subjects completed successfully", args...)
	}(time.Now())

	return lm.svc.ListAllSubjects(ctx, pr)
}

func (lm *loggingMiddleware) CountSubjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Count subjects failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Count subjects completed successfully", args...)
	}(time.Now())
	return lm.svc.CountSubjects(ctx, pr)
}

func (lm *loggingMiddleware) ListPermissions(ctx context.Context, pr auth.PolicyReq, filterPermissions []string) (p auth.Permissions, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Any("filter_permissions", filterPermissions),
			slog.Group("policy_request",
				slog.String("object_id", pr.Object),
				slog.String("object_type", pr.ObjectType),
				slog.String("subject_id", pr.Subject),
				slog.String("subject_type", pr.SubjectType),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List permissions failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List permissions completed successfully", args...)
	}(time.Now())

	return lm.svc.ListPermissions(ctx, pr, filterPermissions)
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
			lm.logger.Warn("Issue key failed to complete successfully", args...)
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
			lm.logger.Warn("Revoke key failed to complete successfully", args...)
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
			lm.logger.Warn("Retrieve key failed to complete successfully", args...)
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
			lm.logger.Warn("Identify key failed to complete successfully", args...)
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
			lm.logger.Warn("Authorize failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Authorize completed successfully", args...)
	}(time.Now())
	return lm.svc.Authorize(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("policy_request",
				slog.String("object_id", pr.Object),
				slog.String("object_type", pr.ObjectType),
				slog.String("subject_id", pr.Subject),
				slog.String("subject_type", pr.SubjectType),
				slog.String("relation", pr.Relation),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Add policy failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Add policy completed successfully", args...)
	}(time.Now())
	return lm.svc.AddPolicy(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicies(ctx context.Context, prs []auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn(fmt.Sprintf("Add %d policies failed to complete successfully", len(prs)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Add %d policies completed successfully", len(prs)), args...)
	}(time.Now())

	return lm.svc.AddPolicies(ctx, prs)
}

func (lm *loggingMiddleware) DeletePolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("policy_request",
				slog.String("object_id", pr.Object),
				slog.String("object_type", pr.ObjectType),
				slog.String("subject_id", pr.Subject),
				slog.String("subject_type", pr.SubjectType),
				slog.String("relation", pr.Relation),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete policy failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Delete policy completed successfully", args...)
	}(time.Now())
	return lm.svc.DeletePolicy(ctx, pr)
}

func (lm *loggingMiddleware) DeletePolicies(ctx context.Context, prs []auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn(fmt.Sprintf("Delete %d policies failed to complete successfully", len(prs)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Delete %d policies completed successfully", len(prs)), args...)
	}(time.Now())
	return lm.svc.DeletePolicies(ctx, prs)
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
			lm.logger.Warn("Create domain failed to complete successfully", args...)
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
			lm.logger.Warn("Retrieve domain failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Retrieve domain completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveDomain(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveDomainPermissions(ctx context.Context, token, id string) (permissions auth.Permissions, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve domain permissions failed to complete successfully", args...)
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
			lm.logger.Warn("Update domain failed to complete successfully", args...)
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
			lm.logger.Warn("Change domain status failed to complete successfully", args...)
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
			lm.logger.Warn("List domains failed to complete successfully", args...)
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
			lm.logger.Warn("Assign users to domain failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Assign users to domain completed successfully", args...)
	}(time.Now())
	return lm.svc.AssignUsers(ctx, token, id, userIds, relation)
}

func (lm *loggingMiddleware) UnassignUsers(ctx context.Context, token, id string, userIds []string, relation string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", id),
			slog.String("relation", relation),
			slog.Any("user_ids", userIds),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Unassign users to domain failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Unassign users to domain completed successfully", args...)
	}(time.Now())
	return lm.svc.UnassignUsers(ctx, token, id, userIds, relation)
}

func (lm *loggingMiddleware) ListUserDomains(ctx context.Context, token, userID string, page auth.Page) (do auth.DomainsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", userID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List user domains failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List user domains completed successfully", args...)
	}(time.Now())
	return lm.svc.ListUserDomains(ctx, token, userID, page)
}
