// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/go-kit/kit/metrics"
)

var _ roles.RoleManager = (*RoleManagerMetricsMiddleware)(nil)

type RoleManagerMetricsMiddleware struct {
	svcName string
	svc     roles.RoleManager
	counter metrics.Counter
	latency metrics.Histogram
}

func NewRoleManagerMetricsMiddleware(svcName string, svc roles.RoleManager, counter metrics.Counter, latency metrics.Histogram) RoleManagerMetricsMiddleware {
	return RoleManagerMetricsMiddleware{
		svcName: svcName,
		svc:     svc,
		counter: counter,
		latency: latency,
	}
}

func (rmm *RoleManagerMetricsMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return rmm.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (rmm *RoleManagerMetricsMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rmm.svc.RemoveRole(ctx, session, entityID, roleName)
}
func (rmm *RoleManagerMetricsMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return rmm.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (rmm *RoleManagerMetricsMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return rmm.svc.RetrieveRole(ctx, session, entityID, roleName)
}
func (rmm *RoleManagerMetricsMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return rmm.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (rmm *RoleManagerMetricsMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return rmm.svc.ListAvailableActions(ctx, session)
}
func (rmm *RoleManagerMetricsMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (caps []string, err error) {
	return rmm.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (rmm *RoleManagerMetricsMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return rmm.svc.RoleListActions(ctx, session, entityID, roleName)
}
func (rmm *RoleManagerMetricsMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return rmm.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (rmm *RoleManagerMetricsMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return rmm.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (rmm *RoleManagerMetricsMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rmm.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (rmm *RoleManagerMetricsMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return rmm.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (rmm *RoleManagerMetricsMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return rmm.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (rmm *RoleManagerMetricsMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return rmm.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (rmm *RoleManagerMetricsMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return rmm.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (rmm *RoleManagerMetricsMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return rmm.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (rmm *RoleManagerMetricsMiddleware) RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, memberID string) (err error) {
	return rmm.svc.RemoveMemberFromAllRoles(ctx, session, memberID)
}
