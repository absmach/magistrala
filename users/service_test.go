//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package users_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
)

const wrong string = "wrong-value"

var user = users.User{"user@example.com", "password"}

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
		{"register new user", user, nil},
		{"register existing user", user, users.ErrConflict},
		{"register new user with empty password", users.User{user.Email, ""}, users.ErrMalformedEntity},
	}

	for _, tc := range cases {
		err := svc.Register(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	svc.Register(user)

	cases := map[string]struct {
		user users.User
		err  error
	}{
		"login with good credentials": {user, nil},
		"login with wrong e-mail":     {users.User{wrong, user.Password}, users.ErrUnauthorizedAccess},
		"login with wrong password":   {users.User{user.Email, wrong}, users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.Login(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

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
