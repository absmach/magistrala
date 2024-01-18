// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

// CreateGroup logs the create_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) CreateGroup(ctx context.Context, token, kind string, group groups.Group) (g groups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group for %s %s  with id %s using token %s took %s to complete", g.Name, kind, g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.CreateGroup(ctx, token, kind, group)
}

// UpdateGroup logs the update_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, group groups.Group) (g groups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group for group %s with id %s using token %s took %s to complete", g.Name, g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateGroup(ctx, token, group)
}

// ViewGroup logs the view_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group for group %s with id %s using token %s took %s to complete", g.Name, g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ViewGroup(ctx, token, id)
}

// ViewGroupPerms logs the view_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewGroupPerms(ctx context.Context, token, id string) (p []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_perms for group with id %s using token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ViewGroupPerms(ctx, token, id)
}

// ListGroups logs the list_groups request. It logs the token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListGroups(ctx context.Context, token, memberKind, memberID string, gp groups.Page) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups %d groups using token %s took %s to complete", cg.Total, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListGroups(ctx, token, memberKind, memberID, gp)
}

// EnableGroup logs the enable_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method enable_group for group with id %s using token %s took %s to complete", g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.EnableGroup(ctx, token, id)
}

// DisableGroup logs the disable_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DisableGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disable_group for group with id %s using token %s took %s to complete", g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DisableGroup(ctx, token, id)
}

// ListMembers logs the list_members request. It logs the groupID and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (mp groups.MembersPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_memberships for group with id %s using token %s took %s to complete", groupID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListMembers(ctx, token, groupID, permission, memberKind)
}

func (lm *loggingMiddleware) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign for token %s and member %s group id %s took %s to complete", token, memberIDs, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Assign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (lm *loggingMiddleware) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign for token %s and member %s group id %s took %s to complete", token, memberIDs, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unassign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (lm *loggingMiddleware) DeleteGroup(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_group for group with id %s using token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeleteGroup(ctx, token, id)
}
