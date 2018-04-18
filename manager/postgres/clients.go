package postgres

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // required by GORM
	"github.com/mainflux/mainflux/manager"
	uuid "github.com/satori/go.uuid"
)

var _ manager.ClientRepository = (*clientRepository)(nil)

type clientRepository struct {
	db *gorm.DB
}

// NewClientRepository instantiates a PostgreSQL implementation of client
// repository.
func NewClientRepository(db *gorm.DB) manager.ClientRepository {
	return &clientRepository{db}
}

func (cr *clientRepository) Id() string {
	return uuid.NewV4().String()
}

func (cr *clientRepository) Save(client manager.Client) error {
	return cr.db.Create(&client).Error
}

func (cr *clientRepository) Update(client manager.Client) error {
	sql := "UPDATE clients SET name = ?, payload = ? WHERE owner = ? AND id = ?;"
	res := cr.db.Exec(sql, client.Name, client.Payload, client.Owner, client.ID)

	if res.Error == nil && res.RowsAffected == 0 {
		return manager.ErrNotFound
	}

	return res.Error
}

func (cr *clientRepository) One(owner, id string) (manager.Client, error) {
	client := manager.Client{}

	res := cr.db.First(&client, "owner = ? AND id = ?", owner, id)

	if err := res.Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return client, manager.ErrNotFound
		}

		return client, err
	}

	return client, nil
}

func (cr *clientRepository) All(owner string, offset, limit int) []manager.Client {
	var clients []manager.Client

	cr.db.Offset(offset).Limit(limit).Find(&clients, "owner = ?", owner)

	return clients
}

func (cr *clientRepository) Remove(owner, id string) error {
	cr.db.Delete(&manager.Client{}, "owner = ? AND id = ?", owner, id)
	return nil
}
