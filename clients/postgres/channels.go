package postgres

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/logger"
	uuid "github.com/satori/go.uuid"
)

var _ clients.ChannelRepository = (*channelRepository)(nil)

const (
	errDuplicate = "unique_violation"
	errFK        = "foreign_key_violation"
)

type channelRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewChannelRepository(db *sql.DB, log logger.Logger) clients.ChannelRepository {
	return &channelRepository{db: db, log: log}
}

func (cr channelRepository) Save(channel clients.Channel) (string, error) {
	channel.ID = uuid.NewV4().String()

	q := `INSERT INTO channels (id, owner, name) VALUES ($1, $2, $3)`

	_, err := cr.db.Exec(q, channel.ID, channel.Owner, channel.Name)
	if err != nil {
		return "", err
	}

	return channel.ID, nil
}

func (cr channelRepository) Update(channel clients.Channel) error {
	q := `UPDATE channels SET name = $1 WHERE owner = $2 AND id = $3;`

	res, err := cr.db.Exec(q, channel.Name, channel.Owner, channel.ID)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return clients.ErrNotFound
	}

	return nil
}

func (cr channelRepository) One(owner, id string) (clients.Channel, error) {
	q := `SELECT name FROM channels WHERE id = $1 AND owner = $2`
	channel := clients.Channel{ID: id, Owner: owner}
	if err := cr.db.QueryRow(q, id, owner).Scan(&channel.Name); err != nil {
		empty := clients.Channel{}
		if err == sql.ErrNoRows {
			return empty, clients.ErrNotFound
		}
		return empty, err
	}

	qr := `SELECT id, type, name, key, payload FROM clients cli
	INNER JOIN connections conn
	ON cli.id = conn.client_id AND cli.owner = conn.client_owner
	WHERE conn.channel_id = $1 AND conn.channel_owner = $2`

	rows, err := cr.db.Query(qr, id, owner)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return clients.Channel{}, err
	}
	defer rows.Close()

	for rows.Next() {
		c := clients.Client{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Key, &c.Payload); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected client due to %s", err))
			return clients.Channel{}, err
		}
		channel.Clients = append(channel.Clients, c)
	}

	return channel, nil
}

func (cr channelRepository) All(owner string, offset, limit int) []clients.Channel {
	q := `SELECT id, name FROM channels WHERE owner = $1 LIMIT $2 OFFSET $3`
	items := []clients.Channel{}

	rows, err := cr.db.Query(q, owner, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve channels due to %s", err))
		return []clients.Channel{}
	}
	defer rows.Close()

	for rows.Next() {
		c := clients.Channel{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channel due to %s", err))
			return []clients.Channel{}
		}
		items = append(items, c)
	}

	return items
}

func (cr channelRepository) Remove(owner, id string) error {
	q := `DELETE FROM channels WHERE id = $1 AND owner = $2`
	cr.db.Exec(q, id, owner)
	return nil
}

func (cr channelRepository) Connect(owner, chanID, clientID string) error {
	q := `INSERT INTO connections (channel_id, channel_owner, client_id, client_owner) VALUES ($1, $2, $3, $2)`

	if _, err := cr.db.Exec(q, chanID, owner, clientID); err != nil {
		pqErr, ok := err.(*pq.Error)

		if ok && errFK == pqErr.Code.Name() {
			return clients.ErrNotFound
		}

		// connect is idempotent
		if ok && errDuplicate == pqErr.Code.Name() {
			return nil
		}

		return err
	}

	return nil
}

func (cr channelRepository) Disconnect(owner, chanID, clientID string) error {
	q := `DELETE FROM connections
	WHERE channel_id = $1 AND channel_owner = $2
	AND client_id = $3 AND client_owner = $2`

	res, err := cr.db.Exec(q, chanID, owner, clientID)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return clients.ErrNotFound
	}

	return nil
}

func (cr channelRepository) HasClient(chanID, clientID string) bool {
	q := "SELECT EXISTS (SELECT 1 FROM connections WHERE channel_id = $1 AND client_id = $2);"

	exists := false
	if err := cr.db.QueryRow(q, chanID, clientID).Scan(&exists); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to check client existence due to %s", err))
	}
	return exists
}
