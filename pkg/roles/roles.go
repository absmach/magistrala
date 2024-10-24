// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/svcutil"
)

type Action string

func (ac Action) String() string {
	return string(ac)
}

type Member string

func (mem Member) String() string {
	return string(mem)
}

type RoleName string

func (r RoleName) String() string {
	return string(r)
}

type BuiltInRoleName RoleName

func (b BuiltInRoleName) ToRoleName() RoleName {
	return RoleName(b)
}

func (b BuiltInRoleName) String() string {
	return string(b)
}

type Role struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	EntityID  string    `json:"entity_id"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RoleProvision struct {
	Role
	OptionalActions []string `json:"-"`
	OptionalMembers []string `json:"-"`
}

type RolePage struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Roles  []Role `json:"roles"`
}

type MembersPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Members []string `json:"members"`
}

type EntityActionRole struct {
	EntityID string `json:"entity_id"`
	Action   string `json:"action"`
	RoleID   string `json:"role_id"`
}
type EntityMemberRole struct {
	EntityID string `json:"entity_id"`
	MemberID string `json:"member_id"`
	RoleID   string `json:"role_id"`
}

//go:generate mockery --name Provisioner --output=./mocks --filename provisioner.go --quiet --note "Copyright (c) Abstract Machines"
type Provisioner interface {
	AddNewEntitiesRoles(ctx context.Context, domainID, userID string, entityIDs []string, optionalEntityPolicies []policies.Policy, newBuiltInRoleMembers map[BuiltInRoleName][]Member) ([]RoleProvision, error)
	RemoveEntitiesRoles(ctx context.Context, domainID, userID string, entityIDs []string, optionalFilterDeletePolicies []policies.Policy, optionalDeletePolicies []policies.Policy) error
}

//go:generate mockery --name RoleManager --output=./mocks --filename rolemanager.go --quiet --note "Copyright (c) Abstract Machines"
type RoleManager interface {

	// Add New role to entity
	AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (Role, error)

	// Remove removes the roles of entity.
	RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error

	// UpdateName update the name of the entity role.
	UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (Role, error)

	RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (Role, error)

	RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (RolePage, error)

	ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error)

	RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error)

	RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error)

	RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error)

	RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error)

	RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error

	RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error)

	RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (MembersPage, error)

	RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error)

	RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error)

	RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error)

	RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, memberID string) (err error)
}

//go:generate mockery --name Repository --output=./mocks --filename rolesRepo.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	AddRoles(ctx context.Context, rps []RoleProvision) ([]Role, error)
	RemoveRoles(ctx context.Context, roleIDs []string) error
	UpdateRole(ctx context.Context, ro Role) (Role, error)
	RetrieveRole(ctx context.Context, roleID string) (Role, error)
	RetrieveRoleByEntityIDAndName(ctx context.Context, entityID, roleName string) (Role, error)
	RetrieveAllRoles(ctx context.Context, entityID string, limit, offset uint64) (RolePage, error)
	RoleAddActions(ctx context.Context, role Role, actions []string) (ops []string, err error)
	RoleListActions(ctx context.Context, roleID string) ([]string, error)
	RoleCheckActionsExists(ctx context.Context, roleID string, actions []string) (bool, error)
	RoleRemoveActions(ctx context.Context, role Role, actions []string) (err error)
	RoleRemoveAllActions(ctx context.Context, role Role) error
	RoleAddMembers(ctx context.Context, role Role, members []string) ([]string, error)
	RoleListMembers(ctx context.Context, roleID string, limit, offset uint64) (MembersPage, error)
	RoleCheckMembersExists(ctx context.Context, roleID string, members []string) (bool, error)
	RoleRemoveMembers(ctx context.Context, role Role, members []string) (err error)
	RoleRemoveAllMembers(ctx context.Context, role Role) (err error)
	RetrieveEntitiesRolesActionsMembers(ctx context.Context, entityIDs []string) ([]EntityActionRole, []EntityMemberRole, error)
	RemoveMemberFromAllRoles(ctx context.Context, members string) (err error)
}

type Roles interface {

	// Add New role to entity
	AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (Role, error)

	// Remove removes the roles of entity.
	RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error

	// UpdateName update the name of the entity role.
	UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (Role, error)

	RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (Role, error)

	RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (RolePage, error)

	ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error)

	RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error)

	RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error)

	RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error)

	RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error)

	RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error

	RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error)

	RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (MembersPage, error)

	RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error)

	RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error)

	RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error)

	RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error)

	RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error)

	RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error)

	RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error)
}

const (
	OpAddRole svcutil.Operation = iota
	OpRemoveRole
	OpUpdateRoleName
	OpRetrieveRole
	OpRetrieveAllRoles
	OpRoleAddActions
	OpRoleListActions
	OpRoleCheckActionsExists
	OpRoleRemoveActions
	OpRoleRemoveAllActions
	OpRoleAddMembers
	OpRoleListMembers
	OpRoleCheckMembersExists
	OpRoleRemoveMembers
	OpRoleRemoveAllMembers
)

var expectedOperations = []svcutil.Operation{
	OpAddRole,
	OpRemoveRole,
	OpUpdateRoleName,
	OpRetrieveRole,
	OpRetrieveAllRoles,
	OpRoleAddActions,
	OpRoleListActions,
	OpRoleCheckActionsExists,
	OpRoleRemoveActions,
	OpRoleRemoveAllActions,
	OpRoleAddMembers,
	OpRoleListMembers,
	OpRoleCheckMembersExists,
	OpRoleRemoveMembers,
	OpRoleRemoveAllMembers,
}

var operationNames = []string{
	"OpAddRole",
	"OpRemoveRole",
	"OpUpdateRoleName",
	"OpRetrieveRole",
	"OpRetrieveAllRoles",
	"OpRoleAddActions",
	"OpRoleListActions",
	"OpRoleCheckActionsExists",
	"OpRoleRemoveActions",
	"OpRoleRemoveAllActions",
	"OpRoleAddMembers",
	"OpRoleListMembers",
	"OpRoleCheckMembersExists",
	"OpRoleRemoveMembers",
	"OpRoleRemoveAllMembers",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}
