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
	"strings"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/logger"
)

const (
	duplicateErr      = "unique_violation"
	uuidErr           = "invalid input syntax for type uuid"
	connConstraintErr = "connections_config_id_fkey"
	fkViolation       = "foreign_key_violation"
	configFieldsNum   = 8
	chanFieldsNum     = 3
	connFieldsNum     = 2
	cleanupQuery      = `DELETE FROM channels ch WHERE NOT EXISTS (
						 SELECT channel_id FROM connections c WHERE ch.mainflux_channel = c.channel_id);`
)

var _ bootstrap.ConfigRepository = (*configRepository)(nil)

type configRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewConfigRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewConfigRepository(db *sql.DB, log logger.Logger) bootstrap.ConfigRepository {
	return &configRepository{db: db, log: log}
}

func (cr configRepository) Save(cfg bootstrap.Config, connections []string) (string, error) {
	q := `INSERT INTO configs (mainflux_thing, owner, name, mainflux_key, external_id, external_key, content, state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	content := nullString(cfg.Content)
	name := nullString(cfg.Name)
	tx, err := cr.db.Begin()

	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(q, cfg.MFThing, cfg.Owner, name, cfg.MFKey, cfg.ExternalID, cfg.ExternalKey, content, cfg.State); err != nil {
		e := err
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == duplicateErr {
			e = bootstrap.ErrConflict
		}

		cr.rollback("Failed to insert a Config", tx, err)

		return "", e
	}

	if err := insertChannels(cfg.Owner, cfg.MFChannels, tx); err != nil {
		cr.rollback("Failed to insert Channels", tx, err)

		return "", err
	}

	if err := insertConnections(cfg, connections, tx); err != nil {
		cr.rollback("Failed to insert connections", tx, err)

		return "", err
	}

	q = "DELETE FROM unknown_configs WHERE external_id = $1 AND external_key = $2"

	if _, err := tx.Exec(q, cfg.ExternalID, cfg.ExternalKey); err != nil {
		cr.rollback("Failed to remove from unknown", tx, err)

		return "", err
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config save", tx, err)
	}

	return cfg.MFThing, nil
}

func (cr configRepository) RetrieveByID(key, id string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, external_id, external_key, name, content, state FROM configs WHERE mainflux_thing = $1 AND owner = $2`
	cfg := bootstrap.Config{MFThing: id, Owner: key, MFChannels: []bootstrap.Channel{}}
	var name, content sql.NullString
	if err := cr.db.QueryRow(q, id, key).
		Scan(&cfg.MFThing, &cfg.MFKey, &cfg.ExternalID, &cfg.ExternalKey, &name, &content, &cfg.State); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, bootstrap.ErrNotFound
		}

		return empty, err
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
	INNER JOIN connections conn
	ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
	WHERE conn.config_id = $1 AND conn.config_owner = $2`

	rows, err := cr.db.Query(q, cfg.MFThing, cfg.Owner)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, err
	}
	defer rows.Close()

	for rows.Next() {
		c := bootstrap.Channel{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, err
		}

		cfg.MFChannels = append(cfg.MFChannels, c)
	}

	cfg.Content = content.String
	cfg.Name = name.String

	return cfg, nil
}

func (cr configRepository) RetrieveAll(key string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	search, params := cr.retrieveAll(key, filter)
	n := len(params)

	q := `SELECT mainflux_thing, mainflux_key, external_id, external_key, name, content, state
	FROM configs %s ORDER BY mainflux_thing LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, search, n+1, n+2)

	rows, err := cr.db.Query(q, append(params, limit, offset)...)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}
	defer rows.Close()

	var name, content sql.NullString
	configs := []bootstrap.Config{}

	for rows.Next() {
		c := bootstrap.Config{Owner: key}
		if err := rows.Scan(&c.MFThing, &c.MFKey, &c.ExternalID, &c.ExternalKey, &name, &content, &c.State); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return bootstrap.ConfigsPage{}
		}

		c.Name = name.String
		c.Content = content.String
		configs = append(configs, c)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM configs %s`, search)

	var total uint64
	if err := cr.db.QueryRow(q, params...).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}

	return bootstrap.ConfigsPage{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Configs: configs,
	}
}

func (cr configRepository) RetrieveByExternalID(externalKey, externalID string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, owner, name, content, state FROM configs WHERE external_key = $1 AND external_id = $2`
	cfg := bootstrap.Config{ExternalID: externalID, ExternalKey: externalKey, MFChannels: []bootstrap.Channel{}}
	var name, content sql.NullString

	if err := cr.db.QueryRow(q, externalKey, externalID).
		Scan(&cfg.MFThing, &cfg.MFKey, &cfg.Owner, &name, &content, &cfg.State); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, bootstrap.ErrNotFound
		}
		return empty, err
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
	INNER JOIN connections conn
	ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
	WHERE conn.config_id = $1 AND conn.config_owner = $2`

	rows, err := cr.db.Query(q, cfg.MFThing, cfg.Owner)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, err
	}
	defer rows.Close()

	for rows.Next() {
		c := bootstrap.Channel{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, err
		}

		cfg.MFChannels = append(cfg.MFChannels, c)
	}

	cfg.Content = content.String
	cfg.Name = name.String

	return cfg, nil
}

func (cr configRepository) Update(cfg bootstrap.Config) error {
	q := `UPDATE configs SET name = $1, content = $2 WHERE mainflux_thing = $3 AND owner = $4`

	content := nullString(cfg.Content)
	name := nullString(cfg.Name)

	res, err := cr.db.Exec(q, name, content, cfg.MFThing, cfg.Owner)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return bootstrap.ErrNotFound
	}

	return nil
}

func (cr configRepository) UpdateConnections(key, id string, channels []bootstrap.Channel, connections []string) error {
	tx, err := cr.db.Begin()
	if err != nil {
		return err
	}

	if err := insertChannels(key, channels, tx); err != nil {
		cr.rollback("Failed to insert Channels during the update", tx, err)

		return err
	}

	if err := updateConnections(key, id, connections, tx); err != nil {
		if e, ok := err.(*pq.Error); ok {
			if e.Code.Name() == fkViolation && e.Constraint == connConstraintErr {
				return bootstrap.ErrNotFound
			}
		}
		cr.rollback("Failed to update connections during the update", tx, err)

		return err
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config update", tx, err)
	}

	return nil
}

func (cr configRepository) Remove(key, id string) error {
	q := `DELETE FROM configs WHERE mainflux_thing = $1 AND owner = $2`
	if _, err := cr.db.Exec(q, id, key); err != nil {
		return err
	}

	if _, err := cr.db.Exec(cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}

	return nil
}

func (cr configRepository) ChangeState(key, id string, state bootstrap.State) error {
	q := `UPDATE configs SET state = $1 WHERE mainflux_thing = $2 AND owner = $3;`

	res, err := cr.db.Exec(q, state, id, key)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return bootstrap.ErrNotFound
	}

	return nil
}

func (cr configRepository) ListExisting(key string, ids []string) ([]bootstrap.Channel, error) {
	q := "SELECT mainflux_channel, name, metadata FROM channels WHERE owner = $1 AND mainflux_channel = ANY ($2)"

	rows, err := cr.db.Query(q, key, pq.Array(ids))
	if err != nil {
		return []bootstrap.Channel{}, err
	}

	var channels []bootstrap.Channel
	for rows.Next() {
		var ch bootstrap.Channel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Metadata); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channels due to %s", err))
			return []bootstrap.Channel{}, nil
		}

		channels = append(channels, ch)
	}

	return channels, nil
}

func (cr configRepository) SaveUnknown(key, id string) error {
	q := `INSERT INTO unknown_configs (external_id, external_key) VALUES ($1, $2)`

	if _, err := cr.db.Exec(q, id, key); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == duplicateErr {
			return nil
		}
		return err
	}

	return nil
}

func (cr configRepository) RetrieveUnknown(offset, limit uint64) bootstrap.ConfigsPage {
	q := `SELECT external_id, external_key FROM unknown_configs LIMIT $1 OFFSET $2`
	rows, err := cr.db.Query(q, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve config due to %s", err))
		return bootstrap.ConfigsPage{}
	}
	defer rows.Close()

	items := []bootstrap.Config{}
	for rows.Next() {
		c := bootstrap.Config{}
		if err := rows.Scan(&c.ExternalID, &c.ExternalKey); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return bootstrap.ConfigsPage{}
		}

		items = append(items, c)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM unknown_configs`)

	var total uint64
	if err := cr.db.QueryRow(q).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count unknown configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}

	return bootstrap.ConfigsPage{
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		Configs: items,
	}
}

