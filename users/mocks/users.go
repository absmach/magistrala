// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
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

func (urm *userRepositoryMock) RetrieveAll(ctx context.Context, ids []string, pm users.PageMetadata) (users.UserPage, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	up := users.UserPage{}
	i := uint64(0)

	if pm.Email != "" {
		val, ok := urm.users[pm.Email]
		if !ok {
			return users.UserPage{}, errors.ErrNotFound
		}
		up.Offset = pm.Offset
		up.Limit = pm.Limit
		up.Total = uint64(i)
		up.Users = []users.User{val}
		return up, nil
	}

	if pm.Status == users.EnabledStatusKey || pm.Status == users.DisabledStatusKey {
		for _, u := range sortUsers(urm.users) {
			if i >= pm.Offset && i < (pm.Limit+pm.Offset) {
				if pm.Status == u.Status {
					up.Users = append(up.Users, u)
				}
			}
			i++
		}
		up.Offset = pm.Offset
		up.Limit = pm.Limit
		up.Total = uint64(i)
		return up, nil
	}
	for _, u := range sortUsers(urm.users) {
		if i >= pm.Offset && i < (pm.Limit+pm.Offset) {
			up.Users = append(up.Users, u)
		}
		i++
	}

	up.Offset = pm.Offset
	up.Limit = pm.Limit
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

func (urm *userRepositoryMock) ChangeStatus(ctx context.Context, id, status string) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	user, ok := urm.usersByID[id]
	if !ok {
		return errors.ErrNotFound
	}
	user.Status = status
	urm.usersByID[id] = user
	urm.users[user.Email] = user
	return nil
}
func sortUsers(us map[string]users.User) []users.User {
	users := []users.User{}
	ids := make([]string, 0, len(us))
	for k := range us {
		ids = append(ids, k)
	}

	sort.Strings(ids)
	for _, id := range ids {
		users = append(users, us[id])
	}

	return users
}
