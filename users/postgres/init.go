package postgres

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // required by GORM
	"github.com/mainflux/mainflux/users"
)

// Connect creates a connection to the PostgreSQL instance. A non-nil error
// is returned to indicate failure.
func Connect(host, port, name, user, pass string) (*gorm.DB, error) {
	t := "host=%s port=%s user=%s dbname=%s password=%s sslmode=disable"
	url := fmt.Sprintf(t, host, port, user, name, pass)

	db, err := gorm.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	db = db.AutoMigrate(&users.User{})

	return db.LogMode(false), nil
}
