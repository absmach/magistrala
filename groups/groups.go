// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/roles"
)

// MaxLevel represents the maximum group hierarchy level.
const (
	MaxLevel      = uint64(20)
	MaxPathLength = 20
)

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

// Group represents the group of Clients.
// Indicates a level in tree hierarchy. Root node is level 1.
// Path in a tree consisting of group IDs
// Paths are unique per domain.
type Group struct {
	ID                        string    `json:"id"`
	Domain                    string    `json:"domain_id,omitempty"`
	Parent                    string    `json:"parent_id,omitempty"`
	Name                      string    `json:"name"`
	Description               string    `json:"description,omitempty"`
	Metadata                  Metadata  `json:"metadata,omitempty"`
	Level                     int       `json:"level,omitempty"`
	Path                      string    `json:"path,omitempty"`
	Children                  []*Group  `json:"children,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at,omitempty"`
	UpdatedBy                 string    `json:"updated_by,omitempty"`
	Status                    Status    `json:"status"`
	RoleID                    string    `json:"role_id,omitempty"`
	RoleName                  string    `json:"role_name,omitempty"`
	Actions                   []string  `json:"actions,omitempty"`
	AccessType                string    `json:"access_type,omitempty"`
	AccessProviderId          string    `json:"access_provider_id,omitempty"`
	AccessProviderRoleId      string    `json:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string    `json:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions []string  `json:"access_provider_role_actions,omitempty"`
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
	Groups []Group
}

type HierarchyPageMeta struct {
	Level     uint64 `json:"level"`
	Direction int64  `json:"direction"` // ancestors (+1) or descendants (-1)
	// - `true`  - result is JSON tree representing groups hierarchy,
	// - `false` - result is JSON array of groups.
	Tree bool `json:"tree"`
}
type HierarchyPage struct {
	HierarchyPageMeta
	Groups []Group
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

	RetrieveByIDAndUser(ctx context.Context, domainID, userID, groupID string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, pm PageMeta) (Page, error)

	// RetrieveByIDs retrieves group by ids and query.
	RetrieveByIDs(ctx context.Context, pm PageMeta, ids ...string) (Page, error)

	RetrieveHierarchy(ctx context.Context, id string, hm HierarchyPageMeta) (HierarchyPage, error)

	// ChangeStatus changes groups status to active or inactive
	ChangeStatus(ctx context.Context, group Group) (Group, error)

	// AssignParentGroup assigns parent group id to a given group id
	AssignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error

	// UnassignParentGroup unassign parent group id fr given group id
	UnassignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error

	UnassignAllChildrenGroups(ctx context.Context, id string) error

	RetrieveUserGroups(ctx context.Context, domainID, userID string, pm PageMeta) (Page, error)

	// RetrieveChildrenGroups at given level in ltree
	// Condition: startLevel == 0 and endLevel < 0, Retrieve all children groups from parent group level, Example: If we pass startLevel 0 and endLevel -1, then function will return all children of parent group
	// Condition: startLevel > 0 and endLevel == 0, Retrieve specific level of children groups from parent group level, Example: If we pass startLevel 1 and endLevel 0, then function will return children of parent group from level 1
	// Condition: startLevel > 0 and endLevel < 0,  Retrieve all children groups from specific level from parent group level, Example: If we pass startLevel 2 and endLevel -1, then function will return all children of parent group from level 2
	// Condition: startLevel > 0 and endLevel > 0, Retrieve children groups between specific level from parent group level, Example: If we pass startLevel 3 and endLevel 5, then function will return all children of parent group between level 3 and 5
	RetrieveChildrenGroups(ctx context.Context, domainID, userID, groupID string, startLevel, endLevel int64, pm PageMeta) (Page, error)

	RetrieveAllParentGroups(ctx context.Context, domainID, userID, groupID string, pm PageMeta) (Page, error)
	// Delete a group
	Delete(ctx context.Context, groupID string) error

	roles.Repository
}

//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines" --unroll-variadic=false
type Service interface {
	// CreateGroup creates new  group.
	CreateGroup(ctx context.Context, session authn.Session, g Group) (Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, session authn.Session, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, session authn.Session, id string) (Group, error)

	// ListGroups retrieves
	ListGroups(ctx context.Context, session authn.Session, pm PageMeta) (Page, error)

	ListUserGroups(ctx context.Context, session authn.Session, userID string, pm PageMeta) (Page, error)

	// EnableGroup logically enables the group identified with the provided ID.
	EnableGroup(ctx context.Context, session authn.Session, id string) (Group, error)

	// DisableGroup logically disables the group identified with the provided ID.
	DisableGroup(ctx context.Context, session authn.Session, id string) (Group, error)

	// DeleteGroup delete the given group id
	DeleteGroup(ctx context.Context, session authn.Session, id string) error

	RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm HierarchyPageMeta) (HierarchyPage, error)

	AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error

	RemoveParentGroup(ctx context.Context, session authn.Session, id string) error

	AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error

	RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error

	RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error

	ListChildrenGroups(ctx context.Context, session authn.Session, id string, startLevel, endLevel int64, pm PageMeta) (Page, error)

	roles.RoleManager
}
