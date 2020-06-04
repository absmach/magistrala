// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"

	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wrong string = "wrong-value"

var (
	user            = users.User{Email: "user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	nonExistingUser = users.User{Email: "non-ex-user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	host            = "example.com"
)

func newService() users.Service {
	repo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	e := mocks.NewEmailer()

	return users.New(repo, hasher, auth, e)
}

func TestRegister(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc string
		user users.User
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
			err:  users.ErrConflict,
		},
		{
			desc: "register new user with empty password",
			user: users.User{
				Email:    user.Email,
				Password: "",
			},
			err: users.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := svc.Register(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	noAuthUser := users.User{
		Email:    "email@test.com",
		Password: "pwd",
	}
	svc.Register(context.Background(), user)
	svc.Register(context.Background(), noAuthUser)

	cases := map[string]struct {
		user users.User
		err  error
	}{
		"login with good credentials": {
			user: user,
			err:  nil,
		},
		"login with wrong e-mail": {
			user: users.User{
				Email:    wrong,
				Password: user.Password,
			},
			err: users.ErrUnauthorizedAccess,
		},
		"login with wrong password": {
			user: users.User{
				Email:    user.Email,
				Password: wrong,
			},
			err: users.ErrUnauthorizedAccess,
		},
		"login failed auth": {
			user: noAuthUser,
			err:  users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewUser(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	u := user
	u.Password = ""

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"valid token's user info": {
			user:  u,
			token: token,
			err:   nil,
		},
		"invalid token's user info": {
			user:  users.User{},
			token: "",
			err:   users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewUser(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.Metadata = map[string]interface{}{"role": "test"}

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"update user with valid token": {
			user:  user,
			token: token,
			err:   nil,
		},
		"update user with invalid token": {
			user:  user,
			token: "non-existent",
			err:   users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		err := svc.UpdateUser(context.Background(), tc.token, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)

	cases := map[string]struct {
		email string
		err   error
	}{
		"valid user reset token":  {user.Email, nil},
		"invalid user rest token": {nonExistingUser.Email, users.ErrUserNotFound},
	}

	for desc, tc := range cases {
		err := svc.GenerateResetToken(context.Background(), tc.email, host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChangePassword(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token       string
		password    string
		oldPassword string
		err         error
	}{
		"valid user change password ":                    {token, "newpassword", user.Password, nil},
		"valid user change password with wrong password": {token, "newpassword", "wrongpassword", users.ErrUnauthorizedAccess},
		"valid user change password invalid token":       {"", "newpassword", user.Password, users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		err := svc.ChangePassword(context.Background(), tc.token, tc.password, tc.oldPassword)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestResetPassword(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	resetToken, err := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 2})
	assert.Nil(t, err, fmt.Sprintf("Generating reset token expected to succeed: %s", err))
	cases := map[string]struct {
		token    string
		password string
		err      error
	}{
		"valid user reset password ":   {resetToken.GetValue(), user.Email, nil},
		"invalid user reset password ": {"", "newpassword", users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		err := svc.ResetPassword(context.Background(), tc.token, tc.password)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSendPasswordReset(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token string
		email string
		err   error
	}{
		"valid user reset password ": {token, user.Email, nil},
	}

	for desc, tc := range cases {
		err := svc.SendPasswordReset(context.Background(), host, tc.email, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}
