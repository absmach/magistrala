// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"

	"github.com/mainflux/mainflux/pkg/groups"
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateGroup creates new  group.
	CreateGroups(ctx context.Context, token string, gs ...groups.Group) ([]groups.Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (groups.Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token string, gm groups.GroupsPage) (groups.GroupsPage, error)

	// ListMemberships retrieves everything that is assigned to a group identified by clientID.
	ListMemberships(ctx context.Context, token, clientID string, gm groups.GroupsPage) (groups.MembershipsPage, error)

	// EnableGroup logically enables the group identified with the provided ID.
	EnableGroup(ctx context.Context, token, id string) (groups.Group, error)

	// DisableGroup logically disables the group identified with the provided ID.
	DisableGroup(ctx context.Context, token, id string) (groups.Group, error)
}
