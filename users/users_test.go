// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"
	"github.com/stretchr/testify/assert"
)

const (
	email    = "user@example.com"
	password = "password"

	maxLocalLen  = 64
	maxDomainLen = 255
	maxTLDLen    = 24
)

var letters = "abcdefghijklmnopqrstuvwxyz"

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestValidate(t *testing.T) {
	cases := map[string]struct {
		user users.User
		err  error
	}{
		"validate user with valid data": {
			user: users.User{
				Email:    email,
				Password: password,
			},
			err: nil,
		},
		"validate user with valid domain and subdomain": {
			user: users.User{
				Email:    "user@example.sub.domain.com",
				Password: password,
			},
			err: nil,
		},
		"validate user with invalid subdomain": {
			user: users.User{
				Email:    "user@example..domain.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with invalid domain": {
			user: users.User{
				Email:    "user@.sub.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with empty email": {
			user: users.User{
				Email:    "",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with invalid email": {
			user: users.User{
				Email:    "userexample.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with utf8 email (cyrillic)": {
			user: users.User{
				Email:    "почта@кино-россия.рф",
				Password: password,
			},
			err: nil,
		},
		"validate user with utf8 email (hieroglyph)": {
			user: users.User{
				Email:    "艾付忧西开@艾付忧西开.再得",
				Password: password,
			},
			err: nil,
		},
		"validate user with no email tld": {
			user: users.User{
				Email:    "user@example.",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with too long email tld": {
			user: users.User{
				Email:    "user@example." + randomString(maxTLDLen+1),
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with no email domain": {
			user: users.User{
				Email:    "user@.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with too long email domain": {
			user: users.User{
				Email:    "user@" + randomString(maxDomainLen+1) + ".com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with no email local": {
			user: users.User{
				Email:    "@example.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
		"validate user with too long email local": {
			user: users.User{
				Email:    randomString(maxLocalLen+1) + "@example.com",
				Password: password,
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		err := tc.user.Validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
