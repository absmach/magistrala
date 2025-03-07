// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
	"encoding/json"
)

type Policy struct {
	// Domain contains the domain ID.
	Domain string `json:"domain,omitempty"`

	// Subject contains the subject ID or Token.
	Subject string `json:"subject"`

	// SubjectType contains the subject type. Supported subject types are
	// platform, group, domain, client, users.
	SubjectType string `json:"subject_type"`

	// SubjectKind contains the subject kind. Supported subject kinds are
	// token, users, platform, clients,  channels, groups, domain.
	SubjectKind string `json:"subject_kind"`

	// SubjectRelation contains subject relations.
	SubjectRelation string `json:"subject_relation,omitempty"`

	// ObjectPrefix contains the Optional Object Prefix which is used for delete with filter.
	ObjectPrefix string `json:"object_prefix"`

	// Object contains the object ID.
	Object string `json:"object"`

	// ObjectKind contains the object kind. Supported object kinds are
	// users, platform, clients,  channels, groups, domain.
	ObjectKind string `json:"object_kind"`

	// ObjectType contains the object type. Supported object types are
	// platform, group, domain, client, users.
	ObjectType string `json:"object_type"`

	// Relation contains the relation. Supported relations are administrator, editor, contributor, member, guest, parent_group,group,domain.
	Relation string `json:"relation,omitempty"`

	// Permission contains the permission. Supported permissions are admin, delete, edit, share, view,
	// membership, create, admin_only, edit_only, view_only, membership_only, ext_admin, ext_edit, ext_view.
	Permission string `json:"permission,omitempty"`
}

func (pr Policy) String() string {
	data, err := json.Marshal(pr)
	if err != nil {
		return ""
	}
	return string(data)
}

type PolicyPage struct {
	Policies      []string
	NextPageToken string
}

type Permissions []string

// PolicyService facilitates the communication to authorization
// services and implements Authz functionalities for spicedb
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr Policy) error

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, prs []Policy) error

	// DeletePolicyFilter removes policy for given policy filter request.
	DeletePolicyFilter(ctx context.Context, pr Policy) error

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, prs []Policy) error

	// ListObjects lists policies based on the given Policy structure.
	ListObjects(ctx context.Context, pr Policy, nextPageToken string, limit uint64) (PolicyPage, error)

	// ListAllObjects lists all policies based on the given Policy structure.
	ListAllObjects(ctx context.Context, pr Policy) (PolicyPage, error)

	// CountObjects count policies based on the given Policy structure.
	CountObjects(ctx context.Context, pr Policy) (uint64, error)

	// ListSubjects lists subjects based on the given Policy structure.
	ListSubjects(ctx context.Context, pr Policy, nextPageToken string, limit uint64) (PolicyPage, error)

	// ListAllSubjects lists all subjects based on the given Policy structure.
	ListAllSubjects(ctx context.Context, pr Policy) (PolicyPage, error)

	// CountSubjects count policies based on the given Policy structure.
	CountSubjects(ctx context.Context, pr Policy) (uint64, error)

	// ListPermissions lists permission betweeen given subject and object .
	ListPermissions(ctx context.Context, pr Policy, permissionsFilter []string) (Permissions, error)
}

func EncodeDomainUserID(domainID, userID string) string {
	if domainID == "" || userID == "" {
		return ""
	}
	return domainID + "_" + userID
}
