// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/clients"
)

// MaxLevel represents the maximum group hierarchy level.
const MaxLevel = uint64(5)

// Group represents the group of Clients.
// Indicates a level in tree hierarchy. Root node is level 1.
// Path in a tree consisting of group IDs
// Paths are unique per owner.
type Group struct {
	ID          string           `json:"id"`
	Owner       string           `json:"owner_id,omitempty"`
	Parent      string           `json:"parent_id,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Metadata    clients.Metadata `json:"metadata,omitempty"`
	Level       int              `json:"level,omitempty"`
	Path        string           `json:"path,omitempty"`
	Children    []*Group         `json:"children,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`
	UpdatedBy   string           `json:"updated_by,omitempty"`
	Status      clients.Status   `json:"status"`
	Permissions []string         `json:"permissions,omitempty"`
}

type Member struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Memberships contains page related metadata as well as list of memberships that
// belong to this page.
type MembersPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Members []Member `json:"members"`
}

// Page contains page related metadata as well as list
// of Groups that belong to the page.
type Page struct {
	PageMeta
	Path       string
	Level      uint64
	ID         string
	Permission string
	ListPerms  bool
	Direction  int64 // ancestors (+1) or descendants (-1)
	Groups     []Group
}

// Repository specifies a group persistence API.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines" --unroll-variadic=false
type Repository interface {
	// Save group.
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group.
	Update(ctx context.Context, g Group) (Group, error)

	// RetrieveByID retrieves group by its id.
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, gm Page) (Page, error)

	// RetrieveByIDs retrieves group by ids and query.
	RetrieveByIDs(ctx context.Context, gm Page, ids ...string) (Page, error)

	// ChangeStatus changes groups status to active or inactive
	ChangeStatus(ctx context.Context, group Group) (Group, error)

	// AssignParentGroup assigns parent group id to a given group id
	AssignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error

	// UnassignParentGroup unassign parent group id fr given group id
	UnassignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error

	// Delete a group
	Delete(ctx context.Context, groupID string) error
}

//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines" --unroll-variadic=false
type Service interface {
	// CreateGroup creates new  group.
	CreateGroup(ctx context.Context, token, kind string, g Group) (Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ViewGroupPerms retrieves permissions on the group id for the given authorized token.
	ViewGroupPerms(ctx context.Context, token, id string) ([]string, error)

	// ListGroups retrieves
	ListGroups(ctx context.Context, token, memberKind, memberID string, gm Page) (Page, error)

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (MembersPage, error)

	// EnableGroup logically enables the group identified with the provided ID.
	EnableGroup(ctx context.Context, token, id string) (Group, error)

	// DisableGroup logically disables the group identified with the provided ID.
	DisableGroup(ctx context.Context, token, id string) (Group, error)

	// DeleteGroup delete the given group id
	DeleteGroup(ctx context.Context, token, id string) error

	// Assign member to group
	Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error)

	// Unassign member from group
	Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error)
}
