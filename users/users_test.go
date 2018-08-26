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
	"github.com/stretchr/testify/assert"
)

const (
	email    = "user@example.com"
	password = "password"
)

func TestValidate(t *testing.T) {
	cases := map[string]struct {
		user users.User
		err  error
	}{
		"validate user with valid data":     {users.User{email, password}, nil},
		"validate user with empty email":    {users.User{"", password}, users.ErrMalformedEntity},
		"validate user with empty password": {users.User{email, ""}, users.ErrMalformedEntity},
		"validate user with invalid email":  {users.User{"userexample.com", password}, users.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		err := tc.user.Validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s", desc, tc.err, err))
	}
}
