// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddUserVerification(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM users_verifications")
		require.Nil(t, err, fmt.Sprintf("clean users_verifications unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	username := namesgen.Generate()
	user := users.User{
		ID:        "test-user-id",
		Email:     "test@example.com",
		FirstName: first_name,
		LastName:  last_name,
		Credentials: users.Credentials{
			Username: username,
		},
	}
	_, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("saving user unexpected error: %s", err))

	cases := []struct {
		desc string
		uv   users.UserVerification
		err  error
	}{
		{
			desc: "add new user verification",
			uv: users.UserVerification{
				UserID:    user.ID,
				Email:     user.Email,
				CreatedAt: time.Now().UTC(),
				OTP:       "123456",
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			},
			err: nil,
		},
		{
			desc: "add user verification for non-existing user",
			uv: users.UserVerification{
				UserID:    "non-existing-user",
				Email:     "non-existing@example.com",
				OTP:       "654321",
				CreatedAt: time.Now().UTC(),
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.AddUserVerification(context.Background(), tc.uv)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRetrieveUserVerification(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM users_verifications")
		require.Nil(t, err, fmt.Sprintf("clean users_verifications unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	username := namesgen.Generate()
	user := users.User{
		ID:        "test-user-id",
		Email:     "test@example.com",
		FirstName: first_name,
		LastName:  last_name,
		Credentials: users.Credentials{
			Username: username,
		},
	}
	_, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("saving user unexpected error: %s", err))

	uv := users.UserVerification{
		UserID:    user.ID,
		Email:     user.Email,
		OTP:       "123456",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	err = repo.AddUserVerification(context.Background(), uv)
	require.Nil(t, err, fmt.Sprintf("adding user verification unexpected error: %s", err))

	cases := []struct {
		desc   string
		userID string
		email  string
		err    error
	}{
		{
			desc:   "retrieve existing user verification",
			userID: user.ID,
			email:  user.Email,
			err:    nil,
		},
		{
			desc:   "retrieve non-existing user verification",
			userID: "non-existing-user",
			email:  "non-existing@example.com",
			err:    repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		retrieved, err := repo.RetrieveUserVerification(context.Background(), tc.userID, tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, uv.UserID, retrieved.UserID, fmt.Sprintf("%s: expected %v got %v", tc.desc, uv.UserID, retrieved.UserID))
			assert.Equal(t, uv.Email, retrieved.Email, fmt.Sprintf("%s: expected %v got %v", tc.desc, uv.Email, retrieved.Email))
			assert.Equal(t, uv.OTP, retrieved.OTP, fmt.Sprintf("%s: expected %v got %v", tc.desc, uv.OTP, retrieved.OTP))
		}
	}
}

func TestUpdateUserVerification(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM users_verifications")
		require.Nil(t, err, fmt.Sprintf("clean users_verifications unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	username := namesgen.Generate()
	user := users.User{
		ID:        "test-user-id",
		Email:     "test@example.com",
		FirstName: first_name,
		LastName:  last_name,
		Credentials: users.Credentials{
			Username: username,
		},
	}
	_, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("saving user unexpected error: %s", err))

	uv := users.UserVerification{
		UserID:    user.ID,
		Email:     user.Email,
		OTP:       "123456",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	err = repo.AddUserVerification(context.Background(), uv)
	require.Nil(t, err, fmt.Sprintf("adding user verification unexpected error: %s", err))

	usedTime := time.Now()
	cases := []struct {
		desc string
		uv   users.UserVerification
		err  error
	}{
		{
			desc: "update existing user verification",
			uv: users.UserVerification{
				UserID: user.ID,
				Email:  user.Email,
				OTP:    "654321",
				UsedAt: usedTime,
			},
			err: nil,
		},
		{
			desc: "update non-existing user verification",
			uv: users.UserVerification{
				UserID: "non-existing-user",
				Email:  "non-existing@example.com",
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.UpdateUserVerification(context.Background(), tc.uv)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if err == nil {
			retrieved, err := repo.RetrieveUserVerification(context.Background(), tc.uv.UserID, tc.uv.Email)
			require.Nil(t, err, fmt.Sprintf("retrieving updated verification unexpected error: %s", err))
			assert.Equal(t, tc.uv.OTP, retrieved.OTP, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.uv.OTP, retrieved.OTP))
			assert.WithinDuration(t, tc.uv.UsedAt, retrieved.UsedAt, 10*time.Second, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.uv.UsedAt, retrieved.UsedAt))
		}
	}
}
