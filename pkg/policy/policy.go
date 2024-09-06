// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policy

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

// PolicyClient facilitates the communication to authorization
// services and implements Authz functionalities for spicedb
//
//go:generate mockery --name PolicyClient --filename client.go --quiet --note "Copyright (c) Abstract Machines"
type PolicyClient interface {
	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, prs []PolicyReq) error

	// DeletePolicyFilter removes policy for given policy filter request.
	DeletePolicyFilter(ctx context.Context, pr PolicyReq) error

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, prs []PolicyReq) error

	// ListObjects lists policies based on the given PolicyReq structure.
	ListObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit uint64) (PolicyPage, error)

	// ListAllObjects lists all policies based on the given PolicyReq structure.
	ListAllObjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountObjects count policies based on the given PolicyReq structure.
	CountObjects(ctx context.Context, pr PolicyReq) (uint64, error)

	// ListSubjects lists subjects based on the given PolicyReq structure.
	ListSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit uint64) (PolicyPage, error)

	// ListAllSubjects lists all subjects based on the given PolicyReq structure.
	ListAllSubjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountSubjects count policies based on the given PolicyReq structure.
	CountSubjects(ctx context.Context, pr PolicyReq) (uint64, error)

	// ListPermissions lists permission betweeen given subject and object .
	ListPermissions(ctx context.Context, pr PolicyReq, permissionsFilter []string) (Permissions, error)
}
