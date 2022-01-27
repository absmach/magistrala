// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var _ auth.GroupRepository = (*groupRepositoryMock)(nil)

type groupRepositoryMock struct {
	mu sync.Mutex
	// Map of groups, group id as a key.
	// groups      map[GroupID]auth.Group
	groups map[string]auth.Group
	// Map of groups with group id as key that are
	// children (i.e. has same parent id) is element
	// in children's map where parent id is key.
	// children    map[ParentID]map[GroupID]auth.Group
	children map[string]map[string]auth.Group
	// Map of parents' id with child group id as key.
	// Each child has one parent.
	// parents     map[ChildID]ParentID
	parents map[string]string
	// Map of groups (with group id as key) which
	// represent memberships is element in
	// memberships' map where member id is a key.
	// memberships map[MemberID]map[GroupID]auth.Group
	memberships map[string]map[string]auth.Group
	// Map of group members where member id is a key
	// is an element in the map members where group id is a key.
	// members     map[type][GroupID]map[MemberID]MemberID
	members map[string]map[string]map[string]string
}

// NewGroupRepository creates in-memory user repository
func NewGroupRepository() auth.GroupRepository {
	return &groupRepositoryMock{
		groups:      make(map[string]auth.Group),
		children:    make(map[string]map[string]auth.Group),
		parents:     make(map[string]string),
		memberships: make(map[string]map[string]auth.Group),
		members:     make(map[string]map[string]map[string]string),
	}
}

func (grm *groupRepositoryMock) Save(ctx context.Context, group auth.Group) (auth.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[group.ID]; ok {
		return auth.Group{}, auth.ErrGroupConflict
	}
	path := group.ID

	if group.ParentID != "" {
		parent, ok := grm.groups[group.ParentID]
		if !ok {
			return auth.Group{}, auth.ErrCreateGroup
		}
		if _, ok := grm.children[group.ParentID]; !ok {
			grm.children[group.ParentID] = make(map[string]auth.Group)
		}
		grm.children[group.ParentID][group.ID] = group
		grm.parents[group.ID] = group.ParentID
		path = fmt.Sprintf("%s.%s", parent.Path, path)
	}

	group.Path = path
	group.Level = len(strings.Split(path, "."))

	grm.groups[group.ID] = group
	return group, nil
}

func (grm *groupRepositoryMock) Update(ctx context.Context, group auth.Group) (auth.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	up, ok := grm.groups[group.ID]
	if !ok {
		return auth.Group{}, errors.ErrNotFound
	}
	up.Name = group.Name
	up.Description = group.Description
	up.Metadata = group.Metadata
	up.UpdatedAt = time.Now()

	grm.groups[group.ID] = up
	return up, nil
}

func (grm *groupRepositoryMock) Delete(ctx context.Context, id string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[id]; !ok {
		return auth.ErrGroupNotFound
	}

	if len(grm.members[id]) > 0 {
		return auth.ErrGroupNotEmpty
	}

	// This is not quite exact, it should go in depth
	for _, ch := range grm.children[id] {
		if len(grm.members[ch.ID]) > 0 {
			return auth.ErrGroupNotEmpty
		}
	}

	// This is not quite exact, it should go in depth
	delete(grm.groups, id)
	for _, ch := range grm.children[id] {
		delete(grm.members, ch.ID)
	}

	delete(grm.children, id)

	return nil

}

func (grm *groupRepositoryMock) RetrieveByID(ctx context.Context, id string) (auth.Group, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	val, ok := grm.groups[id]
	if !ok {
		return auth.Group{}, auth.ErrGroupNotFound
	}
	return val, nil
}

func (grm *groupRepositoryMock) RetrieveAll(ctx context.Context, pm auth.PageMetadata) (auth.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []auth.Group
	for _, g := range grm.groups {
		items = append(items, g)
	}
	return auth.GroupPage{
		Groups: items,
		PageMetadata: auth.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) Unassign(ctx context.Context, groupID string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return auth.ErrGroupNotFound
	}
	for _, memberID := range memberIDs {
		for typ, m := range grm.members[groupID] {
			_, ok := m[memberID]
			if !ok {
				return auth.ErrGroupNotFound
			}
			delete(grm.members[groupID][typ], memberID)
			delete(grm.memberships[memberID], groupID)
		}

	}
	return nil
}

func (grm *groupRepositoryMock) Assign(ctx context.Context, groupID, groupType string, memberIDs ...string) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if _, ok := grm.groups[groupID]; !ok {
		return auth.ErrGroupNotFound
	}

	if _, ok := grm.members[groupID]; !ok {
		grm.members[groupID] = make(map[string]map[string]string)
	}

	for _, memberID := range memberIDs {
		if _, ok := grm.members[groupID][groupType]; !ok {
			grm.members[groupID][groupType] = make(map[string]string)
		}
		if _, ok := grm.memberships[memberID]; !ok {
			grm.memberships[memberID] = make(map[string]auth.Group)
		}

		grm.members[groupID][groupType][memberID] = memberID
		grm.memberships[memberID][groupID] = grm.groups[groupID]
	}
	return nil

}

func (grm *groupRepositoryMock) Memberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []auth.Group

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	i := uint64(0)
	for _, g := range grm.memberships[memberID] {
		if i >= first && i < last {
			items = append(items, g)
		}
		i++
	}

	return auth.GroupPage{
		Groups: items,
		PageMetadata: auth.PageMetadata{
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Total:  uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) Members(ctx context.Context, groupID, groupType string, pm auth.PageMetadata) (auth.MemberPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	var items []auth.Member
	members, ok := grm.members[groupID][groupType]
	if !ok {
		return auth.MemberPage{}, auth.ErrGroupNotFound
	}

	first := uint64(pm.Offset)
	last := first + uint64(pm.Limit)

	i := uint64(0)
	for _, g := range members {
		if i >= first && i < last {
			items = append(items, auth.Member{ID: g, Type: groupType})
		}
		i++
	}
	return auth.MemberPage{
		Members: items,
		PageMetadata: auth.PageMetadata{
			Total: uint64(len(items)),
		},
	}, nil
}

func (grm *groupRepositoryMock) RetrieveAllParents(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	if groupID == "" {
		return auth.GroupPage{}, nil
	}

	group, ok := grm.groups[groupID]
	if !ok {
		return auth.GroupPage{}, auth.ErrGroupNotFound
	}

	groups := make([]auth.Group, 0)
	groups, err := grm.getParents(groups, group)
	if err != nil {
		return auth.GroupPage{}, err
	}

	return auth.GroupPage{
		Groups: groups,
		PageMetadata: auth.PageMetadata{
			Total: uint64(len(groups)),
		},
	}, nil
}

func (grm *groupRepositoryMock) getParents(groups []auth.Group, group auth.Group) ([]auth.Group, error) {
	groups = append(groups, group)
	parentID, ok := grm.parents[group.ID]
	if !ok && parentID == "" {
		return groups, nil
	}
	parent, ok := grm.groups[parentID]
	if !ok {
		panic(fmt.Sprintf("parent with id: %s not found", parentID))
	}
	return grm.getParents(groups, parent)
}

func (grm *groupRepositoryMock) RetrieveAllChildren(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()
	group, ok := grm.groups[groupID]
	if !ok {
		return auth.GroupPage{}, nil
	}

	groups := make([]auth.Group, 0)
	groups = append(groups, group)
	for ch := range grm.parents {
		g, ok := grm.groups[ch]
		if !ok {
			panic(fmt.Sprintf("child with id %s not found", ch))
		}
		groups = append(groups, g)
	}

	return auth.GroupPage{
		Groups: groups,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(groups)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}
