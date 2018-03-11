package postgres

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mainflux/mainflux/manager"
)

const errDuplicate string = "unique_violation"

type connection struct {
	ClientID  string `gorm:"primary_key"`
	ChannelID string `gorm:"primary_key"`
}

func (c connection) TableName() string {
	return "channel_clients"
}

// Connect creates a connection to the PostgreSQL instance. A non-nil error
// is returned to indicate failure.
func Connect(host, port, name, user, pass string) (*gorm.DB, error) {
	t := "host=%s port=%s user=%s dbname=%s password=%s sslmode=disable"
	url := fmt.Sprintf(t, host, port, user, name, pass)

	db, err := gorm.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	db = db.AutoMigrate(&manager.User{}, &manager.Client{}, &manager.Channel{}, &connection{})

	db = db.Model(&manager.Client{}).
		AddForeignKey("owner", "users(email)", "CASCADE", "CASCADE").
		AddUniqueIndex("idx_pk_client", "id", "owner")

	db = db.Model(&manager.Channel{}).
		AddForeignKey("owner", "users(email)", "CASCADE", "CASCADE").
		AddUniqueIndex("idx_pk_channel", "id", "owner")

	db = db.Model(&connection{}).
		AddForeignKey("client_id", "clients(id)", "CASCADE", "CASCADE").
		AddForeignKey("channel_id", "channels(id)", "CASCADE", "CASCADE")

	return db.LogMode(false), nil
}
