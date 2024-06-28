// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIssueToken(t *testing.T) {
	ts, cRepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	client := sdk.User{
		ID: generateUUID(t),
		Credentials: sdk.Credentials{
			Identity: "valid@example.com",
			Secret:   "secret",
		},
		Status: sdk.EnabledStatus,
	}
	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	wrongClient := client
	wrongClient.Credentials.Secret, _ = phasher.Hash("wrong")

	cases := []struct {
		desc     string
		login    sdk.Login
		token    *magistrala.Token
		dbClient sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "issue token for a new user",
			login:    sdk.Login{Identity: client.Credentials.Identity, Secret: client.Credentials.Secret},
			dbClient: rClient,
			token: &magistrala.Token{
				AccessToken:  validToken,
				RefreshToken: &validToken,
				AccessType:   "Bearer",
			},
			err: nil,
		},
		{
			desc:  "issue token for an empty user",
			login: sdk.Login{},
			token: &magistrala.Token{},
			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc:     "issue token for invalid identity",
			login:    sdk.Login{Identity: "invalid", Secret: "secret"},
			token:    &magistrala.Token{},
			dbClient: wrongClient,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		repoCall := auth.On("Issue", mock.Anything, mock.Anything).Return(tc.token, nil)
		repoCall1 := cRepo.On("RetrieveByIdentity", mock.Anything, mock.Anything).Return(convertClient(tc.dbClient), tc.err)
		token, err := mgsdk.CreateToken(tc.login)
		switch tc.err {
		case nil:
			assert.NotEmpty(t, token, fmt.Sprintf("%s: expected token, got empty", tc.desc))
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
		default:
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user := sdk.User{
		ID:   generateUUID(t),
		Name: "validtoken",
		Credentials: sdk.Credentials{
			Identity: "validtoken",
			Secret:   "secret",
		},
		Status: sdk.EnabledStatus,
	}
	rUser := user
	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

	cases := []struct {
		desc         string
		token        string
		domainID     string
		identifyResp *magistrala.IdentityRes
		identifyErr  error
		refreshResp  *magistrala.Token
		refresErr    error
		repoResp     mgclients.Client
		repoErr      error
		err          error
	}{
		{
			desc:         "refresh token with refresh token for an existing client",
			token:        token,
			identifyResp: &magistrala.IdentityRes{UserId: user.ID},
			refreshResp:  &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "Bearer"},
			repoResp:     convertClient(rUser),
			err:          nil,
		},
		{
			desc:         "refresh token with refresh token for an existing client with domain",
			token:        token,
			domainID:     "domain",
			identifyResp: &magistrala.IdentityRes{UserId: user.ID},
			refreshResp:  &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "Bearer"},
			repoResp:     convertClient(rUser),
			err:          nil,
		},
		{
			desc:        "refresh token for an empty token",
			token:       "",
			identifyErr: svcerr.ErrAuthentication,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:         "refresh token with invalid token",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{},
			identifyErr:  svcerr.ErrAuthentication,
			err:          errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:         "refresh token with refresh token for a disable client",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{UserId: user.ID},
			repoResp:     mgclients.Client{Status: mgclients.DisabledStatus},
			err:          errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:         "refresh token with empty domain id",
			token:        validToken,
			identifyResp: &magistrala.IdentityRes{UserId: user.ID},
			refreshResp:  &magistrala.Token{},
			refresErr:    svcerr.ErrAuthentication,
			repoResp:     convertClient(rUser),
			err:          errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResp, tc.identifyErr)
		authCall1 := auth.On("Refresh", mock.Anything, &magistrala.RefreshReq{RefreshToken: tc.token, DomainId: &tc.domainID}).Return(tc.refreshResp, tc.refresErr)
		repoCall := crepo.On("RetrieveByID", mock.Anything, tc.identifyResp.GetUserId()).Return(tc.repoResp, tc.repoErr)
		token, err := mgsdk.RefreshToken(sdk.Login{DomainID: tc.domainID}, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.AccessToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.AccessToken))
			assert.NotEmpty(t, token.RefreshToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.RefreshToken))
			ok := authCall.Parent.AssertCalled(t, "Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token})
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = authCall.Parent.AssertCalled(t, "Refresh", mock.Anything, &magistrala.RefreshReq{RefreshToken: tc.token, DomainId: &tc.domainID})
			assert.True(t, ok, fmt.Sprintf("Refresh was not called on %s", tc.desc))
			ok = repoCall.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.identifyResp.UserId)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}