func (cr configRepository) RemoveThing(id string) error {
	q := `DELETE FROM configs WHERE mainflux_thing = $1`
	_, err := cr.db.Exec(q, id)

	if _, err := cr.db.Exec(cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}

	return err
}

func (cr configRepository) UpdateChannel(channel bootstrap.Channel) error {
	q := `UPDATE channels SET name = $1, metadata = $2 WHERE mainflux_channel = $3`
	_, err := cr.db.Exec(q, channel.Name, channel.Metadata, channel.ID)

	return err
}

func (cr configRepository) RemoveChannel(id string) error {
	q := `DELETE FROM channels WHERE mainflux_channel = $1`
	_, err := cr.db.Exec(q, id)

	return err
}

func (cr configRepository) DisconnectThing(channelID, thingID string) error {
	q := `UPDATE configs SET state = $1 WHERE EXISTS (
		SELECT 1 FROM connections WHERE config_id = $2 AND channel_id = $3)`
	_, err := cr.db.Exec(q, bootstrap.Inactive, thingID, channelID)

	return err
}

func (cr configRepository) retrieveAll(key string, filter bootstrap.Filter) (string, []interface{}) {
	template := `WHERE owner = $1 %s`
	params := []interface{}{key}
	// One empty string so that strings Join works if only one filter is applied.
	queries := []string{""}
	// Since key is the first param, start from 2.
	counter := 2
	for k, v := range filter.FullMatch {
		queries = append(queries, fmt.Sprintf("%s = $%d", k, counter))
		params = append(params, v)
		counter++
	}
	for k, v := range filter.PartialMatch {
		queries = append(queries, fmt.Sprintf("LOWER(%s) LIKE '%%' || $%d || '%%'", k, counter))
		params = append(params, v)
		counter++
	}

	f := strings.Join(queries, " AND ")

	return fmt.Sprintf(template, f), params
}

