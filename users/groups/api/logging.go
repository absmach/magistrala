// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/users/groups"
)

var _ groups.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    groups.Service
}

// LoggingMiddleware adds logging facilities to the groups service.
func LoggingMiddleware(svc groups.Service, logger mflog.Logger) groups.Service {
	return &loggingMiddleware{logger, svc}
}

// CreateGroup logs the create_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) CreateGroup(ctx context.Context, token string, group mfgroups.Group) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group for group %s with id %s using token %s took %s to complete", g.Name, g.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.CreateGroup(ctx, token, group)
}

// UpdateGroup logs the update_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, group mfgroups.Group) (g mfgroups.Group, err error) {
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
func (lm *loggingMiddleware) ViewGroup(ctx context.Context, token, id string) (g mfgroups.Group, err error) {
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

// ListGroups logs the list_groups request. It logs the token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListGroups(ctx context.Context, token string, gp mfgroups.GroupsPage) (cg mfgroups.GroupsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups %d groups using token %s took %s to complete", cg.Total, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListGroups(ctx, token, gp)
}

// EnableGroup logs the enable_group request. It logs the group name, id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableGroup(ctx context.Context, token, id string) (g mfgroups.Group, err error) {
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
func (lm *loggingMiddleware) DisableGroup(ctx context.Context, token, id string) (g mfgroups.Group, err error) {
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

// ListMemberships logs the list_memberships request. It logs the clientID and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListMemberships(ctx context.Context, token, clientID string, cp mfgroups.GroupsPage) (mp mfgroups.MembershipsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_memberships for client with id %s using token %s took %s to complete", clientID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListMemberships(ctx, token, clientID, cp)
}
