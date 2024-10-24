// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations_test

import (
	"context"
	"testing"
	"time"

	authmocks "github.com/absmach/magistrala/auth/mocks"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/mocks"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validInvitation = invitations.Invitation{
		UserID:   testsutil.GenerateUUID(&testing.T{}),
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		Relation: policies.ContributorRelation,
	}
	validDomainUserID = "domain_user_id"
	validUserID       = "user_id"
	validDomainID     = "domain_id"
	validToken        = "valid_token"
	invalidToken      = "invalid"
)

func TestSendInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	svc := invitations.NewService(token, repo, nil)

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		tokenUserID string
		req         invitations.Invitation
		err         error
		issueErr    error
		repoErr     error
	}{
		{
			desc:        "send invitation successful",
			token:       validToken,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         nil,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "failed to issue token",
			token:       invalidToken,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         svcerr.ErrCreateEntity,
			issueErr:    svcerr.ErrCreateEntity,
			repoErr:     nil,
		},
		{
			desc:        "invalid relation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         invitations.Invitation{Relation: "invalid"},
			err:         apiutil.ErrInvalidRelation,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "resend invitation",
			token:       invalidToken,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			tokenUserID: testsutil.GenerateUUID(t),
			req: invitations.Invitation{
				UserID:   validInvitation.UserID,
				DomainID: validInvitation.DomainID,
				Relation: validInvitation.Relation,
				Resend:   true,
			},
			err:      nil,
			issueErr: nil,
			repoErr:  nil,
		},
		{
			desc:        "error during token issuance",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         svcerr.ErrAuthentication,
			issueErr:    svcerr.ErrAuthentication,
			repoErr:     nil,
		},
	}

	for _, tc := range cases {
		repocall1 := token.On("Issue", context.Background(), mock.Anything).Return(&grpcTokenV1.Token{AccessToken: tc.req.Token}, tc.issueErr)
		repocall2 := repo.On("Create", context.Background(), mock.Anything).Return(tc.repoErr)
		if tc.req.Resend {
			repocall2 = repo.On("UpdateToken", context.Background(), mock.Anything).Return(tc.repoErr)
		}
		err := svc.SendInvitation(context.Background(), tc.session, tc.req)
		assert.Equal(t, tc.err, err, tc.desc)
		repocall1.Unset()
		repocall2.Unset()
	}
}

func TestViewInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	svc := invitations.NewService(token, repo, nil)

	validInvitation := invitations.Invitation{
		InvitedBy:   testsutil.GenerateUUID(t),
		UserID:      testsutil.GenerateUUID(t),
		DomainID:    testsutil.GenerateUUID(t),
		Relation:    policies.ContributorRelation,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
		ConfirmedAt: time.Now().Add(-time.Hour),
	}
	cases := []struct {
		desc        string
		token       string
		userID      string
		domainID    string
		session     authn.Session
		tokenUserID string
		req         invitations.Invitation
		resp        invitations.Invitation
		err         error
		issueErr    error
		repoErr     error
	}{
		{
			desc:        "view invitation successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			resp:        validInvitation,
			err:         nil,
			repoErr:     nil,
		},

		{
			desc:        "error retrieving invitation",
			token:       validToken,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			tokenUserID: testsutil.GenerateUUID(t),
			err:         svcerr.ErrNotFound,
			repoErr:     svcerr.ErrNotFound,
		},
		{
			desc:        "valid invitation for the same user",
			token:       validToken,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			resp:        validInvitation,
			tokenUserID: validInvitation.UserID,
			err:         nil,
			repoErr:     nil,
		},
		{
			desc:        "valid invitation for the invited user",
			token:       validToken,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			session:     authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			tokenUserID: validInvitation.InvitedBy,
			resp:        validInvitation,
			err:         nil,
			repoErr:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.repoErr)
			inv, err := svc.ViewInvitation(context.Background(), tc.session, tc.userID, tc.domainID)
			assert.Equal(t, tc.err, err, tc.desc)
			assert.Equal(t, tc.resp, inv, tc.desc)
			repocall1.Unset()
		})
	}
}

