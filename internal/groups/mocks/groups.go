// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/stretchr/testify/mock"
)

const WrongID = "wrongID"

var _ mggroups.Repository = (*Repository)(nil)

type Repository struct {
	mock.Mock
}

func (m *Repository) ChangeStatus(ctx context.Context, group mggroups.Group) (mggroups.Group, error) {
	ret := m.Called(ctx, group)

	if group.ID == WrongID {
		return mggroups.Group{}, repoerr.ErrNotFound
	}

	if group.Status != mgclients.EnabledStatus && group.Status != mgclients.DisabledStatus {
		return mggroups.Group{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mggroups.Group), ret.Error(1)
}

func (m *Repository) RetrieveByIDs(ctx context.Context, gm mggroups.Page, ids ...string) (mggroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mggroups.Page), ret.Error(1)
}

func (m *Repository) MembershipsByGroupIDs(ctx context.Context, gm mggroups.Page) (mggroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mggroups.Page), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, gm mggroups.Page) (mggroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mggroups.Page), ret.Error(1)
}

func (m *Repository) RetrieveByID(ctx context.Context, id string) (mggroups.Group, error) {
	ret := m.Called(ctx, id)

	if id == WrongID {
		return mggroups.Group{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mggroups.Group), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, g mggroups.Group) (mggroups.Group, error) {
	ret := m.Called(ctx, g)

	if g.Parent == WrongID {
		return mggroups.Group{}, repoerr.ErrCreateEntity
	}

	if g.Owner == WrongID {
		return mggroups.Group{}, repoerr.ErrCreateEntity
	}

	return g, ret.Error(1)
}

func (m *Repository) Update(ctx context.Context, g mggroups.Group) (mggroups.Group, error) {
	ret := m.Called(ctx, g)

	if g.ID == WrongID {
		return mggroups.Group{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mggroups.Group), ret.Error(1)
}

func (m *Repository) UnassignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error {
	ret := m.Called(ctx, parentGroupID, groupIDs)

	return ret.Error(0)
}

func (m *Repository) AssignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error {
	ret := m.Called(ctx, parentGroupID, groupIDs)

	return ret.Error(0)
}

func (m *Repository) Delete(ctx context.Context, groupID string) error {
	ret := m.Called(ctx, groupID)
	if groupID == WrongID {
		return repoerr.ErrNotFound
	}
	return ret.Error(0)
}
