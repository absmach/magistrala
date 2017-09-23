package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.ChannelRepository = (*channelRepository)(nil)

type channelRepository struct {
	session *gocql.Session
}

// NewChannelRepository instantiates Cassandra channel repository.
func NewChannelRepository(session *gocql.Session) manager.ChannelRepository {
	return &channelRepository{session}
}

func (repo *channelRepository) Save(channel manager.Channel) (string, error) {
	cql := `INSERT INTO channels_by_user (user, id, name, connected)
		VALUES (?, ?, ?, ?)`
	id := gocql.TimeUUID()

	if err := repo.session.Query(cql, channel.Owner, id,
		channel.Name, channel.Connected).Exec(); err != nil {
		return "", err
	}

	return id.String(), nil
}

func (repo *channelRepository) Update(channel manager.Channel) error {
	cql := `UPDATE channels_by_user SET name = ?, connected = ?
		WHERE user = ? AND id = ? IF EXISTS`

	if applied, _ := repo.session.Query(cql, channel.Name, channel.Connected,
		channel.Owner, channel.ID).ScanCAS(); !applied {
		return manager.ErrNotFound
	}

	return nil
}

func (repo *channelRepository) One(owner, id string) (manager.Channel, error) {
	cql := `SELECT name, connected FROM channels_by_user
		WHERE user = ? AND id = ? LIMIT 1`

	ch := manager.Channel{
		Owner: owner,
		ID:    id,
	}

	if err := repo.session.Query(cql, owner, id).Scan(&ch.Name, &ch.Connected); err != nil {
		return ch, manager.ErrNotFound
	}

	return ch, nil
}

func (repo *channelRepository) All(owner string) []manager.Channel {
	cql := `SELECT id, name, connected FROM channels_by_user WHERE user = ?`
	var id string
	var name string
	var connected []string

	// NOTE: the closing might failed
	iter := repo.session.Query(cql, owner).Iter()
	defer iter.Close()

	channels := make([]manager.Channel, 0)

	for iter.Scan(&id, &name, &connected) {
		c := manager.Channel{
			Owner:     owner,
			ID:        id,
			Name:      name,
			Connected: replaceNilWithEmpty(connected),
		}

		channels = append(channels, c)
	}

	return channels
}

func replaceNilWithEmpty(items []string) []string {
	if items != nil {
		return items
	}

	return make([]string, 0)
}

func (repo *channelRepository) Remove(owner, id string) error {
	cql := `DELETE FROM channels_by_user WHERE user = ? AND id = ?`
	return repo.session.Query(cql, owner, id).Exec()
}

func (repo *channelRepository) HasClient(channel, client string) bool {
	cql := `SELECT connected FROM clients_by_channel WHERE id = ? LIMIT 1`

	var connected []string

	if err := repo.session.Query(cql, channel).Scan(&connected); err != nil {
		return false
	}

	for _, v := range connected {
		if v == client {
			return true
		}
	}

	return false
}
