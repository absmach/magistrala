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

func (ur *userRepositoryMock) Save(user manager.User) error {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, ok := ur.users[user.Email]; ok {
		return manager.ErrConflict
	}

	ur.users[user.Email] = user
	return nil
}

func (ur *userRepositoryMock) One(email string) (manager.User, error) {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	if val, ok := ur.users[email]; ok {
		return val, nil
	}

	return manager.User{}, manager.ErrUnauthorizedAccess
}
