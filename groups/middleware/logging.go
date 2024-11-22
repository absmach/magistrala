// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/pkg/authn"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
)

var _ groups.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    groups.Service
	rmMW.RoleManagerLoggingMiddleware
}

// LoggingMiddleware adds logging facilities to the groups service.
func LoggingMiddleware(svc groups.Service, logger *slog.Logger) groups.Service {
	return &loggingMiddleware{logger, svc, rmMW.NewRoleManagerLoggingMiddleware("groups", svc, logger)}
}

// CreateGroup logs the create_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) CreateGroup(ctx context.Context, session authn.Session, group groups.Group) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", g.ID),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Create group failed", args...)
			return
		}
		lm.logger.Info("Create group completed successfully", args...)
	}(time.Now())
	return lm.svc.CreateGroup(ctx, session, group)
}

// UpdateGroup logs the update_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, session authn.Session, group groups.Group) (g groups.Group, err error) {
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
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update group failed", args...)
			return
		}
		lm.logger.Info("Update group completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateGroup(ctx, session, group)
}

// ViewGroup logs the view_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", g.ID),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View group failed", args...)
			return
		}
		lm.logger.Info("View group completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewGroup(ctx, session, id)
}

// ListGroups logs the list_groups request. It logs the page metadata and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListGroups(ctx context.Context, session authn.Session, pm groups.PageMeta) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cg.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List groups failed", args...)
			return
		}
		lm.logger.Info("List groups completed successfully", args...)
	}(time.Now())
	return lm.svc.ListGroups(ctx, session, pm)
}

func (lm *loggingMiddleware) ListUserGroups(ctx context.Context, session authn.Session, userID string, pm groups.PageMeta) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", userID),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cg.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List user groups failed", args...)
			return
		}
		lm.logger.Info("List user groups completed successfully", args...)
	}(time.Now())
	return lm.svc.ListUserGroups(ctx, session, userID, pm)
}

// EnableGroup logs the enable_group request. It logs the group name, id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", id),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable group failed", args...)
			return
		}
		lm.logger.Info("Enable group completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableGroup(ctx, session, id)
}

// DisableGroup logs the disable_group request. It logs the group id and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DisableGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("group",
				slog.String("id", id),
				slog.String("name", g.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable group failed", args...)
			return
		}
		lm.logger.Info("Disable group completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableGroup(ctx, session, id)
}

func (lm *loggingMiddleware) DeleteGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Delete group failed", args...)
			return
		}
		lm.logger.Info("Delete group completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteGroup(ctx, session, id)
}

func (lm *loggingMiddleware) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (gp groups.HierarchyPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
			slog.Group("page",
				slog.Uint64("limit", hm.Level),
				slog.Int64("offset", hm.Direction),
				slog.Bool("total", hm.Tree),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Retrieve group hierarchy failed", args...)
			return
		}
		lm.logger.Info("Retrieve group hierarchy completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveGroupHierarchy(ctx, session, id, hm)
}

func (lm *loggingMiddleware) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
			slog.String("parent_group_id", parentID),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Add parent group failed", args...)
			return
		}
		lm.logger.Info("Add parent group completed successfully", args...)
	}(time.Now())
	return lm.svc.AddParentGroup(ctx, session, id, parentID)
}

func (lm *loggingMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove parent group failed", args...)
			return
		}
		lm.logger.Info("Remove parent group completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveParentGroup(ctx, session, id)
}

func (lm *loggingMiddleware) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
			slog.Any("children_group_ids", childrenGroupIDs),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Add children groups failed", args...)
			return
		}
		lm.logger.Info("Add parent group completed successfully", args...)
	}(time.Now())
	return lm.svc.AddChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (lm *loggingMiddleware) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
			slog.Any("children_group_ids", childrenGroupIDs),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove children groups failed", args...)
			return
		}
		lm.logger.Info("Remove parent group completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (lm *loggingMiddleware) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove all children groups failed", args...)
			return
		}
		lm.logger.Info("Remove all parent group completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveAllChildrenGroups(ctx, session, id)
}

func (lm *loggingMiddleware) ListChildrenGroups(ctx context.Context, session authn.Session, id string, startLevel, endLevel int64, pm groups.PageMeta) (gp groups.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("group_id", id),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", gp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List children groups failed", args...)
			return
		}
		lm.logger.Info("List children groups completed successfully", args...)
	}(time.Now())
	return lm.svc.ListChildrenGroups(ctx, session, id, startLevel, endLevel, pm)
}
