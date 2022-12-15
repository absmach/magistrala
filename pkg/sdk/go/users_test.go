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
	mfauth "github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	invalidEmail = "userexample.com"
)

var (
	passRegex        = regexp.MustCompile("^.{8,}$")
	limit     uint64 = 5
	offset    uint64 = 0
	total     uint64 = 200
)

func newUserService() users.Service {
	usersRepo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	userEmail := "user@example.com"

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[userEmail] = append(mockAuthzDB[userEmail], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{userEmail: userEmail}, mockAuthzDB)

	emailer := mocks.NewEmailer()
	idProvider := uuid.New()

	return users.New(usersRepo, hasher, auth, emailer, idProvider, passRegex)
}

func newUserServer(svc users.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func TestCreateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()

	mainfluxSDK := sdk.NewSDK(sdkConf)
	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   errors.SDKError
	}{
		{
			desc:  "register new user",
			user:  user,
			token: token,
			err:   nil,
		},
		{
			desc:  "register existing user",
			user:  user,
			token: token,
			err:   errors.NewSDKErrorWithStatus(errors.ErrConflict, http.StatusConflict),
		},
		{
			desc:  "register user with invalid email address",
			user:  sdk.User{Email: invalidEmail, Password: "password"},
			token: token,
			err:   errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
		},
		{
			desc:  "register user with empty password",
			user:  sdk.User{Email: "user2@example.com", Password: ""},
			token: token,
			err:   errors.NewSDKErrorWithStatus(users.ErrPasswordFormat, http.StatusBadRequest),
		},
		{
			desc:  "register user without password",
			user:  sdk.User{Email: "user2@example.com"},
			token: token,
			err:   errors.NewSDKErrorWithStatus(users.ErrPasswordFormat, http.StatusBadRequest),
		},
		{
			desc:  "register user without email",
			user:  sdk.User{Password: "password"},
			token: token,
			err:   errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
		},
		{
			desc:  "register empty user",
			user:  sdk.User{},
			token: token,
			err:   errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		_, err := mainfluxSDK.CreateUser(tc.token, tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
	}
}

func TestUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()
	userID, err := mainfluxSDK.CreateUser(token, user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	usertoken, err := mainfluxSDK.CreateToken(user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	user.ID = userID
	user.Password = ""

	cases := []struct {
		desc     string
		userID   string
		token    string
		err      errors.SDKError
		response sdk.User
	}{
		{
			desc:     "get existing user",
			userID:   userID,
			token:    usertoken,
			err:      nil,
			response: user,
		},
		{
			desc:     "get non-existent user",
			userID:   "43",
			token:    usertoken,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
			response: sdk.User{},
		},

		{
			desc:     "get user with invalid token",
			userID:   userID,
			token:    wrongValue,
			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			response: sdk.User{},
		},
	}
	for _, tc := range cases {
		respUs, err := mainfluxSDK.User(tc.userID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respUs, fmt.Sprintf("%s: expected response user %s, got %s", tc.desc, tc.response, respUs))
	}
}

func TestUsers(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()

	var users []sdk.User

	for i := 10; i < 100; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		password := fmt.Sprintf("password%d", i)
		metadata := map[string]interface{}{"name": fmt.Sprintf("user%d", i)}
		us := sdk.User{Email: email, Password: password, Metadata: metadata}
		userID, err := mainfluxSDK.CreateUser(token, us)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		us.ID = userID
		us.Password = ""
		users = append(users, us)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		err      errors.SDKError
		response []sdk.User
		email    string
		metadata map[string]interface{}
	}{
		{
			desc:     "get a list of users",
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			email:    "",
			response: users[offset:limit],
		},
		{
			desc:     "get a list of users with invalid token",
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			email:    "",
			response: nil,
		},
		{
			desc:     "get a list of users with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			email:    "",
			response: nil,
		},
		{
			desc:     "get a list of users with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			email:    "",
			response: nil,
		},
		{
			desc:     "get a list of users with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			email:    "",
			response: []sdk.User(nil),
		},
		{
			desc:     "get a list of users with same email address",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			email:    "user99@example.com",
			metadata: make(map[string]interface{}),
			response: []sdk.User{users[89]},
		},
		{
			desc:   "get a list of users with same email address and metadata",
			token:  token,
			offset: 0,
			limit:  1,
			err:    nil,
			email:  "user99@example.com",
			metadata: map[string]interface{}{
				"name": "user99",
			},
			response: []sdk.User{users[89]},
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Email:    tc.email,
			Total:    total,
			Offset:   uint64(tc.offset),
			Limit:    uint64(tc.limit),
			Metadata: tc.metadata,
		}
		page, err := mainfluxSDK.Users(tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Users, fmt.Sprintf("%s: expected response user %s, got %s", tc.desc, tc.response, page.Users))
	}
}

func TestCreateToken(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()
	_, err := mainfluxSDK.CreateUser(token, user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   errors.SDKError
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
			err:   errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "create user with empty email",
			user:  sdk.User{Email: "", Password: "password"},
			token: "",
			err:   errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		token, err := mainfluxSDK.CreateToken(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.token, token, fmt.Sprintf("%s: expected response: %s, got:  %s", tc.desc, token, tc.token))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()
	userID, err := mainfluxSDK.CreateUser(token, user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	usertoken, err := mainfluxSDK.CreateToken(user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		user  sdk.User
		token string
		err   errors.SDKError
	}{
		{
			desc:  "update email for user",
			user:  sdk.User{ID: userID, Email: "user2@example.com", Password: "password"},
			token: usertoken,
			err:   nil,
		},
		{
			desc:  "update email for user with invalid token",
			user:  sdk.User{ID: userID, Email: "user2@example.com", Password: "password"},
			token: wrongValue,
			err:   errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update email for user with empty token",
			user:  sdk.User{ID: userID, Email: "user2@example.com", Password: "password"},
			token: "",
			err:   errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update metadata for user",
			user:  sdk.User{ID: userID, Metadata: metadata, Password: "password"},
			token: usertoken,
			err:   nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.UpdateUser(tc.user, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestUpdatePassword(t *testing.T) {
	svc := newUserService()
	ts := newUserServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	user := sdk.User{Email: "user@example.com", Password: "password"}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: "authorities", Relation: "member"})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: mfauth.APIKey})
	token := tkn.GetValue()
	_, err := mainfluxSDK.CreateUser(token, user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	usertoken, err := mainfluxSDK.CreateToken(user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		oldPass string
		newPass string
		token   string
		err     errors.SDKError
	}{
		{
			desc:    "update password for user",
			oldPass: "password",
			newPass: "password123",
			token:   usertoken,
			err:     nil,
		},
		{
			desc:    "update password for user with invalid token",
			oldPass: "password",
			newPass: "password123",
			token:   wrongValue,
			err:     errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "update password for user with empty token",
			oldPass: "password",
			newPass: "password123",
			token:   "",
			err:     errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.UpdatePassword(tc.oldPass, tc.newPass, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
