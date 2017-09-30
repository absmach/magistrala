package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.UserRepository = (*userRepository)(nil)

type userRepository struct {
	session *gocql.Session
}

// NewUserRepository instantiates Cassandra user repository.
func NewUserRepository(session *gocql.Session) manager.UserRepository {
	return &userRepository{session}
}

func (repo *userRepository) Save(user manager.User) error {
	cql := `INSERT INTO users (email, password) VALUES (?, ?) IF NOT EXISTS`

	applied, err := repo.session.Query(cql, user.Email, user.Password).ScanCAS()
	if !applied {
		return manager.ErrConflict
	}

	return err
}

func (repo *userRepository) One(email string) (manager.User, error) {
	cql := `SELECT email, password FROM users WHERE email = ? LIMIT 1`

	user := manager.User{}

	if err := repo.session.Query(cql, email).
		Scan(&user.Email, &user.Password); err != nil {
		return user, manager.ErrUnauthorizedAccess
	}

	return user, nil
}
