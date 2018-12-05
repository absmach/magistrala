//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
)

var _ things.ChannelRepository = (*channelRepository)(nil)

const (
	errDuplicate = "unique_violation"
	errFK        = "foreign_key_violation"
	errInvalid   = "invalid_text_representation"
)

type channelRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewChannelRepository(db *sql.DB, log logger.Logger) things.ChannelRepository {
	return &channelRepository{
		db:  db,
		log: log,
	}
}

func (cr channelRepository) Save(channel things.Channel) (string, error) {
	q := `INSERT INTO channels (id, owner, name, metadata) VALUES ($1, $2, $3, $4)`

	metadata := channel.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	_, err := cr.db.Exec(q, channel.ID, channel.Owner, channel.Name, metadata)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && errInvalid == pqErr.Code.Name() {
			return "", things.ErrMalformedEntity
		}

		return "", err
	}

	return channel.ID, nil
}

func (cr channelRepository) Update(channel things.Channel) error {
	q := `UPDATE channels SET name = $1, metadata = $2 WHERE owner = $3 AND id = $4;`

	metadata := channel.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	res, err := cr.db.Exec(q, channel.Name, metadata, channel.Owner, channel.ID)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && errInvalid == pqErr.Code.Name() {
			return things.ErrMalformedEntity
		}

		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (cr channelRepository) RetrieveByID(owner, id string) (things.Channel, error) {
	q := `SELECT name, metadata FROM channels WHERE id = $1 AND owner = $2`
	channel := things.Channel{ID: id, Owner: owner}
	if err := cr.db.QueryRow(q, id, owner).Scan(&channel.Name, &channel.Metadata); err != nil {
		empty := things.Channel{}
		if err == sql.ErrNoRows {
			return empty, things.ErrNotFound
		}
		return empty, err
	}

	q = `SELECT id, name, type, key, metadata FROM things t
	INNER JOIN connections conn
	ON t.id = conn.thing_id AND t.owner = conn.thing_owner
	WHERE conn.channel_id = $1 AND conn.channel_owner = $2`

	rows, err := cr.db.Query(q, id, owner)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return things.Channel{}, err
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Thing{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Key, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return things.Channel{}, err
		}
		channel.Things = append(channel.Things, c)
	}

	return channel, nil
}

func (cr channelRepository) RetrieveAll(owner string, offset, limit uint64) []things.Channel {
	q := `SELECT id, name, metadata FROM channels WHERE owner = $1 ORDER BY id LIMIT $2 OFFSET $3`
	items := []things.Channel{}

	rows, err := cr.db.Query(q, owner, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve channels due to %s", err))
		return []things.Channel{}
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Channel{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channel due to %s", err))
			return []things.Channel{}
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

func (cr channelRepository) Connect(owner, chanID, thingID string) error {
	q := `INSERT INTO connections (channel_id, channel_owner, thing_id, thing_owner) VALUES ($1, $2, $3, $2)`

	if _, err := cr.db.Exec(q, chanID, owner, thingID); err != nil {
		pqErr, ok := err.(*pq.Error)

		if ok && errFK == pqErr.Code.Name() {
			return things.ErrNotFound
		}

		// connect is idempotent
		if ok && errDuplicate == pqErr.Code.Name() {
			return nil
		}

		return err
	}

	return nil
}

func (cr channelRepository) Disconnect(owner, chanID, thingID string) error {
	q := `DELETE FROM connections
	WHERE channel_id = $1 AND channel_owner = $2
	AND thing_id = $3 AND thing_owner = $2`

	res, err := cr.db.Exec(q, chanID, owner, thingID)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (cr channelRepository) HasThing(chanID, key string) (string, error) {
	var thingID string

	q := `SELECT id FROM things WHERE key = $1`
	if err := cr.db.QueryRow(q, key).Scan(&thingID); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to obtain thing's ID due to %s", err))
		return "", err
	}

	q = `SELECT EXISTS (SELECT 1 FROM connections WHERE channel_id = $1 AND thing_id = $2);`
	exists := false
	if err := cr.db.QueryRow(q, chanID, thingID).Scan(&exists); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to check thing existence due to %s", err))
		return "", err
	}

	if !exists {
		return "", things.ErrUnauthorizedAccess
	}

	return thingID, nil
}
