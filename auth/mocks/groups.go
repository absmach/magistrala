// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/mainflux/mainflux/internal/groups"
)

var _ groups.Repository = (*groupRepositoryMock)(nil)

type groupRepositoryMock struct {
	mu     sync.Mutex
	groups map[string]groups.Group
	// Map of "Maps of users assigned to a group" where group is a key
	childrenByGroups map[string]map[string]groups.Group
	groupsByMember   map[string]map[string]groups.Group
	members          map[string]map[string]interface{}
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository() groups.Repository {
	return &groupRepositoryMock{
		groups:           make(map[string]groups.Group),
		childrenByGroups: make(map[string]map[string]groups.Group),
	}
}

func (grm *groupRepositoryMock) Save(ctx context.Context, g groups.Group) (groups.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[g.ID]; ok {
		return groups.Group{}, groups.ErrGroupConflict
	}

	if g.ParentID != "" {
		if _, ok := grm.groups[g.ParentID]; !ok {
			return groups.Group{}, groups.ErrCreateGroup
		}
		if _, ok := grm.childrenByGroups[g.ParentID]; !ok {
			grm.childrenByGroups[g.ParentID] = make(map[string]groups.Group)
		}
		grm.childrenByGroups[g.ParentID][g.ID] = g
	}
	grm.groups[g.ID] = g
	return g, nil
}

func (grm *groupRepositoryMock) Update(ctx context.Context, g groups.Group) (groups.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	up, ok := grm.groups[g.ID]
	if !ok {
		return groups.Group{}, groups.ErrNotFound
	}
	up.Name = g.Name
	up.Description = g.Description
	up.Metadata = g.Metadata
	up.UpdatedAt = time.Now()

	grm.groups[g.ID] = up
	if g.ParentID != "" {
		grm.childrenByGroups[g.ParentID][g.ID] = g
	}
	return g, nil
}

func (grm *groupRepositoryMock) Delete(ctx context.Context, id string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[id]; !ok {
		return groups.ErrNotFound
	}

	delete(grm.groups, id)
	delete(grm.childrenByGroups, id)
	return nil

}

func (grm *groupRepositoryMock) RetrieveByID(ctx context.Context, id string) (groups.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	val, ok := grm.groups[id]
	if !ok {
		return groups.Group{}, groups.ErrNotFound
	}
	return val, nil
}

func (grm *groupRepositoryMock) RetrieveAll(ctx context.Context, level uint64, m groups.Metadata) (groups.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []groups.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}
	return groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) Unassign(ctx context.Context, memberID, groupID string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return groups.ErrNotFound
	}
	delete(grm.members[groupID], memberID)
	delete(grm.groupsByMember, memberID)
	return nil
}

func (grm *groupRepositoryMock) Assign(ctx context.Context, memberID, groupID string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return groups.ErrNotFound
	}
	if _, ok := grm.members[groupID]; !ok {
		grm.members[groupID] = make(map[string]interface{})
	}

	grm.members[groupID][memberID] = memberID
	grm.groupsByMember[memberID][groupID] = grm.groups[groupID]
	return nil

}

func (grm *groupRepositoryMock) Memberships(ctx context.Context, memberID string, offset, limit uint64, um groups.Metadata) (groups.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []groups.Group
	memberships, ok := grm.groupsByMember[memberID]
	if !ok {
		return groups.GroupPage{}, groups.ErrNotFound
	}
	for _, g := range memberships {
		items = append(items, g)
	}
	return groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Limit:  limit,
			Offset: offset,
			Total:  uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) Members(ctx context.Context, groupID string, offset, limit uint64, m groups.Metadata) (groups.MemberPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []groups.Member
	members, ok := grm.members[groupID]
	if !ok {
		return groups.MemberPage{}, groups.ErrNotFound
	}
	for _, g := range members {
		items = append(items, g)
	}
	return groups.MemberPage{
		Members: items,
		PageMetadata: groups.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveAllParents(ctx context.Context, groupID string, level uint64, m groups.Metadata) (groups.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if groupID == "" {
		return groups.GroupPage{}, nil
	}

	var items []groups.Group
	parent, ok := grm.groups[groupID]
	if !ok {
		return groups.GroupPage{}, nil
	}

	for {
		items = append(items, parent)
		parent, ok = grm.groups[parent.ParentID]
		if !ok {
			break
		}
	}
	return groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveAllChildren(ctx context.Context, groupID string, level uint64, um groups.Metadata) (groups.GroupPage, error) {
	panic("not implemented")
}
