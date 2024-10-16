// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/api"
	"github.com/absmach/magistrala/invitations/mocks"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policies "github.com/absmach/magistrala/pkg/policies"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	sdkInvitation = generateTestInvitation(&testing.T{})
	invitation    = convertInvitation(sdkInvitation)
)

func setupInvitations() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)
	logger := mglog.NewMock()

	mux := api.MakeHandler(svc, logger, "test")
	return httptest.NewServer(mux), svc
}

func TestSendInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	sendInvitationReq := sdk.Invitation{
		UserID:   invitation.UserID,
		DomainID: invitation.DomainID,
		Relation: invitation.Relation,
		Resend:   invitation.Resend,
	}

	cases := []struct {
		desc              string
		token             string
		sendInvitationReq sdk.Invitation
		svcReq            invitations.Invitation
		svcErr            error
		err               error
	}{
		{
			desc:              "send invitation successfully",
			token:             validToken,
			sendInvitationReq: sendInvitationReq,
			svcReq:            convertInvitation(sendInvitationReq),
			svcErr:            nil,
			err:               nil,
		},
		{
			desc:              "send invitation with invalid token",
			token:             invalidToken,
			sendInvitationReq: sendInvitationReq,
			svcReq:            convertInvitation(sendInvitationReq),
			svcErr:            svcerr.ErrAuthentication,
			err:               errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:              "send invitation with empty token",
			token:             "",
			sendInvitationReq: sendInvitationReq,
			svcReq:            invitations.Invitation{},
			svcErr:            nil,
			err:               errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "send invitation with empty userID",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				UserID:   "",
				DomainID: invitation.DomainID,
				Relation: invitation.Relation,
				Resend:   invitation.Resend,
			},
			svcReq: invitations.Invitation{},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "send invitation with invalid relation",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				UserID:   invitation.UserID,
				DomainID: invitation.DomainID,
				Relation: "invalid",
				Resend:   invitation.Resend,
			},
			svcReq: invitations.Invitation{},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidRelation), http.StatusInternalServerError),
		},
		{
			desc:  "send inviation with invalid domainID",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				UserID:   invitation.UserID,
				DomainID: wrongID,
				Relation: invitation.Relation,
				Resend:   invitation.Resend,
			},
			svcReq: invitations.Invitation{
				UserID:   invitation.UserID,
				DomainID: wrongID,
				Relation: invitation.Relation,
				Resend:   invitation.Resend,
			},
			svcErr: svcerr.ErrCreateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SendInvitation", mock.Anything, tc.token, tc.svcReq).Return(tc.svcErr)
			err := mgsdk.SendInvitation(tc.sendInvitationReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "SendInvitation", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		userID   string
		domainID string
		svcRes   invitations.Invitation
		svcErr   error
		response sdk.Invitation
		err      error
	}{
		{
			desc:     "view invitation successfully",
			token:    validToken,
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcRes:   invitation,
			svcErr:   nil,
			response: sdkInvitation,
			err:      nil,
		},
		{
			desc:     "view invitation with invalid token",
			token:    invalidToken,
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcRes:   invitations.Invitation{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Invitation{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view invitation with empty token",
			token:    "",
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcRes:   invitations.Invitation{},
			svcErr:   nil,
			response: sdk.Invitation{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view invitation with empty userID",
			token:    validToken,
			userID:   "",
			domainID: invitation.DomainID,
			svcRes:   invitations.Invitation{},
			svcErr:   nil,
			response: sdk.Invitation{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "view invitation with invalid domainID",
			token:    validToken,
			userID:   invitation.UserID,
			domainID: wrongID,
			svcRes:   invitations.Invitation{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Invitation{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewInvitation", mock.Anything, tc.token, tc.userID, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Invitation(tc.userID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewInvitation", mock.Anything, tc.token, tc.userID, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   invitations.Page
		svcRes   invitations.InvitationPage
		svcErr   error
		response sdk.InvitationPage
		err      error
	}{
		{
			desc:  "list invitations successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: invitations.Page{
				Offset: 0,
				Limit:  10,
			},
			svcRes: invitations.InvitationPage{
				Total:       1,
				Invitations: []invitations.Invitation{invitation},
			},
			svcErr: nil,
			response: sdk.InvitationPage{
				Total:       1,
				Invitations: []sdk.Invitation{sdkInvitation},
			},
			err: nil,
		},
		{
			desc:  "list invitations with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: invitations.Page{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   invitations.InvitationPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.InvitationPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list invitations with empty token",
			token:    "",
			pageMeta: sdk.PageMetadata{},
			svcRes:   invitations.InvitationPage{},
			svcErr:   nil,
			response: sdk.InvitationPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list invitations with limit greater than max limit",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  101,
			},
			svcReq:   invitations.Page{},
			svcRes:   invitations.InvitationPage{},
			svcErr:   nil,
			response: sdk.InvitationPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListInvitations", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Invitations(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListInvitations", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcErr   error
		err      error
	}{
		{
			desc:     "accept invitation successfully",
			token:    validToken,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "accept invitation with invalid token",
			token:    invalidToken,
			domainID: invitation.DomainID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "accept invitation with empty token",
			token:    "",
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "accept invitation with invalid domainID",
			token:    validToken,
			domainID: wrongID,
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AcceptInvitation", mock.Anything, tc.token, tc.domainID).Return(tc.svcErr)
			err := mgsdk.AcceptInvitation(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AcceptInvitation", mock.Anything, tc.token, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcErr   error
		err      error
	}{
		{
			desc:     "reject invitation successfully",
			token:    validToken,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "reject invitation with invalid token",
			token:    invalidToken,
			domainID: invitation.DomainID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "reject invitation with empty token",
			token:    "",
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "reject invitation with invalid domainID",
			token:    validToken,
			domainID: wrongID,
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RejectInvitation", mock.Anything, tc.token, tc.domainID).Return(tc.svcErr)
			err := mgsdk.RejectInvitation(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RejectInvitation", mock.Anything, tc.token, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	is, svc := setupInvitations()
	defer is.Close()

	conf := sdk.Config{
		InvitationsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		userID   string
		domainID string
		svcErr   error
		err      error
	}{
		{
			desc:     "delete invitation successfully",
			token:    validToken,
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "delete invitation with invalid token",
			token:    invalidToken,
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "delete invitation with empty token",
			token:    "",
			userID:   invitation.UserID,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "delete invitation with empty userID",
			token:    validToken,
			userID:   "",
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "delete invitation with invalid domainID",
			token:    validToken,
			userID:   invitation.UserID,
			domainID: wrongID,
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteInvitation", mock.Anything, tc.token, tc.userID, tc.domainID).Return(tc.svcErr)
			err := mgsdk.DeleteInvitation(tc.userID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteInvitation", mock.Anything, tc.token, tc.userID, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func generateTestInvitation(t *testing.T) sdk.Invitation {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return sdk.Invitation{
		InvitedBy: testsutil.GenerateUUID(t),
		UserID:    testsutil.GenerateUUID(t),
		DomainID:  testsutil.GenerateUUID(t),
		Token:     validToken,
		Relation:  policies.MemberRelation,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Resend:    false,
	}
}
