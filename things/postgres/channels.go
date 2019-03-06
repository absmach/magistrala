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
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return empty, things.ErrNotFound
		}
		return empty, err
	}

	return channel, nil
}

func (cr channelRepository) RetrieveAll(owner string, offset, limit uint64) things.ChannelsPage {
	q := `SELECT id, name, metadata FROM channels WHERE owner = $1 ORDER BY id LIMIT $2 OFFSET $3`
	items := []things.Channel{}

	rows, err := cr.db.Query(q, owner, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve channels due to %s", err))
		return things.ChannelsPage{}
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Channel{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channel due to %s", err))
			return things.ChannelsPage{}
		}
		items = append(items, c)
	}

	q = `SELECT COUNT(*) FROM channels WHERE owner = $1`

	var total uint64
	if err := cr.db.QueryRow(q, owner).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count channels due to %s", err))
		return things.ChannelsPage{}
	}

	page := things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page
}

func (cr channelRepository) RetrieveByThing(owner, thing string, offset, limit uint64) things.ChannelsPage {
	q := `SELECT id, name, metadata
	      FROM channels ch
	      INNER JOIN connections co
		  ON ch.id = co.channel_id
		  WHERE ch.owner = $1 AND co.thing_id = $2
		  ORDER BY ch.id
		  LIMIT $3
		  OFFSET $4`
	items := []things.Channel{}

	rows, err := cr.db.Query(q, owner, thing, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve channels due to %s", err))
		return things.ChannelsPage{}
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Channel{Owner: owner}
		if err := rows.Scan(&c.ID, &c.Name, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channel due to %s", err))
			return things.ChannelsPage{}
		}
		items = append(items, c)
	}

	q = `SELECT COUNT(*)
	     FROM channels ch
	     INNER JOIN connections co
	     ON ch.id = co.channel_id
	     WHERE ch.owner = $1 AND co.thing_id = $2`

	var total uint64
	if err := cr.db.QueryRow(q, owner, thing).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count channels due to %s", err))
		return things.ChannelsPage{}
	}

	return things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}
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
