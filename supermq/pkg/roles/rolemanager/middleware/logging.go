// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/roles"
)

var _ roles.RoleManager = (*RoleManagerLoggingMiddleware)(nil)

type RoleManagerLoggingMiddleware struct {
	svcName string
	svc     roles.RoleManager
	logger  *slog.Logger
}

func NewRoleManagerLoggingMiddleware(svcName string, svc roles.RoleManager, logger *slog.Logger) RoleManagerLoggingMiddleware {
	return RoleManagerLoggingMiddleware{
		svcName: svcName,
		svc:     svc,
		logger:  logger,
	}
}

func (lm *RoleManagerLoggingMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (ro roles.RoleProvision, err error) {
	prefix := fmt.Sprintf("Add %s roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_add_role",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
				slog.Any("optional_actions", optionalActions),
				slog.Any("optional_members", optionalMembers),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}

func (lm *RoleManagerLoggingMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleID string) (err error) {
	prefix := fmt.Sprintf("Delete %s role", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_delete_role",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveRole(ctx, session, entityID, roleID)
}

func (lm *RoleManagerLoggingMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, roleID, newRoleName string) (ro roles.Role, err error) {
	prefix := fmt.Sprintf("Update %s role name", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_update_role_name",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.String("new_role_name", newRoleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateRoleName(ctx, session, entityID, roleID, newRoleName)
}

func (lm *RoleManagerLoggingMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleID string) (ro roles.Role, err error) {
	prefix := fmt.Sprintf("Retrieve %s role", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_retrieve_role",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveRole(ctx, session, entityID, roleID)
}

func (lm *RoleManagerLoggingMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (rp roles.RolePage, err error) {
	prefix := fmt.Sprintf("List %s roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_roles_retrieve_all",
				slog.String("entity_id", entityID),
				slog.Uint64("limit", limit),
				slog.Uint64("offset", offset),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}

func (lm *RoleManagerLoggingMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) (acts []string, err error) {
	prefix := fmt.Sprintf("List %s available actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName + "_list_available_actions"),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.ListAvailableActions(ctx, session)
}

func (lm *RoleManagerLoggingMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (caps []string, err error) {
	prefix := fmt.Sprintf("%s role add actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_actions",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.Any("actions", actions),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleAddActions(ctx, session, entityID, roleID, actions)
}

func (lm *RoleManagerLoggingMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleID string) (roOps []string, err error) {
	prefix := fmt.Sprintf("%s role list actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_list_role_actions",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleListActions(ctx, session, entityID, roleID)
}

func (lm *RoleManagerLoggingMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (bool, error) {
	return lm.svc.RoleCheckActionsExists(ctx, session, entityID, roleID, actions)
}

func (lm *RoleManagerLoggingMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (err error) {
	prefix := fmt.Sprintf("%s role remove actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_actions",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.Any("actions", actions),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveActions(ctx, session, entityID, roleID, actions)
}

func (lm *RoleManagerLoggingMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleID string) (err error) {
	prefix := fmt.Sprintf("%s role remove all actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_all_actions",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveAllActions(ctx, session, entityID, roleID)
}

func (lm *RoleManagerLoggingMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (mems []string, err error) {
	prefix := fmt.Sprintf("%s role add members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_members",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.Any("members", members),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleAddMembers(ctx, session, entityID, roleID, members)
}

func (lm *RoleManagerLoggingMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleID string, limit, offset uint64) (mp roles.MembersPage, err error) {
	prefix := fmt.Sprintf("%s role list members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_members",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.Uint64("limit", limit),
				slog.Uint64("offset", offset),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleListMembers(ctx, session, entityID, roleID, limit, offset)
}

func (lm *RoleManagerLoggingMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (bool, error) {
	return lm.svc.RoleCheckMembersExists(ctx, session, entityID, roleID, members)
}

func (lm *RoleManagerLoggingMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (err error) {
	prefix := fmt.Sprintf("%s role remove members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_members",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
				slog.Any("members", members),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveMembers(ctx, session, entityID, roleID, members)
}

func (lm *RoleManagerLoggingMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleID string) (err error) {
	prefix := fmt.Sprintf("%s role remove all members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_all_members",
				slog.String("entity_id", entityID),
				slog.String("role_id", roleID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveAllMembers(ctx, session, entityID, roleID)
}

func (lm *RoleManagerLoggingMiddleware) ListEntityMembers(ctx context.Context, session authn.Session, entityID string, pageQuery roles.MembersRolePageQuery) (mems roles.MembersRolePage, err error) {
	prefix := fmt.Sprintf("%s list entity members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_entity_members",
				slog.String("entity_id", entityID),
				slog.Uint64("limit", pageQuery.Limit),
				slog.Uint64("offset", pageQuery.Offset),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.ListEntityMembers(ctx, session, entityID, pageQuery)
}

func (lm *RoleManagerLoggingMiddleware) RemoveEntityMembers(ctx context.Context, session authn.Session, entityID string, members []string) (err error) {
	prefix := fmt.Sprintf("%s remove entity members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_entity_members",
				slog.String("entity_id", entityID),
				slog.Any("member_ids", members),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveEntityMembers(ctx, session, entityID, members)
}

func (lm *RoleManagerLoggingMiddleware) RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, memberID string) (err error) {
	prefix := fmt.Sprintf("%s remove members from all roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_members_from_all_roles",
				slog.Any("member_id", memberID),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveMemberFromAllRoles(ctx, session, memberID)
}
