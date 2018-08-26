//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
)

func TestUserSave(t *testing.T) {
	email := "user-save@example.com"

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{"new user", users.User{email, "pass"}, nil},
		{"duplicate user", users.User{email, "pass"}, users.ErrConflict},
	}

	repo := postgres.New(db)

	for _, tc := range cases {
		err := repo.Save(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleUserRetrieval(t *testing.T) {
	email := "user-retrieval@example.com"

	repo := postgres.New(db)
	repo.Save(users.User{email, "pass"})

	cases := map[string]struct {
		email string
		err   error
	}{
		"existing user":     {email, nil},
		"non-existing user": {"unknown@example.com", users.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := repo.RetrieveByID(tc.email)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
