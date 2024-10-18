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

var _ roles.Roles = (*RolesSvcMetricsMiddleware)(nil)

type RolesSvcMetricsMiddleware struct {
	svcName string
	svc     roles.Roles
	counter metrics.Counter
	latency metrics.Histogram
}

func NewRolesSvcMetricsMiddleware(svcName string, svc roles.Roles, counter metrics.Counter, latency metrics.Histogram) RolesSvcMetricsMiddleware {
	return RolesSvcMetricsMiddleware{
		svcName: svcName,
		svc:     svc,
		counter: counter,
		latency: latency,
	}
}

func (rmm *RolesSvcMetricsMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return rmm.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (rmm *RolesSvcMetricsMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rmm.svc.RemoveRole(ctx, session, entityID, roleName)
}
func (rmm *RolesSvcMetricsMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return rmm.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (rmm *RolesSvcMetricsMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return rmm.svc.RetrieveRole(ctx, session, entityID, roleName)
}
func (rmm *RolesSvcMetricsMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return rmm.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (rmm *RolesSvcMetricsMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return rmm.svc.ListAvailableActions(ctx, session)
}
func (rmm *RolesSvcMetricsMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (caps []string, err error) {
	return rmm.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (rmm *RolesSvcMetricsMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return rmm.svc.RoleListActions(ctx, session, entityID, roleName)
}
func (rmm *RolesSvcMetricsMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return rmm.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (rmm *RolesSvcMetricsMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return rmm.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (rmm *RolesSvcMetricsMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rmm.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (rmm *RolesSvcMetricsMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return rmm.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (rmm *RolesSvcMetricsMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return rmm.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (rmm *RolesSvcMetricsMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return rmm.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (rmm *RolesSvcMetricsMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return rmm.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (rmm *RolesSvcMetricsMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return rmm.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (rmm *RolesSvcMetricsMiddleware) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	return rmm.svc.RemoveMembersFromAllRoles(ctx, session, members)
}
func (rmm *RolesSvcMetricsMiddleware) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	return rmm.svc.RemoveMembersFromRoles(ctx, session, members, roleNames)
}
func (rmm *RolesSvcMetricsMiddleware) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	return rmm.svc.RemoveActionsFromAllRoles(ctx, session, actions)
}
func (rmm *RolesSvcMetricsMiddleware) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	return rmm.svc.RemoveActionsFromRoles(ctx, session, actions, roleNames)
}
