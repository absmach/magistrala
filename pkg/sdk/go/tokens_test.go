// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
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
			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusInternalServerError),
		},
		{
			desc:     "issue token for invalid identity",
			login:    sdk.Login{Identity: "invalid", Secret: "secret"},
			token:    &magistrala.Token{},
			dbClient: wrongClient,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(repoerr.ErrNotFound, repoerr.ErrNotFound), http.StatusNotFound),
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
	ts, cRepo, _, auth := setupUsers()
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
		desc   string
		token  string
		rtoken *magistrala.Token
		err    errors.SDKError
	}{
		{
			desc:  "refresh token for a valid refresh token",
			token: token,
			rtoken: &magistrala.Token{
				AccessToken:  validToken,
				RefreshToken: &validToken,
				AccessType:   "Bearer",
			},
			err: nil,
		},
		{
			desc:   "refresh token for an empty token",
			token:  "",
			rtoken: &magistrala.Token{},
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(convertClient(user), tc.err)
		repoCall1 := auth.On("Refresh", mock.Anything, mock.Anything).Return(tc.rtoken, nil)
		_, err := mgsdk.RefreshToken(sdk.Login{}, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}
