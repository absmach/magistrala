// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/auth"
	log "github.com/mainflux/mainflux/logger"
)

var _ auth.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    auth.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc auth.Service, logger log.Logger) auth.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) ListPolicies(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_policies took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListPolicies(ctx, pr)
}

func (lm *loggingMiddleware) Issue(ctx context.Context, token string, newKey auth.Key) (key auth.Key, secret string, err error) {
	defer func(begin time.Time) {
		d := "infinite duration"
		if !key.ExpiresAt.IsZero() {
			d = fmt.Sprintf("the key with expiration date %v", key.ExpiresAt)
		}
		message := fmt.Sprintf("Method issue for %s took %s to complete", d, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Issue(ctx, token, newKey)
}

func (lm *loggingMiddleware) Revoke(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Revoke(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveKey(ctx context.Context, token, id string) (key auth.Key, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method retrieve for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RetrieveKey(ctx, token, id)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id auth.Identity, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Authorize(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.AddPolicy(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_policy_bulk took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddPolicies(ctx, token, object, subjectIDs, relations)
}

func (lm *loggingMiddleware) DeletePolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policy took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicy(ctx, pr)
}

func (lm *loggingMiddleware) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policies took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicies(ctx, token, object, subjectIDs, relations)
}

func (lm *loggingMiddleware) CreateGroup(ctx context.Context, token string, group auth.Group) (g auth.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_group for token %s and name %s took %s to complete", token, group.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateGroup(ctx, token, group)
}

func (lm *loggingMiddleware) UpdateGroup(ctx context.Context, token string, group auth.Group) (gr auth.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group for token %s and name %s took %s to complete", token, group.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateGroup(ctx, token, group)
}

func (lm *loggingMiddleware) RemoveGroup(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group for token %s and id %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ViewGroup(ctx context.Context, token, id string) (group auth.Group, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group for token %s and id %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroup(ctx, token, id)
}

func (lm *loggingMiddleware) ListGroups(ctx context.Context, token string, pm auth.PageMetadata) (gp auth.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_groups for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroups(ctx, token, pm)
}

func (lm *loggingMiddleware) ListChildren(ctx context.Context, token, parentID string, pm auth.PageMetadata) (gp auth.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_children for token %s and parent %s took %s to complete", token, parentID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChildren(ctx, token, parentID, pm)
}

func (lm *loggingMiddleware) ListParents(ctx context.Context, token, childID string, pm auth.PageMetadata) (gp auth.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_parents for token %s and child %s took for child %s to complete", token, childID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListParents(ctx, token, childID, pm)
}

func (lm *loggingMiddleware) ListMembers(ctx context.Context, token, groupID, groupType string, pm auth.PageMetadata) (gp auth.MemberPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_members for token %s and group id %s took %s to complete", token, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListMembers(ctx, token, groupID, groupType, pm)
}

func (lm *loggingMiddleware) ListMemberships(ctx context.Context, token, memberID string, pm auth.PageMetadata) (gp auth.GroupPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_memberships for token %s and member id %s took %s to complete", token, memberID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListMemberships(ctx, token, memberID, pm)
}

func (lm *loggingMiddleware) Assign(ctx context.Context, token, groupID, groupType string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign for token %s and member %s group id %s took %s to complete", token, memberIDs, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Assign(ctx, token, groupID, groupType, memberIDs...)
}

func (lm *loggingMiddleware) Unassign(ctx context.Context, token string, groupID string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign for token %s and member %s group id %s took %s to complete", token, memberIDs, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unassign(ctx, token, groupID, memberIDs...)
}

func (lm *loggingMiddleware) AssignGroupAccessRights(ctx context.Context, token, thingGroupID, userGroupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method share_group_access took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignGroupAccessRights(ctx, token, thingGroupID, userGroupID)
}
