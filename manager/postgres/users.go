package postgres

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.UserRepository = (*userRepository)(nil)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository instantiates a PostgreSQL implementation of user
// repository.
func NewUserRepository(db *gorm.DB) manager.UserRepository {
	return &userRepository{db}
}

func (ur *userRepository) Save(user manager.User) error {
	if err := ur.db.Create(&user).Error; err != nil {
		if pqErr, ok := err.(*pq.Error); ok && errDuplicate == pqErr.Code.Name() {
			return manager.ErrConflict
		}

		return err
	}

	return nil
}

func (ur *userRepository) One(email string) (manager.User, error) {
	user := manager.User{}

	q := ur.db.First(&user, "email = ?", email)

	if err := q.Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, manager.ErrNotFound
		}

		return user, err
	}

	return user, nil
}
