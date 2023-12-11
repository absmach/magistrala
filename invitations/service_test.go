// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations_test

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/mocks"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validInvitation = invitations.Invitation{
		UserID:   testsutil.GenerateUUID(&testing.T{}),
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		Relation: auth.ViewerRelation,
	}
	validToken = "token"
)

func TestSendInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := invitations.NewService(repo, authsvc, nil)

	cases := []struct {
		desc        string
		token       string
		tokenUserID string
		req         invitations.Invitation
		err         error
		authNErr    error
		domainErr   error
		adminErr    error
		authorised  bool
		issueErr    error
		repoErr     error
	}{
		{
			desc:        "send invitation successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "invalid token",
			token:       "invalid",
			tokenUserID: "",
			req:         validInvitation,
			err:         svcerr.ErrAuthentication,
			authNErr:    svcerr.ErrAuthentication,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "invalid relation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         invitations.Invitation{Relation: "invalid"},
			err:         errors.New("invalid relation"),
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "error during domain admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         svcerr.ErrAuthorization,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    nil,
			authorised:  false,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "error during platform admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         svcerr.ErrAuthorization,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			issueErr:    nil,
			repoErr:     nil,
		},
		{
			desc:        "error during token issuance",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req:         validInvitation,
			err:         svcerr.ErrAuthentication,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			issueErr:    svcerr.ErrAuthentication,
			repoErr:     nil,
		},
		{
			desc:        "resend invitation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			req: invitations.Invitation{
				UserID:   testsutil.GenerateUUID(t),
				DomainID: testsutil.GenerateUUID(t),
				Relation: auth.ViewerRelation,
				Resend:   true,
			},
			err:        nil,
			authNErr:   nil,
			domainErr:  nil,
			adminErr:   nil,
			authorised: true,
			issueErr:   nil,
			repoErr:    nil,
		},
	}

	for _, tc := range cases {
		idRes := &magistrala.IdentityRes{
			UserId: tc.tokenUserID,
			Id:     testsutil.GenerateUUID(t) + "_" + tc.tokenUserID,
		}
		repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(idRes, tc.authNErr)
		domainReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.DomainType,
			Object:      tc.req.DomainID,
		}
		domaincall := authsvc.On("Authorize", context.Background(), &domainReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.domainErr)
		platformReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		}
		platformcall := authsvc.On("Authorize", context.Background(), &platformReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.adminErr)
		repocall1 := authsvc.On("Issue", context.Background(), mock.Anything).Return(&magistrala.Token{AccessToken: tc.req.Token}, tc.issueErr)
		repocall2 := repo.On("Create", context.Background(), mock.Anything).Return(tc.repoErr)
		if tc.req.Resend {
			repocall2 = repo.On("UpdateToken", context.Background(), mock.Anything).Return(tc.repoErr)
		}
		err := svc.SendInvitation(context.Background(), tc.token, tc.req)
		assert.Equal(t, tc.err, err, tc.desc)
		repocall.Unset()
		domaincall.Unset()
		platformcall.Unset()
		repocall1.Unset()
		repocall2.Unset()
	}
}

func TestViewInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := invitations.NewService(repo, authsvc, nil)

	validInvitation := invitations.Invitation{
		InvitedBy:   testsutil.GenerateUUID(t),
		UserID:      testsutil.GenerateUUID(t),
		DomainID:    testsutil.GenerateUUID(t),
		Relation:    auth.ViewerRelation,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
		ConfirmedAt: time.Now().Add(-time.Hour),
	}
	cases := []struct {
		desc        string
		token       string
		tokenUserID string
		userID      string
		domainID    string
		resp        invitations.Invitation
		err         error
		authNErr    error
		domainErr   error
		adminErr    error
		authorised  bool
		repoErr     error
	}{
		{
			desc:        "view invitation successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "invalid token",
			token:       authmocks.InvalidValue,
			tokenUserID: "",
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        invitations.Invitation{},
			err:         svcerr.ErrAuthentication,
			authNErr:    svcerr.ErrAuthentication,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "error retrieving invitation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        invitations.Invitation{},
			err:         svcerr.ErrNotFound,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     svcerr.ErrNotFound,
		},
		{
			desc:        "valid invitation for the same user",
			token:       validToken,
			tokenUserID: validInvitation.UserID,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "valid invitation for the invited user",
			token:       validToken,
			tokenUserID: validInvitation.InvitedBy,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "valid invitation for the domain admin",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "valid invitation for the platform admin",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "invalid user trying to access invitation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      testsutil.GenerateUUID(t),
			domainID:    testsutil.GenerateUUID(t),
			resp:        invitations.Invitation{},
			err:         svcerr.ErrAuthorization,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
	}

	for _, tc := range cases {
		idRes := &magistrala.IdentityRes{
			UserId: tc.tokenUserID,
			Id:     testsutil.GenerateUUID(t) + "_" + tc.tokenUserID,
		}
		repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(idRes, tc.authNErr)
		domainReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.DomainType,
			Object:      tc.domainID,
		}
		domaincall := authsvc.On("Authorize", context.Background(), &domainReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.domainErr)
		platformReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		}
		platformcall := authsvc.On("Authorize", context.Background(), &platformReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.adminErr)
		repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.repoErr)
		inv, err := svc.ViewInvitation(context.Background(), tc.token, tc.userID, tc.domainID)
		assert.Equal(t, tc.err, err, tc.desc)
		assert.Equal(t, tc.resp, inv, tc.desc)
		repocall.Unset()
		domaincall.Unset()
		platformcall.Unset()
		repocall1.Unset()
	}
}