func (cr configRepository) rollback(content string, tx *sql.Tx, err error) {
	cr.log.Error(fmt.Sprintf("%s %s", content, err))

	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

func insertChannels(key string, channels []bootstrap.Channel, tx *sql.Tx) error {
	if len(channels) == 0 {
		return nil
	}

	q := `INSERT INTO channels (mainflux_channel, owner, name, metadata) VALUES `
	v := []interface{}{key}
	var vals []string
	// Since the first value is owner, start with the second one.
	count := 2
	for _, ch := range channels {
		vals = append(vals, fmt.Sprintf("($%d, $1, $%d, $%d)", count, count+1, count+2))
		v = append(v, ch.ID, ch.Name, ch.Metadata)
		count += chanFieldsNum
	}

	q = fmt.Sprintf("%s%s", q, strings.Join(vals, ","))
	if _, err := tx.Exec(q, v...); err != nil {
		e := err
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == duplicateErr {
			e = bootstrap.ErrConflict
		}
		return e
	}

	return nil
}

func insertConnections(cfg bootstrap.Config, connections []string, tx *sql.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner) VALUES`
	v := []interface{}{cfg.MFThing, cfg.Owner}
	var vals []string

	// Since the first value is Config ID and the second and third
	// are Config owner, start with  the second one.
	count := 3
	for _, id := range connections {
		vals = append(vals, fmt.Sprintf("($1, $%d, $2, $2)", count))
		v = append(v, id)
		count++
	}

	q = fmt.Sprintf("%s%s", q, strings.Join(vals, ","))
	_, err := tx.Exec(q, v...)

	return err
}

func updateConnections(key, id string, connections []string, tx *sql.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `DELETE FROM connections
	WHERE config_id = $1 AND config_owner = $2 AND channel_owner = $2
	AND channel_id NOT IN ($3)`

	v := []interface{}{id, key}
	v = append(v, pq.Array(connections))

	res, err := tx.Exec(q, v...)

	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	q = `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner) VALUES`
	v = []interface{}{id, key}
	var vals []string

	// Since the first value is Config ID and the second is Config
	// owner, start with the third one.
	count := 3
	for _, chID := range connections {
		vals = append(vals, fmt.Sprintf("($1, $%d, $2, $2)", count))
		v = append(v, chID)
		count++
	}

	// Add connections for current list of channels. Ignore if already exists.
	q = fmt.Sprintf("%s%s%s", q, strings.Join(vals, ","), "ON CONFLICT (config_id, config_owner, channel_id, channel_owner) DO NOTHING")
	if _, err := tx.Exec(q, v...); err != nil {
		return err
	}

	if cnt == 0 {
		return nil
	}

	_, err = tx.Exec(cleanupQuery)

	return err
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
