// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"net/http"
	"testing"

	mgauth "github.com/absmach/magistrala/auth"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIssueToken(t *testing.T) {
	ts, svc, _ := setupUsers()
	defer ts.Close()

	client := generateTestUser(t)
	token := generateTestToken()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		login    sdk.Login
		svcRes   *grpcTokenV1.Token
		svcErr   error
		response sdk.Token
		err      errors.SDKError
	}{
		{
			desc: "issue token successfully",
			login: sdk.Login{
				Identity: client.Credentials.Username,
				Secret:   client.Credentials.Secret,
			},
			svcRes: &grpcTokenV1.Token{
				AccessToken:  token.AccessToken,
				RefreshToken: &token.RefreshToken,
				AccessType:   mgauth.AccessKey.String(),
			},
			svcErr:   nil,
			response: token,
			err:      nil,
		},
		{
			desc: "issue token with invalid identity",
			login: sdk.Login{
				Identity: invalidIdentity,
				Secret:   client.Credentials.Secret,
			},
			svcRes:   &grpcTokenV1.Token{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Token{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc: "issue token with invalid secret",
			login: sdk.Login{
				Identity: client.Credentials.Username,
				Secret:   "invalid",
			},
			svcRes:   &grpcTokenV1.Token{},
			svcErr:   svcerr.ErrLogin,
			response: sdk.Token{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrLogin, http.StatusUnauthorized),
		},
		{
			desc: "issue token with empty identity",
			login: sdk.Login{
				Identity: "",
				Secret:   client.Credentials.Secret,
			},
			svcRes:   &grpcTokenV1.Token{},
			svcErr:   nil,
			response: sdk.Token{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc: "issue token with empty secret",
			login: sdk.Login{
				Identity: client.Credentials.Username,
				Secret:   "",
			},
			svcRes:   &grpcTokenV1.Token{},
			svcErr:   nil,
			response: sdk.Token{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("IssueToken", mock.Anything, tc.login.Identity, tc.login.Secret).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateToken(tc.login)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "IssueToken", mock.Anything, tc.login.Identity, tc.login.Secret)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestRefreshToken(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	token := generateTestToken()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc        string
		token       string
		svcRes      *grpcTokenV1.Token
		svcErr      error
		identifyErr error
		response    sdk.Token
		err         errors.SDKError
	}{
		{
			desc:  "refresh token successfully",
			token: token.RefreshToken,
			svcRes: &grpcTokenV1.Token{
				AccessToken:  token.AccessToken,
				RefreshToken: &token.RefreshToken,
				AccessType:   token.AccessType,
			},
			response: token,
			err:      nil,
		},
		{
			desc:        "refresh token with invalid token",
			token:       invalidToken,
			svcRes:      nil,
			identifyErr: svcerr.ErrAuthentication,
			response:    sdk.Token{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "refresh token with empty token",
			token:    "",
			response: sdk.Token{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}, tc.identifyErr)
			svcCall := svc.On("RefreshToken", mock.Anything, mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}, tc.token).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.RefreshToken(tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RefreshToken", mock.Anything, mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}, tc.token)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func generateTestToken() sdk.Token {
	return sdk.Token{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		AccessType:   mgauth.AccessKey.String(),
	}
}
