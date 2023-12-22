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
	ViewerRelation        = "viewer"
	MemberRelation        = "member"
	DomainRelation        = "domain"
	ParentGroupRelation   = "parent_group"
	RoleGroupRelation     = "role_group"
	GroupRelation         = "group"
	PlatformRelation      = "platform"
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

	// Relation contains the relation. Supported relations are administrator, editor, viewer, member,parent_group,group,domain.
	Relation string `json:"relation,omitempty"`

	// Permission contains the permission. Supported permissions are admin, delete, edit, share, view, membership,
	// admin_only, edit_only, viewer_only, membership_only, ext_admin, ext_edit, ext_view
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

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, prs []PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, prs []PolicyReq) error

	// ListObjects lists policies based on the given PolicyReq structure.
	ListObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error)

	// ListAllObjects lists all policies based on the given PolicyReq structure.
	ListAllObjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountPolicies count policies based on the given PolicyReq structure.
	CountObjects(ctx context.Context, pr PolicyReq) (int, error)

	// ListSubjects lists subjects based on the given PolicyReq structure.
	ListSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error)

	// ListAllSubjects lists all subjects based on the given PolicyReq structure.
	ListAllSubjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountSubjects count policies based on the given PolicyReq structure.
	CountSubjects(ctx context.Context, pr PolicyReq) (int, error)

	// ListPermissions lists permission betweeen given subject and object .
	ListPermissions(ctx context.Context, pr PolicyReq, filterPermission []string) (Permissions, error)
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

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies creates a Bulk Policies  for the given request
	AddPolicies(ctx context.Context, prs []PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicies(ctx context.Context, pr []PolicyReq) error

	// RetrieveObjects
	RetrieveObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) ([]PolicyRes, string, error)

	// RetrieveAllObjects
	RetrieveAllObjects(ctx context.Context, pr PolicyReq) ([]PolicyRes, error)

	// RetrieveAllObjectsCount
	RetrieveAllObjectsCount(ctx context.Context, pr PolicyReq) (int, error)

	// RetrieveSubjects
	RetrieveSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) ([]PolicyRes, string, error)

	// RetrieveAllSubjects
	RetrieveAllSubjects(ctx context.Context, pr PolicyReq) ([]PolicyRes, error)

	// RetrieveAllSubjectsCount
	RetrieveAllSubjectsCount(ctx context.Context, pr PolicyReq) (int, error)

	// (ctx context.Context, pr PolicyReq, filterPermissions []string) ([]PolicyReq, error)
	RetrievePermissions(ctx context.Context, pr PolicyReq, filterPermission []string) (Permissions, error)
}
