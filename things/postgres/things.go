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

	"github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
)

var _ things.ThingRepository = (*thingRepository)(nil)

type thingRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewThingRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewThingRepository(db *sql.DB, log logger.Logger) things.ThingRepository {
	return &thingRepository{db: db, log: log}
}

func (tr thingRepository) Save(thing things.Thing) (string, error) {
	q := `INSERT INTO things (id, owner, type, name, key, metadata) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	metadata := thing.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	_, err := tr.db.Exec(q, thing.ID, thing.Owner, thing.Type, thing.Name, thing.Key, metadata)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && errInvalid == pqErr.Code.Name() {
			return "", things.ErrMalformedEntity
		}

		return "", err
	}

	return thing.ID, nil
}

func (tr thingRepository) Update(thing things.Thing) error {
	q := `UPDATE things SET name = $1, metadata = $2 WHERE owner = $3 AND id = $4;`

	metadata := thing.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	res, err := tr.db.Exec(q, thing.Name, metadata, thing.Owner, thing.ID)
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

func (tr thingRepository) RetrieveByID(owner, id string) (things.Thing, error) {
	q := `SELECT name, type, key, metadata FROM things WHERE id = $1 AND owner = $2`
	thing := things.Thing{ID: id, Owner: owner}
	err := tr.db.
		QueryRow(q, id, owner).
		Scan(&thing.Name, &thing.Type, &thing.Key, &thing.Metadata)

	if err != nil {
		empty := things.Thing{}

		pqErr, ok := err.(*pq.Error)

		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return empty, things.ErrNotFound
		}

		return empty, err
	}

	return thing, nil
}

func (tr thingRepository) RetrieveByKey(key string) (string, error) {
	q := `SELECT id FROM things WHERE key = $1`
	var id string
	if err := tr.db.QueryRow(q, key).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", things.ErrNotFound
		}
		return "", err
	}

	return id, nil
}

func (tr thingRepository) RetrieveAll(owner string, offset, limit uint64) things.ThingsPage {
	q := `SELECT id, name, type, key, metadata FROM things WHERE owner = $1 ORDER BY id LIMIT $2 OFFSET $3`
	items := []things.Thing{}

	rows, err := tr.db.Query(q, owner, limit, offset)
	if err != nil {
		tr.log.Error(fmt.Sprintf("Failed to retrieve things due to %s", err))
		return things.ThingsPage{}
	}
	defer rows.Close()

	for rows.Next() {
		c := things.Thing{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Key, &c.Metadata); err != nil {
			tr.log.Error(fmt.Sprintf("Failed to read retrieved thing due to %s", err))
			return things.ThingsPage{}
		}
		items = append(items, c)
	}

	q = `SELECT COUNT(*) FROM things WHERE owner = $1`

	var total uint64
	if err := tr.db.QueryRow(q, owner).Scan(&total); err != nil {
		tr.log.Error(fmt.Sprintf("Failed to count things due to %s", err))
		return things.ThingsPage{}
	}

	page := things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page
}

func (tr thingRepository) RetrieveByChannel(owner, channel string, offset, limit uint64) things.ThingsPage {
	q := `SELECT id, type, name, key, metadata
	      FROM things th
	      INNER JOIN connections co
		  ON th.id = co.thing_id
		  WHERE th.owner = $1 AND co.channel_id = $2
		  ORDER BY th.id
		  LIMIT $3
		  OFFSET $4`
	items := []things.Thing{}

	rows, err := tr.db.Query(q, owner, channel, limit, offset)
	if err != nil {
		tr.log.Error(fmt.Sprintf("Failed to retrieve things due to %s", err))
		return things.ThingsPage{}
	}
	defer rows.Close()

	for rows.Next() {
		t := things.Thing{Owner: owner}
		if err := rows.Scan(&t.ID, &t.Type, &t.Name, &t.Key, &t.Metadata); err != nil {
			tr.log.Error(fmt.Sprintf("Failed to read retrieved thing due to %s", err))
			return things.ThingsPage{}
		}
		items = append(items, t)
	}

	q = `SELECT COUNT(*)
	     FROM things th
	     INNER JOIN connections co
	     ON th.id = co.thing_id
	     WHERE th.owner = $1 AND co.channel_id = $2`

	var total uint64
	if err := tr.db.QueryRow(q, owner, channel).Scan(&total); err != nil {
		tr.log.Error(fmt.Sprintf("Failed to count things due to %s", err))
		return things.ThingsPage{}
	}

	return things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}
}

func (tr thingRepository) Remove(owner, id string) error {
	q := `DELETE FROM things WHERE id = $1 AND owner = $2`
	tr.db.Exec(q, id, owner)
	return nil
}
