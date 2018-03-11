package postgres

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mainflux/mainflux/manager"
	uuid "github.com/satori/go.uuid"
)

var _ manager.ChannelRepository = (*channelRepository)(nil)

type channelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) manager.ChannelRepository {
	return &channelRepository{db}
}

func (cr channelRepository) Save(channel manager.Channel) (string, error) {
	channel.ID = uuid.NewV4().String()

	if err := cr.db.Create(&channel).Error; err != nil {
		return "", err
	}

	return channel.ID, nil
}

func (cr channelRepository) Update(channel manager.Channel) error {
	sql := "UPDATE channels SET name = ? WHERE owner = ? AND id = ?;"
	res := cr.db.Exec(sql, channel.Name, channel.Owner, channel.ID)

	if res.Error == nil && res.RowsAffected == 0 {
		return manager.ErrNotFound
	}

	return res.Error
}

func (cr channelRepository) One(owner, id string) (manager.Channel, error) {
	channel := manager.Channel{}

	res := cr.db.Preload("Clients").First(&channel, "owner = ? AND id = ?", owner, id)

	if err := res.Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return channel, manager.ErrNotFound
		}

		return channel, err
	}

	return channel, nil
}

func (cr channelRepository) All(owner string) []manager.Channel {
	var channels []manager.Channel

	cr.db.Find(&channels, "owner = ?", owner)

	return channels
}

func (cr channelRepository) Remove(owner, id string) error {
	cr.db.Delete(&manager.Channel{}, "owner = ? AND id = ?", owner, id)
	return nil
}

func (cr channelRepository) Connect(owner, chanId, clientId string) error {
	// This approach can be replaced by declaring composite keys on both tables
	// (clients and channels), and then propagate them into the m2m table. For
	// some reason GORM does not infer these kind of connections well and
	// raises a "no unique constraint for referenced table". Until we find a
	// way to properly represent this relationship, let's stick with the nested
	// query approach and observe its behaviour.
	sql := `INSERT INTO channel_clients (channel_id, client_id)
	SELECT ?, ? WHERE
	EXISTS (SELECT 1 FROM channels WHERE owner = ? AND id = ?) AND
	EXISTS (SELECT 1 FROM clients WHERE owner = ? AND id = ?);`

	res := cr.db.Exec(sql, chanId, clientId, owner, chanId, owner, clientId)

	if res.Error == nil && res.RowsAffected == 0 {
		return manager.ErrNotFound
	}

	return res.Error
}

func (cr channelRepository) Disconnect(owner, chanId, clientId string) error {
	// The same remark given in Connect applies here.
	sql := `DELETE FROM channel_clients WHERE
	channel_id = ? AND client_id = ? AND
	EXISTS (SELECT 1 FROM channels WHERE owner = ? AND id = ?) AND
	EXISTS (SELECT 1 FROM clients WHERE owner = ? AND id = ?);`

	res := cr.db.Exec(sql, chanId, clientId, owner, chanId, owner, clientId)

	if res.Error == nil && res.RowsAffected == 0 {
		return manager.ErrNotFound
	}

	return res.Error
}

func (cr channelRepository) HasClient(chanId, clientId string) bool {
	sql := "SELECT EXISTS (SELECT 1 FROM channel_clients WHERE channel_id = $1 AND client_id = $2);"

	row := cr.db.DB().QueryRow(sql, chanId, clientId)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		// TODO: this error should be logged
		return false
	}

	return exists
}
