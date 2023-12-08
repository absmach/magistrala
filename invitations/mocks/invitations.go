// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/mock"
)

const Invalid = "invalid"

var _ invitations.Repository = (*Repository)(nil)

type Repository struct {
	mock.Mock
}

func (m *Repository) Create(ctx context.Context, invitation invitations.Invitation) error {
	ret := m.Called(ctx, invitation)

	if invitation.UserID == Invalid || invitation.DomainID == Invalid || invitation.InvitedBy == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Retrieve(ctx context.Context, userID, domainID string) (invitations.Invitation, error) {
	ret := m.Called(ctx, userID, domainID)

	if userID == Invalid || domainID == Invalid {
		return invitations.Invitation{}, errors.ErrNotFound
	}

	return ret.Get(0).(invitations.Invitation), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, page invitations.Page) (invitations.InvitationPage, error) {
	ret := m.Called(ctx, page)

	if page.UserID == Invalid || page.DomainID == Invalid {
		return invitations.InvitationPage{}, errors.ErrNotFound
	}

	return ret.Get(0).(invitations.InvitationPage), ret.Error(1)
}

func (m *Repository) UpdateToken(ctx context.Context, invitation invitations.Invitation) error {
	ret := m.Called(ctx, invitation)

	if invitation.UserID == Invalid || invitation.DomainID == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) UpdateConfirmation(ctx context.Context, invitation invitations.Invitation) error {
	ret := m.Called(ctx, invitation)

	if invitation.UserID == Invalid || invitation.DomainID == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Delete(ctx context.Context, userID, domainID string) error {
	ret := m.Called(ctx, userID, domainID)

	if userID == Invalid || domainID == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}
