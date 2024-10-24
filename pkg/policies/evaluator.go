// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
)

const (
	TokenKind      = "token"
	GroupsKind     = "groups"
	NewGroupKind   = "new_group"
	ChannelsKind   = "channels"
	NewChannelKind = "new_channel"
	ClientsKind    = "clients"
	NewClientKind  = "new_client"
	UsersKind      = "users"
	DomainsKind    = "domains"
	PlatformKind   = "platform"
)

const (
	RoleType     = "role"
	GroupType    = "group"
	ClientType   = "client"
	ChannelType  = "channel"
	UserType     = "user"
	DomainType   = "domain"
	PlatformType = "platform"
)

const (
	AdministratorRelation = "administrator"
	EditorRelation        = "editor"
	ContributorRelation   = "contributor"
	MemberRelation        = "member"
	DomainRelation        = "domain"
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

//go:generate mockery --name Evaluator --output=./mocks --filename evaluator.go --quiet --note "Copyright (c) Abstract Machines"
type Evaluator interface {
	// CheckPolicy checks if the subject has a relation on the object.
	// It returns a non-nil error if the subject has no relation on
	// the object (which simply means the operation is denied).
	CheckPolicy(ctx context.Context, pr Policy) error
}
