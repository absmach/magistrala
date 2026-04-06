// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/internal/testsutil"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	sdkInvitation = generateTestInvitation(&testing.T{})
	invitation    = convertInvitation(sdkInvitation)
)

func TestSendInvitation(t *testing.T) {
	is, svc, auth := setupDomains()
	defer is.Close()

	conf := sdk.Config{
		DomainsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	sendInvitationReq := sdk.Invitation{
		InviteeUserID: invitation.InviteeUserID,
		DomainID:      invitation.DomainID,
		RoleID:        invitation.RoleID,
	}

	cases := []struct {
		desc              string
		token             string
		session           smqauthn.Session
		sendInvitationReq sdk.Invitation
		svcReq            domains.Invitation
		authenticateErr   error
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
			authenticateErr:   svcerr.ErrAuthentication,
			err:               errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:              "send invitation with empty token",
			token:             "",
			sendInvitationReq: sendInvitationReq,
			svcReq:            domains.Invitation{},
			svcErr:            nil,
			err:               errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "send invitation with empty userID",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				InviteeUserID: "",
				DomainID:      invitation.DomainID,
				RoleID:        invitation.RoleID,
			},
			svcReq: domains.Invitation{},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:  "send invitation with empty role ID",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      invitation.DomainID,
				RoleID:        "",
			},
			svcReq: domains.Invitation{},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:  "send inviation with invalid domainID",
			token: validToken,
			sendInvitationReq: sdk.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      wrongID,
				RoleID:        invitation.RoleID,
			},
			svcReq: domains.Invitation{
				InviteeUserID: invitation.InviteeUserID,
				DomainID:      wrongID,
				RoleID:        invitation.RoleID,
			},
			svcErr: svcerr.ErrCreateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = smqauthn.Session{
					UserID:       tc.sendInvitationReq.InviteeUserID,
					DomainID:     tc.sendInvitationReq.DomainID,
					DomainUserID: fmt.Sprintf("%s_%s", tc.sendInvitationReq.DomainID, tc.sendInvitationReq.InviteeUserID),
				}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("SendInvitation", mock.Anything, tc.session, tc.svcReq).Return(domains.Invitation{}, tc.svcErr)
			err := mgsdk.SendInvitation(context.Background(), tc.sendInvitationReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "SendInvitation", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListInvitation(t *testing.T) {
	is, svc, auth := setupDomains()
	defer is.Close()

	conf := sdk.Config{
		DomainsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          domains.InvitationPageMeta
		svcRes          domains.InvitationPage
		svcErr          error
		authenticateErr error
		response        sdk.InvitationPage
		err             error
	}{
		{
			desc:  "list invitations successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: domains.InvitationPageMeta{
				Offset: 0,
				Limit:  10,
			},
			svcRes: domains.InvitationPage{
				Total:       1,
				Invitations: []domains.Invitation{invitation},
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
			svcReq: domains.InvitationPageMeta{
				Offset: 0,
				Limit:  10,
			},
			svcRes:          domains.InvitationPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.InvitationPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list invitations with empty token",
			token:    "",
			pageMeta: sdk.PageMetadata{},
			svcRes:   domains.InvitationPage{},
			svcErr:   nil,
			response: sdk.InvitationPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "list invitations with limit greater than max limit",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  101,
			},
			svcReq:   domains.InvitationPageMeta{},
			svcRes:   domains.InvitationPage{},
			svcErr:   nil,
			response: sdk.InvitationPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = smqauthn.Session{DomainUserID: validID, UserID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ListInvitations", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Invitations(context.Background(), tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListInvitations", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	is, svc, auth := setupDomains()
	defer is.Close()

	conf := sdk.Config{
		DomainsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		authenticateErr error
		svcErr          error
		err             error
	}{
		{
			desc:     "accept invitation successfully",
			token:    validToken,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "accept invitation with invalid token",
			token:           invalidToken,
			domainID:        invitation.DomainID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "accept invitation with empty token",
			token:    "",
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
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
			if tc.token == valid {
				tc.session = smqauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("AcceptInvitation", mock.Anything, tc.session, tc.domainID).Return(domains.Invitation{}, tc.svcErr)
			err := mgsdk.AcceptInvitation(context.Background(), tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AcceptInvitation", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	is, svc, auth := setupDomains()
	defer is.Close()

	conf := sdk.Config{
		DomainsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		authenticateErr error
		svcErr          error
		err             error
	}{
		{
			desc:     "reject invitation successfully",
			token:    validToken,
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "reject invitation with invalid token",
			token:           invalidToken,
			domainID:        invitation.DomainID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "reject invitation with empty token",
			token:    "",
			domainID: invitation.DomainID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
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
			if tc.token == valid {
				tc.session = smqauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("RejectInvitation", mock.Anything, tc.session, tc.domainID).Return(domains.Invitation{}, tc.svcErr)
			err := mgsdk.RejectInvitation(context.Background(), tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RejectInvitation", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	is, svc, auth := setupDomains()
	defer is.Close()

	conf := sdk.Config{
		DomainsURL: is.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		inviteeUserID   string
		domainID        string
		authenticateErr error
		svcErr          error
		err             error
	}{
		{
			desc:          "delete invitation successfully",
			token:         validToken,
			inviteeUserID: invitation.InviteeUserID,
			domainID:      invitation.DomainID,
			svcErr:        nil,
			err:           nil,
		},
		{
			desc:            "delete invitation with invalid token",
			token:           invalidToken,
			inviteeUserID:   invitation.InviteeUserID,
			domainID:        invitation.DomainID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:          "delete invitation with empty token",
			token:         "",
			inviteeUserID: invitation.InviteeUserID,
			domainID:      invitation.DomainID,
			svcErr:        nil,
			err:           errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:          "delete invitation with empty domainID",
			token:         validToken,
			inviteeUserID: invitation.InviteeUserID,
			domainID:      "",
			svcErr:        nil,
			err:           errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:          "delete invitation with invalid domainID",
			token:         validToken,
			inviteeUserID: invitation.InviteeUserID,
			domainID:      wrongID,
			svcErr:        svcerr.ErrNotFound,
			err:           errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = smqauthn.Session{UserID: tc.inviteeUserID, DomainID: tc.domainID, DomainUserID: fmt.Sprintf("%s_%s", tc.domainID, tc.inviteeUserID)}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("DeleteInvitation", mock.Anything, tc.session, tc.inviteeUserID, tc.domainID).Return(tc.svcErr)
			err := mgsdk.DeleteInvitation(context.Background(), tc.inviteeUserID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteInvitation", mock.Anything, tc.session, tc.inviteeUserID, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func generateTestInvitation(t *testing.T) sdk.Invitation {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return sdk.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      testsutil.GenerateUUID(t),
		RoleID:        testsutil.GenerateUUID(t),
		RoleName:      "admin",
		Actions:       []string{"read", "update"},
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
}
