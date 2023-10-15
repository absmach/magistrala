// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/stretchr/testify/mock"
)

const WrongID = "wrongID"

var _ mfgroups.Repository = (*Repository)(nil)

type Repository struct {
	mock.Mock
}

func (m *Repository) ChangeStatus(ctx context.Context, group mfgroups.Group) (mfgroups.Group, error) {
	ret := m.Called(ctx, group)

	if group.ID == WrongID {
		return mfgroups.Group{}, errors.ErrNotFound
	}

	if group.Status != mfclients.EnabledStatus && group.Status != mfclients.DisabledStatus {
		return mfgroups.Group{}, errors.ErrMalformedEntity
	}

	return ret.Get(0).(mfgroups.Group), ret.Error(1)
}

func (m *Repository) RetrieveByIDs(ctx context.Context, gm mfgroups.Page, ids ...string) (mfgroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mfgroups.Page), ret.Error(1)
}

func (m *Repository) MembershipsByGroupIDs(ctx context.Context, gm mfgroups.Page) (mfgroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mfgroups.Page), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, gm mfgroups.Page) (mfgroups.Page, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mfgroups.Page), ret.Error(1)
}

func (m *Repository) RetrieveByID(ctx context.Context, id string) (mfgroups.Group, error) {
	ret := m.Called(ctx, id)

	if id == WrongID {
		return mfgroups.Group{}, errors.ErrNotFound
	}

	return ret.Get(0).(mfgroups.Group), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, g mfgroups.Group) (mfgroups.Group, error) {
	ret := m.Called(ctx, g)

	if g.Parent == WrongID {
		return mfgroups.Group{}, errors.ErrCreateEntity
	}

	if g.Owner == WrongID {
		return mfgroups.Group{}, errors.ErrCreateEntity
	}

	return g, ret.Error(1)
}

func (m *Repository) Update(ctx context.Context, g mfgroups.Group) (mfgroups.Group, error) {
	ret := m.Called(ctx, g)

	if g.ID == WrongID {
		return mfgroups.Group{}, errors.ErrNotFound
	}

	return ret.Get(0).(mfgroups.Group), ret.Error(1)
}

func (m *Repository) Unassign(ctx context.Context, groupID, memberKind string, memberIDs ...string) error {
	ret := m.Called(ctx, groupID, memberKind, memberIDs)

	if groupID == WrongID {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Assign(ctx context.Context, groupID, groupType string, memberIDs ...string) error {
	ret := m.Called(ctx, groupID, groupType, memberIDs)

	if groupID == WrongID {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}
