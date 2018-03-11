package mocks

import (
	"sync"

	"github.com/mainflux/mainflux/manager"
)

var _ manager.UserRepository = (*userRepositoryMock)(nil)

type userRepositoryMock struct {
	mu    sync.Mutex
	users map[string]manager.User
}

// NewUserRepository creates in-memory user repository.
func NewUserRepository() manager.UserRepository {
	return &userRepositoryMock{
		users: make(map[string]manager.User),
	}
}

func (urm *userRepositoryMock) Save(user manager.User) error {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if _, ok := urm.users[user.Email]; ok {
		return manager.ErrConflict
	}

	urm.users[user.Email] = user
	return nil
}

func (urm *userRepositoryMock) One(email string) (manager.User, error) {
	urm.mu.Lock()
	defer urm.mu.Unlock()

	if val, ok := urm.users[email]; ok {
		return val, nil
	}

	return manager.User{}, manager.ErrUnauthorizedAccess
}