func TestListInvitations(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	svc := invitations.NewService(token, repo, nil)

	validPage := invitations.Page{
		Offset: 0,
		Limit:  10,
	}
	validResp := invitations.InvitationPage{
		Total:  1,
		Offset: 0,
		Limit:  10,
		Invitations: []invitations.Invitation{
			{
				InvitedBy:   testsutil.GenerateUUID(t),
				UserID:      testsutil.GenerateUUID(t),
				DomainID:    testsutil.GenerateUUID(t),
				Relation:    policies.ContributorRelation,
				CreatedAt:   time.Now().Add(-time.Hour),
				UpdatedAt:   time.Now().Add(-time.Hour),
				ConfirmedAt: time.Now().Add(-time.Hour),
			},
		},
	}

	cases := []struct {
		desc    string
		session authn.Session
		page    invitations.Page
		resp    invitations.InvitationPage
		err     error
		repoErr error
	}{
		{
			desc:    "list invitations successful",
			session: authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			page:    validPage,
			resp:    validResp,
			err:     nil,
			repoErr: nil,
		},

		{
			desc:    "list invitations unsuccessful",
			session: authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: validUserID},
			page:    validPage,
			err:     repoerr.ErrViewEntity,
			resp:    invitations.InvitationPage{},
			repoErr: repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall1 := repo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.resp, tc.repoErr)
			resp, err := svc.ListInvitations(context.Background(), tc.session, tc.page)
			assert.Equal(t, tc.err, err, tc.desc)
			assert.Equal(t, tc.resp, resp, tc.desc)
			repocall1.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	sdksvc := new(sdkmocks.SDK)
	svc := invitations.NewService(token, repo, sdksvc)

	userID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		token    string
		domainID string
		session  authn.Session
		resp     invitations.Invitation
		err      error
		repoErr  error
		sdkErr   errors.SDKError
		repoErr1 error
	}{
		{
			desc:     "accept invitation successful",
			token:    validToken,
			domainID: "",
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			resp: invitations.Invitation{
				UserID:   userID,
				DomainID: testsutil.GenerateUUID(t),
				Token:    validToken,
				Relation: policies.ContributorRelation,
			},
			err:     nil,
			repoErr: nil,
		},
		{
			desc:    "accept invitation with failed to retrieve all",
			token:   validToken,
			session: authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			err:     svcerr.ErrNotFound,
			repoErr: svcerr.ErrNotFound,
		},
		{
			desc:     "accept invitation with sdk err",
			token:    validToken,
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: "",
			resp: invitations.Invitation{
				UserID:   userID,
				DomainID: testsutil.GenerateUUID(t),
				Token:    validToken,
				Relation: policies.ContributorRelation,
			},
			err:     errors.NewSDKError(svcerr.ErrConflict),
			repoErr: nil,
			sdkErr:  errors.NewSDKError(svcerr.ErrConflict),
		},
		{
			desc:     "accept invitation with failed update confirmation",
			token:    validToken,
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: "",
			resp: invitations.Invitation{
				UserID:   userID,
				DomainID: testsutil.GenerateUUID(t),
				Token:    validToken,
				Relation: policies.ContributorRelation,
			},
			err:      svcerr.ErrUpdateEntity,
			repoErr:  nil,
			repoErr1: svcerr.ErrUpdateEntity,
		},
		{
			desc:     "accept invitation that is already confirmed",
			token:    validToken,
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: "",
			resp: invitations.Invitation{
				UserID:      userID,
				DomainID:    testsutil.GenerateUUID(t),
				Token:       validToken,
				Relation:    policies.ContributorRelation,
				ConfirmedAt: time.Now(),
			},
			err:     svcerr.ErrInvitationAlreadyAccepted,
			repoErr: nil,
		},
		{
			desc:     "accept rejected invitation",
			token:    validToken,
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: "",
			resp: invitations.Invitation{
				UserID:     userID,
				DomainID:   testsutil.GenerateUUID(t),
				Token:      validToken,
				Relation:   policies.ContributorRelation,
				RejectedAt: time.Now(),
			},
			err:     svcerr.ErrInvitationAlreadyRejected,
			repoErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, tc.domainID).Return(tc.resp, tc.repoErr)
			sdkcall := sdksvc.On("AddUserToDomain", mock.Anything, mock.Anything, mock.Anything).Return(tc.sdkErr)
			repocall2 := repo.On("UpdateConfirmation", context.Background(), mock.Anything).Return(tc.repoErr1)
			err := svc.AcceptInvitation(context.Background(), tc.session, tc.domainID)
			assert.Equal(t, tc.err, err, tc.desc)
			repocall1.Unset()
			sdkcall.Unset()
			repocall2.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	svc := invitations.NewService(token, repo, nil)

	cases := []struct {
		desc     string
		token    string
		userID   string
		domainID string
		resp     invitations.Invitation
		err      error
		repoErr  error
	}{
		{
			desc:     "delete invitations successful",
			userID:   testsutil.GenerateUUID(t),
			domainID: testsutil.GenerateUUID(t),
			resp:     validInvitation,
			err:      nil,
			repoErr:  nil,
		},
		{
			desc:     "delete invitations for the same user",
			token:    validToken,
			userID:   validInvitation.UserID,
			domainID: validInvitation.DomainID,
			resp:     validInvitation,
			err:      nil,
			repoErr:  nil,
		},
		{
			desc:     "delete invitations for the invited user",
			token:    validToken,
			userID:   validInvitation.UserID,
			domainID: validInvitation.DomainID,
			resp:     validInvitation,
			err:      nil,
			repoErr:  nil,
		},
		{
			desc:     "error retrieving invitation",
			token:    validToken,
			userID:   validInvitation.UserID,
			domainID: validInvitation.DomainID,
			resp:     invitations.Invitation{},
			err:      svcerr.ErrNotFound,
			repoErr:  svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.repoErr)
			repocall2 := repo.On("Delete", context.Background(), mock.Anything, mock.Anything).Return(tc.repoErr)
			err := svc.DeleteInvitation(context.Background(), authn.Session{}, tc.userID, tc.domainID)
			assert.Equal(t, tc.err, err, tc.desc)
			repocall1.Unset()
			repocall2.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	token := new(authmocks.TokenServiceClient)
	svc := invitations.NewService(token, repo, nil)
	userID := validInvitation.UserID

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		resp     invitations.Invitation
		err      error
		repoErr  error
		repoErr1 error
	}{
		{
			desc:     "reject invitations for the same user",
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: validInvitation.DomainID,
			resp:     validInvitation,
			err:      nil,
			repoErr:  nil,
			repoErr1: nil,
		},
		{
			desc:     "reject invitations for the invited user",
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: validInvitation.DomainID,
			resp:     invitations.Invitation{},
			err:      svcerr.ErrAuthorization,
			repoErr:  nil,
			repoErr1: nil,
		},
		{
			desc:     "error retrieving invitation",
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: validInvitation.DomainID,
			resp:     invitations.Invitation{},
			err:      repoerr.ErrNotFound,
			repoErr:  repoerr.ErrNotFound,
			repoErr1: nil,
		},
		{
			desc:     "error updating rejection",
			session:  authn.Session{DomainUserID: validDomainUserID, DomainID: validDomainID, UserID: userID},
			domainID: validInvitation.DomainID,
			resp:     validInvitation,
			err:      repoerr.ErrUpdateEntity,
			repoErr:  nil,
			repoErr1: repoerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.repoErr)
			repocall3 := repo.On("UpdateRejection", context.Background(), mock.Anything).Return(tc.repoErr1)
			err := svc.RejectInvitation(context.Background(), tc.session, tc.domainID)
			assert.Equal(t, tc.err, err, tc.desc)
			repocall1.Unset()
			repocall3.Unset()
		})
	}
}
