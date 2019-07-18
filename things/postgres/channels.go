//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/things"
)

var _ things.ChannelRepository = (*channelRepository)(nil)

type channelRepository struct {
	db *sqlx.DB
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewChannelRepository(db *sqlx.DB) things.ChannelRepository {
	return &channelRepository{
		db: db,
	}
}

func (cr channelRepository) Save(_ context.Context, channel things.Channel) (string, error) {
	q := `INSERT INTO channels (id, owner, name, metadata)
        VALUES (:id, :owner, :name, :metadata);`

	dbch, err := toDBChannel(channel)
	if err != nil {
		return "", err
	}

	if _, err := cr.db.NamedExec(q, dbch); err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return "", things.ErrMalformedEntity
			}
		}

		return "", err
	}

	return channel.ID, nil
}

func (cr channelRepository) Update(_ context.Context, channel things.Channel) error {
	q := `UPDATE channels SET name = :name, metadata = :metadata WHERE owner = :owner AND id = :id;`

	dbch, err := toDBChannel(channel)
	if err != nil {
		return err
	}

	res, err := cr.db.NamedExec(q, dbch)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return things.ErrMalformedEntity
			}
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

func (cr channelRepository) RetrieveByID(_ context.Context, owner, id string) (things.Channel, error) {
	q := `SELECT name, metadata FROM channels WHERE id = $1 AND owner = $2;`
	dbch := dbChannel{
		ID:    id,
		Owner: owner,
	}
	if err := cr.db.QueryRowx(q, id, owner).StructScan(&dbch); err != nil {
		empty := things.Channel{}
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return empty, things.ErrNotFound
		}
		return empty, err
	}

	return toChannel(dbch)
}

func (cr channelRepository) RetrieveAll(_ context.Context, owner string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	name = strings.ToLower(name)
	nq := ""
	if name != "" {
		name = fmt.Sprintf(`%%%s%%`, name)
		nq = `AND LOWER(name) LIKE :name`
	}

	q := fmt.Sprintf(`SELECT id, name, metadata FROM channels
	      WHERE owner = :owner %s ORDER BY id LIMIT :limit OFFSET :offset;`, nq)

	params := map[string]interface{}{
		"owner":  owner,
		"limit":  limit,
		"offset": offset,
		"name":   name,
	}
	rows, err := cr.db.NamedQuery(q, params)
	if err != nil {
		return things.ChannelsPage{}, err
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, err
		}
		ch, err := toChannel(dbch)
		if err != nil {
			return things.ChannelsPage{}, err
		}

		items = append(items, ch)
	}

	cq := ""
	if name != "" {
		cq = `AND LOWER(name) LIKE $2`
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM channels WHERE owner = $1 %s;`, cq)

	total := uint64(0)
	switch name {
	case "":
		if err := cr.db.Get(&total, q, owner); err != nil {
			return things.ChannelsPage{}, err
		}
	default:
		if err := cr.db.Get(&total, q, owner, name); err != nil {
			return things.ChannelsPage{}, err
		}
	}

	page := things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (cr channelRepository) RetrieveByThing(_ context.Context, owner, thing string, offset, limit uint64) (things.ChannelsPage, error) {
	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thing); err != nil {
		return things.ChannelsPage{}, things.ErrNotFound
	}

	q := `SELECT id, name, metadata
	      FROM channels ch
	      INNER JOIN connections co
		  ON ch.id = co.channel_id
		  WHERE ch.owner = :owner AND co.thing_id = :thing
		  ORDER BY ch.id
		  LIMIT :limit
		  OFFSET :offset`

	params := map[string]interface{}{
		"owner":  owner,
		"thing":  thing,
		"limit":  limit,
		"offset": offset,
	}

	rows, err := cr.db.NamedQuery(q, params)
	if err != nil {
		return things.ChannelsPage{}, err
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, err
		}

		ch, err := toChannel(dbch)
		if err != nil {
			return things.ChannelsPage{}, err
		}

		items = append(items, ch)
	}

	q = `SELECT COUNT(*)
	     FROM channels ch
	     INNER JOIN connections co
	     ON ch.id = co.channel_id
	     WHERE ch.owner = $1 AND co.thing_id = $2`

	var total uint64
	if err := cr.db.Get(&total, q, owner, thing); err != nil {
		return things.ChannelsPage{}, err
	}

	return things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}, nil
}

func (cr channelRepository) Remove(_ context.Context, owner, id string) error {
	dbch := dbChannel{
		ID:    id,
		Owner: owner,
	}
	q := `DELETE FROM channels WHERE id = :id AND owner = :owner`
	cr.db.NamedExec(q, dbch)
	return nil
}

func (cr channelRepository) Connect(_ context.Context, owner, chanID, thingID string) error {
	q := `INSERT INTO connections (channel_id, channel_owner, thing_id, thing_owner)
	      VALUES (:channel, :owner, :thing, :owner);`

	conn := dbConnection{
		Channel: chanID,
		Thing:   thingID,
		Owner:   owner,
	}

	if _, err := cr.db.NamedExec(q, conn); err != nil {
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

func (cr channelRepository) Disconnect(_ context.Context, owner, chanID, thingID string) error {
	q := `DELETE FROM connections
	      WHERE channel_id = :channel AND channel_owner = :owner
	      AND thing_id = :thing AND thing_owner = :owner`

	conn := dbConnection{
		Channel: chanID,
		Thing:   thingID,
		Owner:   owner,
	}

	res, err := cr.db.NamedExec(q, conn)
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

func (cr channelRepository) HasThing(_ context.Context, chanID, key string) (string, error) {
	var thingID string

	q := `SELECT id FROM things WHERE key = $1`
	if err := cr.db.QueryRow(q, key).Scan(&thingID); err != nil {
		return "", err

	}

	if err := cr.hasThing(chanID, thingID); err != nil {
		return "", err
	}

	return thingID, nil
}

func (cr channelRepository) HasThingByID(_ context.Context, chanID, thingID string) error {
	return cr.hasThing(chanID, thingID)
}

func (cr channelRepository) hasThing(chanID, thingID string) error {
	q := `SELECT EXISTS (SELECT 1 FROM connections WHERE channel_id = $1 AND thing_id = $2);`
	exists := false
	if err := cr.db.QueryRow(q, chanID, thingID).Scan(&exists); err != nil {
		return err
	}

	if !exists {
		return things.ErrUnauthorizedAccess
	}

	return nil
}

type dbChannel struct {
	ID       string `db:"id"`
	Owner    string `db:"owner"`
	Name     string `db:"name"`
	Metadata string `db:"metadata"`
}

func toDBChannel(ch things.Channel) (dbChannel, error) {
	data, err := json.Marshal(ch.Metadata)
	if err != nil {
		return dbChannel{}, err
	}

	return dbChannel{
		ID:       ch.ID,
		Owner:    ch.Owner,
		Name:     ch.Name,
		Metadata: string(data),
	}, nil
}

func toChannel(ch dbChannel) (things.Channel, error) {
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(ch.Metadata), &metadata); err != nil {
		return things.Channel{}, err
	}

	return things.Channel{
		ID:       ch.ID,
		Owner:    ch.Owner,
		Name:     ch.Name,
		Metadata: metadata,
	}, nil
}

type dbConnection struct {
	Channel string `db:"channel"`
	Thing   string `db:"thing"`
	Owner   string `db:"owner"`
}
