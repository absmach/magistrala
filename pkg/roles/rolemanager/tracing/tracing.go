package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
	"go.opentelemetry.io/otel/trace"
)

var _ roles.RoleManager = (*RoleManagerTracing)(nil)

type RoleManagerTracing struct {
	svcName string
	roles   roles.RoleManager
	tracer  trace.Tracer
}

func NewRoleManagerTracing(svcName string, svc roles.RoleManager, tracer trace.Tracer) RoleManagerTracing {
	return RoleManagerTracing{svcName, svc, tracer}
}

func (rtm *RoleManagerTracing) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return rtm.roles.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (rtm *RoleManagerTracing) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rtm.roles.RemoveRole(ctx, session, entityID, roleName)
}
func (rtm *RoleManagerTracing) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return rtm.roles.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (rtm *RoleManagerTracing) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return rtm.roles.RetrieveRole(ctx, session, entityID, roleName)
}
func (rtm *RoleManagerTracing) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return rtm.roles.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (rtm *RoleManagerTracing) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return rtm.roles.ListAvailableActions(ctx, session)
}
func (rtm *RoleManagerTracing) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error) {
	return rtm.roles.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (rtm *RoleManagerTracing) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return rtm.roles.RoleListActions(ctx, session, entityID, roleName)
}
func (rtm *RoleManagerTracing) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return rtm.roles.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (rtm *RoleManagerTracing) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return rtm.roles.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (rtm *RoleManagerTracing) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rtm.roles.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (rtm *RoleManagerTracing) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return rtm.roles.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (rtm *RoleManagerTracing) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return rtm.roles.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (rtm *RoleManagerTracing) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return rtm.roles.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (rtm *RoleManagerTracing) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return rtm.roles.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (rtm *RoleManagerTracing) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return rtm.roles.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (rtm *RoleManagerTracing) RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, memberID string) (err error) {
	return rtm.roles.RemoveMemberFromAllRoles(ctx, session, memberID)
}
