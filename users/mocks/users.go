//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/users"
)

var _ users.UserRepository = (*userRepositoryMock)(nil)

type userRepositoryMock struct {
	mu    sync.Mutex
	users map[string]users.User
}

// NewUserRepository creates in-memory user repository.
func NewUserRepository() users.UserRepository {
	return &userRepositoryMock{
		users: make(map[string]users.User),
	}
}

func (urm *userRepositoryMock) Save(ctx context.Context, user users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[user.Email]; ok {
		return users.ErrConflict
	}

	urm.users[user.Email] = user
	return nil
}

func (urm *userRepositoryMock) RetrieveByID(ctx context.Context, email string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := urm.users[email]
	if !ok {
		return users.User{}, users.ErrNotFound
	}

	return val, nil
}