func TestListInvitations(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := invitations.NewService(repo, authsvc, nil)

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
				Relation:    auth.ViewerRelation,
				CreatedAt:   time.Now().Add(-time.Hour),
				UpdatedAt:   time.Now().Add(-time.Hour),
				ConfirmedAt: time.Now().Add(-time.Hour),
			},
		},
	}

	cases := []struct {
		desc        string
		token       string
		tokenUserID string
		page        invitations.Page
		resp        invitations.InvitationPage
		err         error
		authNErr    error
		domainErr   error
		adminErr    error
		authorised  bool
		repoErr     error
	}{
		{
			desc:        "list invitations successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        validPage,
			resp:        validResp,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "invalid token",
			token:       "invalid",
			tokenUserID: "",
			err:         svcerr.ErrAuthentication,
			authNErr:    svcerr.ErrAuthentication,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "error during platform admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        validPage,
			err:         nil,
			resp:        invitations.InvitationPage{},
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "list invitations with admin successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        invitations.Page{DomainID: testsutil.GenerateUUID(t)},
			resp:        validResp,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "error during platform admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        validPage,
			err:         nil,
			resp:        validResp,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "list invitations with domain successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        invitations.Page{DomainID: testsutil.GenerateUUID(t)},
			resp:        validResp,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "list invitations with domain_id and error during domain admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        invitations.Page{DomainID: testsutil.GenerateUUID(t)},
			err:         svcerr.ErrAuthorization,
			resp:        invitations.InvitationPage{},
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "list invitations with domain_id and error during platform admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			page:        invitations.Page{DomainID: testsutil.GenerateUUID(t)},
			err:         svcerr.ErrAuthorization,
			resp:        invitations.InvitationPage{},
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
	}

	for _, tc := range cases {
		idRes := &magistrala.IdentityRes{
			UserId: tc.tokenUserID,
			Id:     testsutil.GenerateUUID(t) + "_" + tc.tokenUserID,
		}
		repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(idRes, tc.authNErr)
		domainReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.DomainType,
			Object:      tc.page.DomainID,
		}
		domaincall := authsvc.On("Authorize", context.Background(), &domainReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.domainErr)
		platformReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		}
		platformcall := authsvc.On("Authorize", context.Background(), &platformReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.adminErr)
		repocall1 := repo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.resp, tc.repoErr)
		resp, err := svc.ListInvitations(context.Background(), tc.token, tc.page)
		assert.Equal(t, tc.err, err, tc.desc)
		assert.Equal(t, tc.resp, resp, tc.desc)
		repocall.Unset()
		domaincall.Unset()
		platformcall.Unset()
		repocall1.Unset()
	}
}

func TestAcceptInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := invitations.NewService(repo, authsvc, nil)

	userID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc        string
		token       string
		tokenUserID string
		domainID    string
		resp        invitations.Invitation
		err         error
		authNErr    error
		domainErr   error
		adminErr    error
		authorised  bool
		repoErr     error
	}{
		{
			desc:        "invalid token",
			token:       "invalid",
			tokenUserID: "",
			err:         svcerr.ErrAuthentication,
			authNErr:    svcerr.ErrAuthentication,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "list invitations successful that have been confirmed",
			token:       validToken,
			tokenUserID: userID,
			domainID:    "",
			resp: invitations.Invitation{
				UserID:      userID,
				DomainID:    testsutil.GenerateUUID(t),
				Token:       validToken,
				Relation:    auth.ViewerRelation,
				ConfirmedAt: time.Now().Add(-time.Second * time.Duration(rand.Intn(100))),
			},
			err:        nil,
			authNErr:   nil,
			domainErr:  nil,
			adminErr:   nil,
			authorised: true,
			repoErr:    nil,
		},
		{
			desc:        "list invitations with failed to retrieve all",
			token:       validToken,
			tokenUserID: userID,
			err:         svcerr.ErrNotFound,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			repoErr:     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{UserId: tc.tokenUserID}, tc.authNErr)
		repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, tc.domainID).Return(tc.resp, tc.repoErr)
		err := svc.AcceptInvitation(context.Background(), tc.token, tc.domainID)
		assert.Equal(t, tc.err, err, tc.desc)
		repocall.Unset()
		repocall1.Unset()
	}
}

func TestDeleteInvitation(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := invitations.NewService(repo, authsvc, nil)

	cases := []struct {
		desc        string
		token       string
		tokenUserID string
		userID      string
		domainID    string
		resp        invitations.Invitation
		err         error
		authNErr    error
		domainErr   error
		adminErr    error
		authorised  bool
		repoErr     error
	}{
		{
			desc:        "delete invitations successful",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      testsutil.GenerateUUID(t),
			domainID:    testsutil.GenerateUUID(t),
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "invalid token",
			token:       "invalid",
			tokenUserID: "",
			userID:      testsutil.GenerateUUID(t),
			domainID:    testsutil.GenerateUUID(t),
			err:         svcerr.ErrAuthentication,
			authNErr:    svcerr.ErrAuthentication,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "delete invitations for the same user",
			token:       validToken,
			tokenUserID: validInvitation.UserID,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "delete invitations for the invited user",
			token:       validToken,
			tokenUserID: validInvitation.InvitedBy,
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        validInvitation,
			err:         nil,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     nil,
		},
		{
			desc:        "error retrieving invitation",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      validInvitation.UserID,
			domainID:    validInvitation.DomainID,
			resp:        invitations.Invitation{},
			err:         svcerr.ErrNotFound,
			authNErr:    nil,
			domainErr:   nil,
			adminErr:    nil,
			authorised:  true,
			repoErr:     svcerr.ErrNotFound,
		},
		{
			desc:        "error during domain admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      testsutil.GenerateUUID(t),
			domainID:    testsutil.GenerateUUID(t),
			resp:        invitations.Invitation{},
			err:         svcerr.ErrAuthorization,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    nil,
			authorised:  false,
			repoErr:     nil,
		},
		{
			desc:        "error during platform admin check",
			token:       validToken,
			tokenUserID: testsutil.GenerateUUID(t),
			userID:      testsutil.GenerateUUID(t),
			domainID:    testsutil.GenerateUUID(t),
			resp:        invitations.Invitation{},
			err:         svcerr.ErrAuthorization,
			authNErr:    nil,
			domainErr:   svcerr.ErrAuthorization,
			adminErr:    svcerr.ErrAuthorization,
			authorised:  false,
			repoErr:     nil,
		},
	}

	for _, tc := range cases {
		idRes := &magistrala.IdentityRes{
			UserId: tc.tokenUserID,
			Id:     tc.domainID + "_" + tc.userID,
		}
		repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(idRes, tc.authNErr)
		domainReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.DomainType,
			Object:      tc.domainID,
		}
		domaincall := authsvc.On("Authorize", context.Background(), &domainReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.domainErr)
		platformReq := magistrala.AuthorizeReq{
			SubjectType: auth.UserType,
			SubjectKind: auth.UsersKind,
			Subject:     idRes.GetId(),
			Permission:  auth.AdminPermission,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		}
		platformcall := authsvc.On("Authorize", context.Background(), &platformReq).Return(&magistrala.AuthorizeRes{Authorized: tc.authorised}, tc.adminErr)
		repocall1 := repo.On("Retrieve", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.repoErr)
		repocall2 := repo.On("Delete", context.Background(), mock.Anything, mock.Anything).Return(tc.repoErr)
		err := svc.DeleteInvitation(context.Background(), tc.token, tc.userID, tc.domainID)
		assert.Equal(t, tc.err, err, tc.desc)
		repocall.Unset()
		repocall1.Unset()
		domaincall.Unset()
		platformcall.Unset()
		repocall2.Unset()
	}
}
