package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
	"go.opentelemetry.io/otel/trace"
)

var _ roles.Roles = (*RolesSvcTracingMiddleware)(nil)

type RolesSvcTracingMiddleware struct {
	svcName string
	roles   roles.Roles
	tracer  trace.Tracer
}

func NewRolesSvcTracingMiddleware(svcName string, svc roles.Roles, tracer trace.Tracer) RolesSvcTracingMiddleware {
	return RolesSvcTracingMiddleware{svcName, svc, tracer}
}

func (rtm *RolesSvcTracingMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return rtm.roles.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (rtm *RolesSvcTracingMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rtm.roles.RemoveRole(ctx, session, entityID, roleName)
}
func (rtm *RolesSvcTracingMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return rtm.roles.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (rtm *RolesSvcTracingMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return rtm.roles.RetrieveRole(ctx, session, entityID, roleName)
}
func (rtm *RolesSvcTracingMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return rtm.roles.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (rtm *RolesSvcTracingMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return rtm.roles.ListAvailableActions(ctx, session)
}
func (rtm *RolesSvcTracingMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error) {
	return rtm.roles.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (rtm *RolesSvcTracingMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return rtm.roles.RoleListActions(ctx, session, entityID, roleName)
}
func (rtm *RolesSvcTracingMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return rtm.roles.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (rtm *RolesSvcTracingMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return rtm.roles.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (rtm *RolesSvcTracingMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return rtm.roles.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (rtm *RolesSvcTracingMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return rtm.roles.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (rtm *RolesSvcTracingMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return rtm.roles.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (rtm *RolesSvcTracingMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return rtm.roles.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (rtm *RolesSvcTracingMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return rtm.roles.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (rtm *RolesSvcTracingMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return rtm.roles.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (rtm *RolesSvcTracingMiddleware) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	return rtm.roles.RemoveMembersFromAllRoles(ctx, session, members)
}
func (rtm *RolesSvcTracingMiddleware) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	return rtm.roles.RemoveMembersFromRoles(ctx, session, members, roleNames)
}
func (rtm *RolesSvcTracingMiddleware) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	return rtm.roles.RemoveActionsFromAllRoles(ctx, session, actions)
}
func (rtm *RolesSvcTracingMiddleware) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	return rtm.roles.RemoveActionsFromRoles(ctx, session, actions, roleNames)
}
