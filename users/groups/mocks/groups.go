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

func (m *Repository) Memberships(ctx context.Context, clientID string, gm mfgroups.GroupsPage) (mfgroups.MembershipsPage, error) {
	ret := m.Called(ctx, clientID, gm)

	if clientID == WrongID {
		return mfgroups.MembershipsPage{}, errors.ErrNotFound
	}

	return ret.Get(0).(mfgroups.MembershipsPage), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, gm mfgroups.GroupsPage) (mfgroups.GroupsPage, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(mfgroups.GroupsPage), ret.Error(1)
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
