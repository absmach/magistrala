// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

type AuthzReq struct {
	ChannelID  string
	ThingID    string
	ThingKey   string
	Permission string
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateThings creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateThings(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, error)

	// ViewClient retrieves client info for a given client ID and an authorized token.
	ViewClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error)

	// UpdateClient updates the client's name and metadata.
	UpdateClient(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientTags updates the client's tags.
	UpdateClientTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientSecret updates the client's secret
	UpdateClientSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error)

	// EnableClient logically enableds the client identified with the provided ID
	EnableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// DisableClient logically disables the client identified with the provided ID
	DisableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// DeleteClient deletes client with given ID.
	DeleteClient(ctx context.Context, session authn.Session, id string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// Authorize used for Things authorization.
	Authorize(ctx context.Context, req AuthzReq) (string, error)

	// SetParentGroup(ctx context.Context, token string, parentGroupID string, id string) error

	// RemoveParentGroup(ctx context.Context, token string, parentGroupID string, id string) error

	roles.Roles
}

// Cache contains thing caching interface.
//
//go:generate mockery --name Cache --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	// Save stores pair thing secret, thing id.
	Save(ctx context.Context, thingSecret, thingID string) error

	// ID returns thing ID for given thing secret.
	ID(ctx context.Context, thingSecret string) (string, error)

	// Removes thing from cache.
	Remove(ctx context.Context, thingID string) error
}

const (
	OpCreateThing svcutil.Operation = iota
	OpListThing
	OpViewThing
	OpUpdateThing
	OpUpdateClientTags
	OpUpdateClientSecret
	OpEnableThing
	OpDisableThing
	OpDeleteThing
)

var expectedOperations = []svcutil.Operation{
	OpCreateThing,
	OpListThing,
	OpViewThing,
	OpUpdateThing,
	OpUpdateClientTags,
	OpUpdateClientSecret,
	OpEnableThing,
	OpDisableThing,
	OpDeleteThing,
}

var operationNames = []string{
	"OpCreateThing",
	"OpListThing",
	"OpViewThing",
	"OpUpdateThing",
	"OpUpdateClientTags",
	"OpUpdateClientSecret",
	"OpEnableThing",
	"OpDisableThing",
	"OpDeleteThing",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	// this permission is check over domain or group
	createPermission = "thing_create_permission"
	// this permission is check over domain or group
	listPermissions = "thing_list_permission"

	updatePermission           = "update_permission"
	readPermission             = "read_permission"
	deletePermission           = "delete_permission"
	setParentGroupPermission   = "set_parent_group_permission"
	connectToChannelPermission = "connect_to_channel_permission"

	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpCreateThing:        createPermission,
		OpListThing:          listPermissions,
		OpViewThing:          readPermission,
		OpUpdateThing:        updatePermission,
		OpUpdateClientTags:   updatePermission,
		OpUpdateClientSecret: updatePermission,
		OpEnableThing:        updatePermission,
		OpDisableThing:       updatePermission,
		OpDeleteThing:        deletePermission,
	}
	return opPerm
}

func NewRolesOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		roles.OpAddRole:                manageRolePermission,
		roles.OpRemoveRole:             manageRolePermission,
		roles.OpUpdateRoleName:         manageRolePermission,
		roles.OpRetrieveRole:           manageRolePermission,
		roles.OpRetrieveAllRoles:       manageRolePermission,
		roles.OpRoleAddActions:         manageRolePermission,
		roles.OpRoleListActions:        manageRolePermission,
		roles.OpRoleCheckActionsExists: manageRolePermission,
		roles.OpRoleRemoveActions:      manageRolePermission,
		roles.OpRoleRemoveAllActions:   manageRolePermission,
		roles.OpRoleAddMembers:         addRoleUsersPermission,
		roles.OpRoleListMembers:        viewRoleUsersPermission,
		roles.OpRoleCheckMembersExists: viewRoleUsersPermission,
		roles.OpRoleRemoveMembers:      removeRoleUsersPermission,
		roles.OpRoleRemoveAllMembers:   manageRolePermission,
	}
	return opPerm
}
