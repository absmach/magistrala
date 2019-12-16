// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	sdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"

	"github.com/mainflux/mainflux/users/api"

	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/mocks"
)

const (
	invalidEmail = "userexample.com"
)

func newUserService() users.Service {
	repo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	auth := mocks.NewAuthService(map[string]string{"user@example.com": "user@example.com"})

	emailer := mocks.NewEmailer()

	return users.New(repo, hasher, auth, emailer)
}

func newUserServer(svc users.Service) *httptest.Server {
	logger, _ := log.New(os.Stdout, log.Info.String())
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func TestCreateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}
	cases := []struct {
		desc string
		user sdk.User
		err  error
	}{
		{
			desc: "register new user",
			user: user,
			err:  nil,
		},
		{
			desc: "register existing user",
			user: user,
			err:  sdk.ErrConflict,
		},
		{
			desc: "register user with invalid email address",
			user: sdk.User{Email: invalidEmail, Password: "password"},
			err:  sdk.ErrInvalidArgs,
		},
		{
			desc: "register user with empty password",
			user: sdk.User{Email: "user2@example.com", Password: ""},
			err:  sdk.ErrInvalidArgs,
		},
		{
			desc: "register user without password",
			user: sdk.User{Email: "user2@example.com"},
			err:  sdk.ErrInvalidArgs,
		},
		{
			desc: "register user without email",
			user: sdk.User{Password: "password"},
			err:  sdk.ErrInvalidArgs,
		},
		{
			desc: "register empty user",
			user: sdk.User{},
			err:  sdk.ErrInvalidArgs,
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.CreateUser(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
	}
}

func TestCreateToken(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 0})
	token := tkn.GetValue()
	mainfluxSDK.CreateUser(user)
	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   error
	}{
		{
			desc:  "create token for user",
			user:  user,
			token: token,
			err:   nil,
		},
		{
			desc:  "create token for non existing user",
			user:  sdk.User{Email: "user2@example.com", Password: "password"},
			token: "",
			err:   sdk.ErrUnauthorized,
		},
		{
			desc:  "create user with empty email",
			user:  sdk.User{Email: "", Password: "password"},
			token: "",
			err:   sdk.ErrInvalidArgs,
		},
	}
	for _, tc := range cases {
		token, err := mainfluxSDK.CreateToken(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.token, token, fmt.Sprintf("%s: expected response: %s, got:  %s", tc.desc, token, tc.token))
	}
}
