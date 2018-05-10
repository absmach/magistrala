package postgres

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // required by GORM
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/users"
)

const errDuplicate string = "unique_violation"

var _ users.UserRepository = (*userRepository)(nil)

type userRepository struct {
	db *gorm.DB
}

// New instantiates a PostgreSQL implementation of user
// repository.
func New(db *gorm.DB) users.UserRepository {
	return &userRepository{db}
}

func (ur *userRepository) Save(user users.User) error {
	if err := ur.db.Create(&user).Error; err != nil {
		if pqErr, ok := err.(*pq.Error); ok && errDuplicate == pqErr.Code.Name() {
			return users.ErrConflict
		}

		return err
	}

	return nil
}

func (ur *userRepository) One(email string) (users.User, error) {
	user := users.User{}

	q := ur.db.First(&user, "email = ?", email)

	if err := q.Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, users.ErrNotFound
		}

		return user, err
	}

	return user, nil
}
