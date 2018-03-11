package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/postgres"
	"github.com/stretchr/testify/assert"
)

func TestUserSave(t *testing.T) {
	email := "user-save@example.com"

	cases := map[string]struct {
		user manager.User
		err  error
	}{
		"new user":       {manager.User{email, "pass"}, nil},
		"duplicate user": {manager.User{email, "pass"}, manager.ErrConflict},
	}

	repo := postgres.NewUserRepository(db)

	for desc, tc := range cases {
		err := repo.Save(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleUserRetrieval(t *testing.T) {
	email := "user-retrieval@example.com"

	repo := postgres.NewUserRepository(db)
	repo.Save(manager.User{email, "pass"})

	cases := map[string]struct {
		email string
		err   error
	}{
		"existing user":     {email, nil},
		"non-existing user": {"unknown@example.com", manager.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := repo.One(tc.email)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
