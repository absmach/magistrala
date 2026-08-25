// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
)

const (
	GroupsKind     = "groups"
	NewGroupKind   = "new_group"
	ChannelsKind   = "channels"
	NewChannelKind = "new_channel"
	DevicesKind    = "devices"
	NewDeviceKind  = "new_device"
	UsersKind      = "users"
	WorkspacesKind = "workspaces"
	PlatformKind   = "platform"
)

const (
	RoleType      = "role"
	GroupType     = "group"
	DeviceType    = "device"
	ChannelType   = "channel"
	UserType      = "user"
	WorkspaceType = "workspace"
	PlatformType  = "platform"
	RulesType     = "rules"
	ReportsType   = "reports"
	AlarmsType    = "alarms"
)

const (
	AdministratorRelation = "administrator"
	EditorRelation        = "editor"
	ContributorRelation   = "contributor"
	MemberRelation        = "member"
	WorkspaceRelation     = "workspace"
	ParentGroupRelation   = "parent_group"
	RoleGroupRelation     = "role_group"
	GroupRelation         = "group"
	PlatformRelation      = "platform"
	GuestRelation         = "guest"
)

const (
	AdminPermission      = "admin"
	DeletePermission     = "delete"
	EditPermission       = "edit"
	ViewPermission       = "view"
	MembershipPermission = "membership"
	SharePermission      = "share"
	PublishPermission    = "publish"
	SubscribePermission  = "subscribe"
	CreatePermission     = "create"
)

const MagistralaObject = "magistrala"

type Evaluator interface {
	// CheckPolicy checks if the subject has a relation on the object.
	// It returns a non-nil error if the subject has no relation on
	// the object (which simply means the operation is denied).
	CheckPolicy(ctx context.Context, pr Policy) error
}
