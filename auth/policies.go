// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
)

const (
	TokenKind      = "token"
	GroupsKind     = "groups"
	NewGroupKind   = "new_group"
	ChannelsKind   = "channels"
	NewChannelKind = "new_channel"
	ThingsKind     = "things"
	NewThingKind   = "new_thing"
	UsersKind      = "users"
	DomainsKind    = "domains"
	PlatformKind   = "platform"
)

const (
	GroupType    = "group"
	ThingType    = "thing"
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

// PolicyReq represents an argument struct for making policy-related
// function calls. It is used to pass information required for policy
// evaluation and enforcement.
type PolicyReq struct {
	// Domain contains the domain ID.
	Domain string `json:"domain,omitempty"`

	// Subject contains the subject ID or Token.
	Subject string `json:"subject"`

	// SubjectType contains the subject type. Supported subject types are
	// platform, group, domain, thing, users.
	SubjectType string `json:"subject_type"`

	// SubjectKind contains the subject kind. Supported subject kinds are
	// token, users, platform, things, channels, groups, domain.
	SubjectKind string `json:"subject_kind"`

	// SubjectRelation contains subject relations.
	SubjectRelation string `json:"subject_relation,omitempty"`

	// Object contains the object ID.
	Object string `json:"object"`

	// ObjectKind contains the object kind. Supported object kinds are
	// users, platform, things, channels, groups, domain.
	ObjectKind string `json:"object_kind"`

	// ObjectType contains the object type. Supported object types are
	// platform, group, domain, thing, users.
	ObjectType string `json:"object_type"`

	// Relation contains the relation. Supported relations are administrator, editor, contributor, member, guest, parent_group,group,domain.
	Relation string `json:"relation,omitempty"`

	// Permission contains the permission. Supported permissions are admin, delete, edit, share, view,
	// membership, create, admin_only, edit_only, view_only, membership_only, ext_admin, ext_edit, ext_view.
	Permission string `json:"permission,omitempty"`
}

func (pr PolicyReq) String() string {
	data, err := json.Marshal(pr)
	if err != nil {
		return ""
	}
	return string(data)
}

type PolicyRes struct {
	Namespace       string
	Subject         string
	SubjectType     string
	SubjectRelation string
	Object          string
	ObjectType      string
	Relation        string
	Permission      string
}

type PolicyPage struct {
	Policies      []string
	NextPageToken string
}

type Permissions []string

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
//
//go:generate mockery --name Authz --output=./mocks --filename authz.go --quiet --note "Copyright (c) Abstract Machines"
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr PolicyReq) error

	// DeleteUserPolicies deletes all policies for the given user.
	DeleteUserPolicies(ctx context.Context, id string) error
}

// PolicyAgent facilitates the communication to authorization
// services and implements Authz functionalities for certain
// authorization services (e.g. ORY Keto).
//
//go:generate mockery --name PolicyAgent --output=./mocks --filename agent.go --quiet --note "Copyright (c) Abstract Machines"
type PolicyAgent interface {
	// CheckPolicy checks if the subject has a relation on the object.
	// It returns a non-nil error if the subject has no relation on
	// the object (which simply means the operation is denied).
	CheckPolicy(ctx context.Context, pr PolicyReq) error
}
