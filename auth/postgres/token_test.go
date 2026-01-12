// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/supermq/auth/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tokenIDProvider = uuid.New()

func generateTokenID(t *testing.T) string {
	id, err := tokenIDProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	return id
}

func TestTokenSave(t *testing.T) {
	repo := postgres.NewTokensRepository(database)

	tokenID := generateTokenID(t)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "save a new token",
			id:   tokenID,
			err:  nil,
		},
		{
			desc: "save with duplicate id",
			id:   tokenID,
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "save with empty id",
			id:   "",
			err:  repoerr.ErrCreateEntity,
		},
		{
			desc: "save with another valid id",
			id:   generateTokenID(t),
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.Save(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestTokenContains(t *testing.T) {
	repo := postgres.NewTokensRepository(database)

	tokenID := generateTokenID(t)
	err := repo.Save(context.Background(), tokenID)
	assert.Nil(t, err, fmt.Sprintf("Storing Token expected to succeed: %s", err))

	cases := []struct {
		desc     string
		id       string
		expected bool
	}{
		{
			desc:     "check for existing token",
			id:       tokenID,
			expected: true,
		},
		{
			desc:     "check for non-existent token",
			id:       generateTokenID(t),
			expected: false,
		},
		{
			desc:     "check with empty id",
			id:       "",
			expected: false,
		},
		{
			desc:     "check with another non-existent id",
			id:       generateTokenID(t),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := repo.Contains(context.Background(), tc.id)
			assert.Equal(t, tc.expected, result, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.expected, result))
		})
	}
}
