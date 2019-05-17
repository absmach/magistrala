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

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/users"
	"github.com/stretchr/testify/assert"
)

const (
	email = "john.doe@example.com"
	token = "token"
)

func TestIdentify(t *testing.T) {
	svc := users.NewSingleUserService(email, token)

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify non-existing user": {
			token: "non-existing",
			id:    "",
			err:   things.ErrUnauthorizedAccess,
		},
		"identify existing user": {
			token: token,
			id:    email,
			err:   nil,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s, got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s, got %s", desc, tc.err, err))
	}
}
