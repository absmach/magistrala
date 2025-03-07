// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"

	"github.com/absmach/supermq/auth"
)

type PolicyReq struct {
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

type PatReq struct {
	UserID           string          `json:"user_id,omitempty"`           // UserID
	PatID            string          `json:"pat_id,omitempty"`            // UserID
	EntityType       auth.EntityType `json:"entity_type,omitempty"`       // Entity type
	OptionalDomainID string          `json:"optional_domainID,omitempty"` // Optional domain id
	Operation        auth.Operation  `json:"operation,omitempty"`         // Operation
	EntityID         string          `json:"entityID,omitempty"`          // EntityID
}

// Authz is supermq authorization library.
//
//go:generate mockery --name Authorization --output=./mocks --filename authz.go --quiet --note "Copyright (c) Abstract Machines"
type Authorization interface {
	Authorize(ctx context.Context, pr PolicyReq) error
	AuthorizePAT(ctx context.Context, pr PatReq) error
}
