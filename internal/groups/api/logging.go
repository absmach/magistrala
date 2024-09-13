// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/groups"
)

var _ groups.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    groups.Service
}

// LoggingMiddleware adds logging facilities to the groups service.
func LoggingMiddleware(svc groups.Service, logger *slog.Logger) groups.Service {
	return &loggingMiddleware{logger, svc}
}

// CreateGroup logs the create_group request. It logs the group name, id and session and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) CreateGroup(ctx context.Context, session auth.Session, kind string, group groups.Group) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", g.ID),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create group failed", args...)
			return
		}
		lm.logger.Info("Create group completed successfully", args...)
	}(time.Now())
	return lm.svc.CreateGroup(ctx, session, kind, group)
}

// UpdateGroup logs the update_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, session auth.Session, group groups.Group) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", group.ID),
				slog.String("name", group.Name),
				slog.Any("metadata", group.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update group failed", args...)
			return
		}
		lm.logger.Info("Update group completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateGroup(ctx, session, group)
}

// ViewGroup logs the view_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewGroup(ctx context.Context, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", g.ID),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View group failed", args...)
			return
		}
		lm.logger.Info("View group completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewGroup(ctx, id)
}

// ViewGroupPerms logs the view_group request. It logs the group id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewGroupPerms(ctx context.Context, session auth.Session, id string) (p []string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View group permissions failed", args...)
			return
		}
		lm.logger.Info("View group permissions completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewGroupPerms(ctx, session, id)
}

// ListGroups logs the list_groups request. It logs the page metadata and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListGroups(ctx context.Context, session auth.Session, memberKind, memberID string, gp groups.Page) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("member",
				slog.String("id", memberID),
				slog.String("kind", memberKind),
			),
			slog.Group("page",
				slog.Uint64("limit", gp.Limit),
				slog.Uint64("offset", gp.Offset),
				slog.Uint64("total", cg.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List groups failed", args...)
			return
		}
		lm.logger.Info("List groups completed successfully", args...)
	}(time.Now())
	return lm.svc.ListGroups(ctx, session, memberKind, memberID, gp)
}

// EnableGroup logs the enable_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableGroup(ctx context.Context, session auth.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", id),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Enable group failed", args...)
			return
		}
		lm.logger.Info("Enable group completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableGroup(ctx, session, id)
}

// DisableGroup logs the disable_group request. It logs the group id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DisableGroup(ctx context.Context, session auth.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", id),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disable group failed", args...)
			return
		}
		lm.logger.Info("Disable group completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableGroup(ctx, session, id)
}

// ListMembers logs the list_members request. It logs the groupID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListMembers(ctx context.Context, groupID, permission, memberKind string) (mp groups.MembersPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", groupID),
			slog.String("permission", permission),
			slog.String("member_kind", memberKind),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List members failed", args...)
			return
		}
		lm.logger.Info("List members completed successfully", args...)
	}(time.Now())
	return lm.svc.ListMembers(ctx, groupID, permission, memberKind)
}

func (lm *loggingMiddleware) Assign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", groupID),
			slog.String("relation", relation),
			slog.String("member_kind", memberKind),
			slog.Any("member_ids", memberIDs),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Assign member to group failed", args...)
			return
		}
		lm.logger.Info("Assign member to group completed successfully", args...)
	}(time.Now())

	return lm.svc.Assign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

func (lm *loggingMiddleware) Unassign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", groupID),
			slog.String("relation", relation),
			slog.String("member_kind", memberKind),
			slog.Any("member_ids", memberIDs),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Unassign member to group failed", args...)
			return
		}
		lm.logger.Info("Unassign member to group completed successfully", args...)
	}(time.Now())

	return lm.svc.Unassign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

func (lm *loggingMiddleware) DeleteGroup(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete group failed", args...)
			return
		}
		lm.logger.Info("Delete group completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteGroup(ctx, id)
}
