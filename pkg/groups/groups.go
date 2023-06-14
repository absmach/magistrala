// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"time"

	"github.com/mainflux/mainflux/pkg/clients"
)

const (
	// MaxLevel represents the maximum group hierarchy level.
	MaxLevel = uint64(5)
	// MinLevel represents the minimum group hierarchy level.
	MinLevel = uint64(0)
)

// Group represents the group of Clients.
// Indicates a level in tree hierarchy. Root node is level 1.
// Path in a tree consisting of group IDs
// Paths are unique per owner.
type Group struct {
	ID          string           `json:"id"`
	Owner       string           `json:"owner_id"`
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
}

// MembershipsPage contains page related metadata as well as list of memberships that
// belong to this page.
type MembershipsPage struct {
	Page
	Memberships []Group
}

// GroupsPage contains page related metadata as well as list
// of Groups that belong to the page.
type GroupsPage struct {
	Page
	Path      string
	Level     uint64
	ID        string
	Direction int64 // ancestors (-1) or descendants (+1)
	Groups    []Group
}

// Repository specifies a group persistence API.
type Repository interface {
	// Save group.
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group.
	Update(ctx context.Context, g Group) (Group, error)

	// RetrieveByID retrieves group by its id.
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, gm GroupsPage) (GroupsPage, error)

	// Memberships retrieves everything that is assigned to a group identified by clientID.
	Memberships(ctx context.Context, clientID string, gm GroupsPage) (MembershipsPage, error)

	// ChangeStatus changes groups status to active or inactive
	ChangeStatus(ctx context.Context, group Group) (Group, error)
}
