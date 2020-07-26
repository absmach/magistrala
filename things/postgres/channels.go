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

var (
	// ErrSaveChannel indicates error while saving to database
	ErrSaveChannel = errors.New("save channel to db error")
	// ErrUpdateChannel indicates error while updating channel in database
	ErrUpdateChannel = errors.New("update channel to db error")
	// ErrDeleteChannel indicates error while deleting channel in database
	ErrDeleteChannel = errors.New("delete channel from db error")
	// ErrSelectChannel indicates error while reading channel from database
	ErrSelectChannel = errors.New("select channel from db error")
	// ErrDeleteConnection indicates error while deleting connection in database
	ErrDeleteConnection = errors.New("unmarshal json error")
	// ErrHasThing indicates error while checking connection in database
	ErrHasThing = errors.New("check thing-channel connection in database error")
	//ErrScan indicates error in database scanner
	ErrScan = errors.New("database scanner error")
	//ErrValue indicates error in database valuer
	ErrValue = errors.New("database valuer error")
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
		return nil, errors.Wrap(ErrSaveChannel, err)
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
					return []things.Channel{}, things.ErrMalformedEntity
				case errDuplicate:
					return []things.Channel{}, things.ErrConflict
				}
			}
			return []things.Channel{}, errors.Wrap(ErrSaveChannel, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Channel{}, errors.Wrap(ErrSaveChannel, err)
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
				return things.ErrMalformedEntity
			}
		}

		return errors.Wrap(ErrUpdateChannel, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(ErrUpdateChannel, err)
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (cr channelRepository) RetrieveByID(ctx context.Context, owner, id string) (things.Channel, error) {
	q := `SELECT name, metadata FROM channels WHERE id = $1 AND owner = $2;`

	dbch := dbChannel{
		ID:    id,
		Owner: owner,
	}
	if err := cr.db.QueryRowxContext(ctx, q, id, owner).StructScan(&dbch); err != nil {
		empty := things.Channel{}
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return empty, things.ErrNotFound
		}
		return empty, errors.Wrap(ErrSelectChannel, err)
	}

	return toChannel(dbch), nil
}

func (cr channelRepository) RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string, metadata things.Metadata) (things.ChannelsPage, error) {
	nq, name := getNameQuery(name)
	m, mq, err := getMetadataQuery(metadata)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
	}

	q := fmt.Sprintf(`SELECT id, name, metadata FROM channels
	      WHERE owner = :owner %s%s ORDER BY id LIMIT :limit OFFSET :offset;`, mq, nq)

	params := map[string]interface{}{
		"owner":    owner,
		"limit":    limit,
		"offset":   offset,
		"name":     name,
		"metadata": m,
	}
	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
		}
		ch := toChannel(dbch)

		items = append(items, ch)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM channels WHERE owner = :owner %s%s;`, nq, mq)

	total, err := total(ctx, cr.db, cq, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
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

func (cr channelRepository) RetrieveByThing(ctx context.Context, owner, thing string, offset, limit uint64, connected bool) (things.ChannelsPage, error) {
	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thing); err != nil {
		return things.ChannelsPage{}, things.ErrNotFound
	}

	var q, qc string
	switch connected {
	case true:
		q = `SELECT id, name, metadata FROM channels ch
		        INNER JOIN connections conn
		        ON ch.id = conn.channel_id
		        WHERE ch.owner = :owner AND conn.thing_id = :thing
		        ORDER BY ch.id
		        LIMIT :limit
		        OFFSET :offset;`

		qc = `SELECT COUNT(*)
		        FROM channels ch
		        INNER JOIN connections conn
		        ON ch.id = conn.channel_id
		        WHERE ch.owner = $1 AND conn.thing_id = $2`
	default:
		q = `SELECT id, name, metadata
		        FROM channels ch
		        WHERE ch.owner = :owner AND ch.id NOT IN
		        (SELECT id FROM channels ch
		          INNER JOIN connections conn
		          ON ch.id = conn.channel_id
		          WHERE ch.owner = :owner AND conn.thing_id = :thing)
		        ORDER BY ch.id
		        LIMIT :limit
		        OFFSET :offset;`

		qc = `SELECT COUNT(*)
		        FROM channels ch
		        WHERE ch.owner = $1 AND ch.id NOT IN
		        (SELECT id FROM channels ch
		          INNER JOIN connections conn
		          ON ch.id = conn.channel_id
		          WHERE ch.owner = $1 AND conn.thing_id = $2);`
	}

	params := map[string]interface{}{
		"owner":  owner,
		"thing":  thing,
		"limit":  limit,
		"offset": offset,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{Owner: owner}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
		}

		ch := toChannel(dbch)
		items = append(items, ch)
	}

	var total uint64
	if err := cr.db.GetContext(ctx, &total, qc, owner, thing); err != nil {
		return things.ChannelsPage{}, errors.Wrap(ErrSelectChannel, err)
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
		return errors.Wrap(ErrDeleteChannel, err)
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
						return things.ErrNotFound
					case errDuplicate:
						return things.ErrConflict
					}
				}

				return errors.Wrap(ErrDeleteChannel, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(ErrDeleteChannel, err)
	}

	return nil
}

func (cr channelRepository) Disconnect(ctx context.Context, owner, chanID, thingID string) error {
	q := `DELETE FROM connections
	      WHERE channel_id = :channel AND channel_owner = :owner
	      AND thing_id = :thing AND thing_owner = :owner`

	conn := dbConnection{
		Channel: chanID,
		Thing:   thingID,
		Owner:   owner,
	}

	res, err := cr.db.NamedExecContext(ctx, q, conn)
	if err != nil {
		return errors.Wrap(ErrDeleteConnection, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(ErrDeleteConnection, err)
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (cr channelRepository) HasThing(ctx context.Context, chanID, key string) (string, error) {
	var thingID string
	q := `SELECT id FROM things WHERE key = $1`
	if err := cr.db.QueryRowxContext(ctx, q, key).Scan(&thingID); err != nil {
		return "", errors.Wrap(ErrHasThing, err)

	}

	if err := cr.hasThing(ctx, chanID, thingID); err != nil {
		return "", errors.Wrap(ErrHasThing, err)
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
		return errors.Wrap(ErrHasThing, err)
	}

	if !exists {
		return things.ErrUnauthorizedAccess
	}

	return nil
}

// dbMetadata type for handling metadata properly in database/sql.
type dbMetadata map[string]interface{}

// Scan implements the database/sql scanner interface.
func (m *dbMetadata) Scan(value interface{}) error {
	if value == nil {
		m = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		m = &dbMetadata{}
		return things.ErrScanMetadata
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
	name = strings.ToLower(name)
	nq := ""
	if name != "" {
		name = fmt.Sprintf(`%%%s%%`, name)
		nq = ` AND LOWER(name) LIKE :name`
	}
	return nq, name
}

func getMetadataQuery(m things.Metadata) ([]byte, string, error) {
	mq := ""
	mb := []byte("{}")
	if len(m) > 0 {
		mq = ` AND metadata @> :metadata`

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", err
		}
		mb = b
	}
	return mb, mq, nil
}

func total(ctx context.Context, db Database, query string, params map[string]interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}

	return total, nil
}
