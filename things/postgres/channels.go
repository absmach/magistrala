// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
)

var _ things.ChannelRepository = (*channelRepository)(nil)

type channelRepository struct {
	db Database
}

type dbConnection struct {
	Channel string `db:"channel"`
	Thing   string `db:"thing"`
	Owner   string `db:"owner"`
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewChannelRepository(db Database) things.ChannelRepository {
	return &channelRepository{
		db: db,
	}
}

func (cr channelRepository) Save(ctx context.Context, channels ...things.Channel) ([]things.Channel, error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO channels (id, owner, name, metadata)
		  VALUES (:id, :owner, :name, :metadata);`

	for _, channel := range channels {
		dbch := toDBChannel(channel)

		_, err = tx.NamedExecContext(ctx, q, dbch)
		if err != nil {
			tx.Rollback()
			pqErr, ok := err.(*pq.Error)
			if ok {
				switch pqErr.Code.Name() {
				case errInvalid, errTruncation:
					return []things.Channel{}, errors.ErrMalformedEntity
				case errDuplicate:
					return []things.Channel{}, errors.ErrConflict
				}
			}
			return []things.Channel{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Channel{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return channels, nil
}

func (cr channelRepository) Update(ctx context.Context, channel things.Channel) error {
	q := `UPDATE channels SET name = :name, metadata = :metadata WHERE owner = :owner AND id = :id;`

	dbch := toDBChannel(channel)

	res, err := cr.db.NamedExecContext(ctx, q, dbch)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return errors.ErrMalformedEntity
			}
		}

		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (cr channelRepository) RetrieveByID(ctx context.Context, owner, id string) (things.Channel, error) {
	q := `SELECT name, metadata, owner FROM channels WHERE id = $1;`

	dbch := dbChannel{
		ID: id,
	}
	if err := cr.db.QueryRowxContext(ctx, q, id).StructScan(&dbch); err != nil {
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return things.Channel{}, errors.ErrNotFound
		}
		return things.Channel{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	return toChannel(dbch), nil
}

func (cr channelRepository) RetrieveAll(ctx context.Context, owner string, pm things.PageMetadata) (things.ChannelsPage, error) {
	nq, name := getNameQuery(pm.Name)
	oq := getOrderQuery(pm.Order)
	dq := getDirQuery(pm.Dir)
	ownerQuery := getOwnerQuery(pm.FetchSharedThings)
	meta, mq, err := getMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	var whereClause string
	var query []string
	if mq != "" {
		query = append(query, mq)
	}
	if nq != "" {
		query = append(query, nq)
	}
	if ownerQuery != "" {
		query = append(query, ownerQuery)
	}

	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT id, name, metadata FROM channels
		%s ORDER BY %s %s LIMIT :limit OFFSET :offset;`, whereClause, oq, dq)

	params := map[string]interface{}{
		"owner":    owner,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": meta,
	}
	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
		}
		ch := toChannel(dbch)

		items = append(items, ch)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM channels %s;`, whereClause)

	total, err := total(ctx, cr.db, cq, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	page := things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return page, nil
}

func (cr channelRepository) RetrieveByThing(ctx context.Context, owner, thID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	oq := getConnOrderQuery(pm.Order, "ch")
	dq := getDirQuery(pm.Dir)

	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thID); err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrNotFound, err)
	}

	var q, qc string
	switch pm.Disconnected {
	case true:
		q = fmt.Sprintf(`SELECT id, name, metadata
		        FROM channels ch
		        WHERE ch.owner = :owner AND ch.id NOT IN
		        (SELECT id FROM channels ch
		          INNER JOIN connections conn
		          ON ch.id = conn.channel_id
		          WHERE ch.owner = :owner AND conn.thing_id = :thing)
		        ORDER BY %s %s
		        LIMIT :limit
		        OFFSET :offset;`, oq, dq)

		qc = `SELECT COUNT(*)
		        FROM channels ch
		        WHERE ch.owner = $1 AND ch.id NOT IN
		        (SELECT id FROM channels ch
		          INNER JOIN connections conn
		          ON ch.id = conn.channel_id
		          WHERE ch.owner = $1 AND conn.thing_id = $2);`
	default:
		q = fmt.Sprintf(`SELECT id, name, metadata FROM channels ch
		        INNER JOIN connections conn
		        ON ch.id = conn.channel_id
		        WHERE ch.owner = :owner AND conn.thing_id = :thing
		        ORDER BY %s %s
		        LIMIT :limit
		        OFFSET :offset;`, oq, dq)

		qc = `SELECT COUNT(*)
		        FROM channels ch
		        INNER JOIN connections conn
		        ON ch.id = conn.channel_id
		        WHERE ch.owner = $1 AND conn.thing_id = $2`
	}

	params := map[string]interface{}{
		"owner":  owner,
		"thing":  thID,
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
		}

		ch := toChannel(dbch)
		items = append(items, ch)
	}

	var total uint64
	if err := cr.db.GetContext(ctx, &total, qc, owner, thID); err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	return things.ChannelsPage{
		Channels: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (cr channelRepository) Remove(ctx context.Context, owner, id string) error {
	dbch := dbChannel{
		ID:    id,
		Owner: owner,
	}
	q := `DELETE FROM channels WHERE id = :id AND owner = :owner`
	cr.db.NamedExecContext(ctx, q, dbch)
	return nil
}

func (cr channelRepository) Connect(ctx context.Context, owner string, chIDs, thIDs []string) error {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	q := `INSERT INTO connections (channel_id, channel_owner, thing_id, thing_owner)
	      VALUES (:channel, :owner, :thing, :owner);`

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			dbco := dbConnection{
				Channel: chID,
				Thing:   thID,
				Owner:   owner,
			}

			_, err := tx.NamedExecContext(ctx, q, dbco)
			if err != nil {
				tx.Rollback()
				pqErr, ok := err.(*pq.Error)
				if ok {
					switch pqErr.Code.Name() {
					case errFK:
						return errors.ErrNotFound
					case errDuplicate:
						return errors.ErrConflict
					}
				}

				return errors.Wrap(things.ErrConnect, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	return nil
}

func (cr channelRepository) Disconnect(ctx context.Context, owner string, chIDs, thIDs []string) error {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	q := `DELETE FROM connections
	      WHERE channel_id = :channel AND channel_owner = :owner
	      AND thing_id = :thing AND thing_owner = :owner`

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			dbco := dbConnection{
				Channel: chID,
				Thing:   thID,
				Owner:   owner,
			}

			res, err := tx.NamedExecContext(ctx, q, dbco)
			if err != nil {
				tx.Rollback()
				pqErr, ok := err.(*pq.Error)
				if ok {
					switch pqErr.Code.Name() {
					case errFK:
						return errors.ErrNotFound
					case errDuplicate:
						return errors.ErrConflict
					}
				}
				return errors.Wrap(things.ErrDisconnect, err)
			}

			cnt, err := res.RowsAffected()
			if err != nil {
				return errors.Wrap(things.ErrDisconnect, err)
			}

			if cnt == 0 {
				return errors.ErrNotFound
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	return nil
}

func (cr channelRepository) HasThing(ctx context.Context, chanID, thingKey string) (string, error) {
	var thingID string
	q := `SELECT id FROM things WHERE key = $1`
	if err := cr.db.QueryRowxContext(ctx, q, thingKey).Scan(&thingID); err != nil {
		return "", errors.Wrap(things.ErrEntityConnected, err)
	}

	if err := cr.hasThing(ctx, chanID, thingID); err != nil {
		return "", err
	}

	return thingID, nil
}

func (cr channelRepository) HasThingByID(ctx context.Context, chanID, thingID string) error {
	return cr.hasThing(ctx, chanID, thingID)
}

func (cr channelRepository) hasThing(ctx context.Context, chanID, thingID string) error {
	q := `SELECT EXISTS (SELECT 1 FROM connections WHERE channel_id = $1 AND thing_id = $2);`
	exists := false
	if err := cr.db.QueryRowxContext(ctx, q, chanID, thingID).Scan(&exists); err != nil {
		return errors.Wrap(things.ErrEntityConnected, err)
	}

	if !exists {
		return errors.ErrNotFound
	}

	return nil
}

// dbMetadata type for handling metadata properly in database/sql.
type dbMetadata map[string]interface{}

// Scan implements the database/sql scanner interface.
// When interface is nil `m` is set to nil.
// If error occurs on casting data then m points to empty metadata.
func (m *dbMetadata) Scan(value interface{}) error {
	if value == nil {
		m = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		m = &dbMetadata{}
		return errors.ErrScanMetadata
	}

	if err := json.Unmarshal(b, m); err != nil {
		return err
	}

	return nil
}

// Value implements database/sql valuer interface.
func (m dbMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, err
}

type dbChannel struct {
	ID       string     `db:"id"`
	Owner    string     `db:"owner"`
	Name     string     `db:"name"`
	Metadata dbMetadata `db:"metadata"`
}

func toDBChannel(ch things.Channel) dbChannel {
	return dbChannel{
		ID:       ch.ID,
		Owner:    ch.Owner,
		Name:     ch.Name,
		Metadata: ch.Metadata,
	}
}

func toChannel(ch dbChannel) things.Channel {
	return things.Channel{
		ID:       ch.ID,
		Owner:    ch.Owner,
		Name:     ch.Name,
		Metadata: ch.Metadata,
	}
}

func getNameQuery(name string) (string, string) {
	if name == "" {
		return "", ""
	}
	name = fmt.Sprintf(`%%%s%%`, strings.ToLower(name))
	nq := `LOWER(name) LIKE :name`
	return nq, name
}

func getOrderQuery(order string) string {
	switch order {
	case "name":
		return "name"
	default:
		return "id"
	}
}

func getConnOrderQuery(order string, level string) string {
	switch order {
	case "name":
		return level + ".name"
	default:
		return level + ".id"
	}
}

func getDirQuery(dir string) string {
	switch dir {
	case "asc":
		return "ASC"
	default:
		return "DESC"
	}
}

func getMetadataQuery(m things.Metadata) ([]byte, string, error) {
	mq := ""
	mb := []byte("{}")
	if len(m) > 0 {
		mq = `metadata @> :metadata`

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", err
		}
		mb = b
	}
	return mb, mq, nil
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}
