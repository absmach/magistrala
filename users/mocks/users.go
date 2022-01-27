// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"
)

var _ users.UserRepository = (*userRepositoryMock)(nil)

type userRepositoryMock struct {
	mu             sync.Mutex
	users          map[string]users.User
	usersByID      map[string]users.User
	usersByGroupID map[string]users.User
}

// NewUserRepository creates in-memory user repository
func NewUserRepository() users.UserRepository {
	return &userRepositoryMock{
		users:          make(map[string]users.User),
		usersByID:      make(map[string]users.User),
		usersByGroupID: make(map[string]users.User),
	}
}

func (urm *userRepositoryMock) Save(ctx context.Context, user users.User) (string, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[user.Email]; ok {
		return "", errors.ErrConflict
	}

	urm.users[user.Email] = user
	urm.usersByID[user.ID] = user
	return user.ID, nil
}

func (urm *userRepositoryMock) Update(ctx context.Context, user users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[user.Email]; !ok {
		return errors.ErrNotFound
	}

	urm.users[user.Email] = user
	return nil
}

func (urm *userRepositoryMock) UpdateUser(ctx context.Context, user users.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[user.Email]; !ok {
		return errors.ErrNotFound
	}

	urm.users[user.Email] = user
	return nil
}

func (urm *userRepositoryMock) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := urm.users[email]
	if !ok {
		return users.User{}, errors.ErrNotFound
	}

	return val, nil
}

func (urm *userRepositoryMock) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	val, ok := urm.usersByID[id]
	if !ok {
		return users.User{}, errors.ErrNotFound
	}

	return val, nil
}

func (urm *userRepositoryMock) RetrieveAll(ctx context.Context, offset, limit uint64, ids []string, email string, um users.Metadata) (users.UserPage, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	up := users.UserPage{}
	i := uint64(0)

	for _, u := range urm.users {
		if i >= offset && i < (limit+offset) {
			up.Users = append(up.Users, u)
		}
		i++
	}

	up.Offset = offset
	up.Limit = limit
	up.Total = uint64(i)

	return up, nil
}

func (urm *userRepositoryMock) UpdatePassword(_ context.Context, token, password string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[token]; !ok {
		return errors.ErrNotFound
	}
	return nil
}
