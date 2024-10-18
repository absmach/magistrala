// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
)

var _ roles.Roles = (*RolesSvcLoggingMiddleware)(nil)

type RolesSvcLoggingMiddleware struct {
	svcName string
	svc     roles.Roles
	logger  *slog.Logger
}

func NewRolesSvcLoggingMiddleware(svcName string, svc roles.Roles, logger *slog.Logger) RolesSvcLoggingMiddleware {
	return RolesSvcLoggingMiddleware{
		svcName: svcName,
		svc:     svc,
		logger:  logger,
	}
}

func (lm *RolesSvcLoggingMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (ro roles.Role, err error) {
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

func (lm *RolesSvcLoggingMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	prefix := fmt.Sprintf("Delete %s role", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_delete_role",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveRole(ctx, session, entityID, roleName)
}

func (lm *RolesSvcLoggingMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (ro roles.Role, err error) {
	prefix := fmt.Sprintf("Update %s role name", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_update_role_name",
				slog.String("entity_id", entityID),
				slog.String("old_role_name", oldRoleName),
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
	return lm.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}

func (lm *RolesSvcLoggingMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (ro roles.Role, err error) {
	prefix := fmt.Sprintf("Retrieve %s role", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_update_role_name",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveRole(ctx, session, entityID, roleName)
}

func (lm *RolesSvcLoggingMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (rp roles.RolePage, err error) {
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

func (lm *RolesSvcLoggingMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) (acts []string, err error) {
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

func (lm *RolesSvcLoggingMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (caps []string, err error) {
	prefix := fmt.Sprintf("%s role add actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_actions",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
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
	return lm.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}

func (lm *RolesSvcLoggingMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) (roOps []string, err error) {
	prefix := fmt.Sprintf("%s role list actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_list_role_actions",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleListActions(ctx, session, entityID, roleName)
}

func (lm *RolesSvcLoggingMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return lm.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}

func (lm *RolesSvcLoggingMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	prefix := fmt.Sprintf("%s role remove actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_actions",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
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
	return lm.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}

func (lm *RolesSvcLoggingMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	prefix := fmt.Sprintf("%s role remove all actions", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_all_actions",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}

func (lm *RolesSvcLoggingMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (mems []string, err error) {
	prefix := fmt.Sprintf("%s role add members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_members",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
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
	return lm.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}

func (lm *RolesSvcLoggingMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (mp roles.MembersPage, err error) {
	prefix := fmt.Sprintf("%s role list members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_add_members",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
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
	return lm.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}

func (lm *RolesSvcLoggingMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return lm.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}

func (lm *RolesSvcLoggingMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	prefix := fmt.Sprintf("%s role remove members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_members",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
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
	return lm.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}

func (lm *RolesSvcLoggingMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	prefix := fmt.Sprintf("%s role remove all members", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_role_remove_all_members",
				slog.String("entity_id", entityID),
				slog.String("role_name", roleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}

func (lm *RolesSvcLoggingMiddleware) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	prefix := fmt.Sprintf("%s remove members from all roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_members_from_all_roles",
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
	return lm.svc.RemoveMembersFromAllRoles(ctx, session, members)
}
func (lm *RolesSvcLoggingMiddleware) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	prefix := fmt.Sprintf("%s remove members from roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_members_from_roles",
				slog.Any("members", members),
				slog.Any("roleNames", roleNames),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveMembersFromRoles(ctx, session, members, roleNames)
}
func (lm *RolesSvcLoggingMiddleware) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	prefix := fmt.Sprintf("%s remove actions from all roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_actions_from_all_roles",
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
	return lm.svc.RemoveActionsFromAllRoles(ctx, session, actions)
}
func (lm *RolesSvcLoggingMiddleware) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	prefix := fmt.Sprintf("%s remove actions from roles", lm.svcName)
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(lm.svcName+"_remove_actions_from_roles",
				slog.Any("actions", actions),
				slog.Any("roleNames", roleNames),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(prefix+" failed", args...)
			return
		}
		lm.logger.Info(prefix+" completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveActionsFromRoles(ctx, session, actions, roleNames)
}
