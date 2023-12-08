// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/mock"
)

var _ invitations.Service = (*Service)(nil)

type Service struct {
	mock.Mock
}

func (svc *Service) SendInvitation(ctx context.Context, token string, invitation invitations.Invitation) (err error) {
	ret := svc.Called(ctx, token, invitation)

	if token == Invalid || invitation.UserID == Invalid || invitation.DomainID == Invalid || invitation.InvitedBy == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (svc *Service) ViewInvitation(ctx context.Context, token, userID, domainID string) (invitation invitations.Invitation, err error) {
	ret := svc.Called(ctx, token, userID, domainID)

	if token == Invalid || userID == Invalid || domainID == Invalid {
		return invitations.Invitation{}, errors.ErrNotFound
	}

	return ret.Get(0).(invitations.Invitation), ret.Error(1)
}

func (svc *Service) ListInvitations(ctx context.Context, token string, page invitations.Page) (invitations.InvitationPage, error) {
	ret := svc.Called(ctx, token, page)

	if token == Invalid {
		return invitations.InvitationPage{}, errors.ErrAuthentication
	}

	return ret.Get(0).(invitations.InvitationPage), ret.Error(1)
}

func (svc *Service) AcceptInvitation(ctx context.Context, token, domainID string) (err error) {
	ret := svc.Called(ctx, token, domainID)

	if token == Invalid {
		return errors.ErrAuthentication
	}

	return ret.Error(0)
}

func (svc *Service) DeleteInvitation(ctx context.Context, token, userID, domainID string) (err error) {
	ret := svc.Called(ctx, token, userID, domainID)

	if token == Invalid {
		return errors.ErrAuthentication
	}

	if userID == Invalid || domainID == Invalid {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}
