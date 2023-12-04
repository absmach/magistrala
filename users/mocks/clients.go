// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/users/postgres"
	"github.com/stretchr/testify/mock"
)

const WrongID = "wrongID"

var _ postgres.Repository = (*Repository)(nil)

type Repository struct {
	mock.Mock
}

func (m *Repository) ChangeStatus(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	if client.Status != mgclients.EnabledStatus && client.Status != mgclients.DisabledStatus {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) Members(ctx context.Context, groupID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	ret := m.Called(ctx, groupID, pm)
	if groupID == WrongID {
		return mgclients.MembersPage{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mgclients.MembersPage), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(mgclients.ClientsPage), ret.Error(1)
}

func (m *Repository) RetrieveAllBasicInfo(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(mgclients.ClientsPage), ret.Error(1)
}

func (m *Repository) RetrieveByID(ctx context.Context, id string) (mgclients.Client, error) {
	ret := m.Called(ctx, id)

	if id == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) RetrieveByIdentity(ctx context.Context, identity string) (mgclients.Client, error) {
	ret := m.Called(ctx, identity)

	if identity == "" {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)
	if client.Owner == WrongID {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}
	if client.Credentials.Secret == "" {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return client, ret.Error(1)
}

func (m *Repository) Update(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}
	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) UpdateIdentity(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}
	if client.Credentials.Identity == "" {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) UpdateSecret(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}
	if client.Credentials.Secret == "" {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) UpdateTags(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) UpdateOwner(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) UpdateRole(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ret := m.Called(ctx, client)

	if client.ID == WrongID {
		return mgclients.Client{}, errors.ErrNotFound
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error) {
	ret := m.Called(ctx, key)

	if key == "" {
		return mgclients.Client{}, repoerr.ErrMalformedEntity
	}

	return ret.Get(0).(mgclients.Client), ret.Error(1)
}

func (m *Repository) CheckSuperAdmin(ctx context.Context, userID string) error {
	ret := m.Called(ctx, userID)

	if userID == WrongID {
		return errors.ErrAuthorization
	}

	return ret.Error(0)
}

func (m *Repository) RetrieveAllByIDs(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(mgclients.ClientsPage), ret.Error(1)
}

func (m *Repository) RetrieveNames(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(mgclients.ClientsPage), ret.Error(1)
}
