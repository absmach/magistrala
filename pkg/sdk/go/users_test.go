// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/mainflux/mainflux"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	invalidEmail = "userexample.com"
)

var (
	passRegex = regexp.MustCompile("^.{8,}$")
)

func newUserService() users.Service {
	usersRepo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	auth := mocks.NewAuthService(map[string]string{"user@example.com": "user@example.com"})
	emailer := mocks.NewEmailer()
	idProvider := uuid.New()

	return users.New(usersRepo, hasher, auth, emailer, idProvider, passRegex)
}

func newUserServer(svc users.Service) *httptest.Server {
	mux := api.MakeHandler(svc, mocktracer.New())
	return httptest.NewServer(mux)
}

func TestCreateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		GroupsPrefix:      "",
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
			err:  createError(sdk.ErrFailedCreation, http.StatusConflict),
		},
		{
			desc: "register user with invalid email address",
			user: sdk.User{Email: invalidEmail, Password: "password"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user with empty password",
			user: sdk.User{Email: "user2@example.com", Password: ""},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user without password",
			user: sdk.User{Email: "user2@example.com"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register user without email",
			user: sdk.User{Password: "password"},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc: "register empty user",
			user: sdk.User{},
			err:  createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		_, err := mainfluxSDK.CreateUser(tc.user)
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
		GroupsPrefix:      "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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
			err:   createError(sdk.ErrFailedCreation, http.StatusForbidden),
		},
		{
			desc:  "create user with empty email",
			user:  sdk.User{Email: "", Password: "password"},
			token: "",
			err:   createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		token, err := mainfluxSDK.CreateToken(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.token, token, fmt.Sprintf("%s: expected response: %s, got:  %s", tc.desc, token, tc.token))
	}
}
