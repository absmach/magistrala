//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package users_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
)

const wrong string = "wrong-value"

var user = users.User{Email: "user@example.com", Password: "password"}

func newService() users.Service {
	repo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()

	return users.New(repo, hasher, idp)
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
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)

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
	}

	for desc, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()
	svc.Register(context.Background(), user)
	key, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		key string
		err error
	}{
		"valid token's identity":   {key, nil},
		"invalid token's identity": {"", users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.Identify(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
