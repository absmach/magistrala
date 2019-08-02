//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
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
	db  *sqlx.DB
	log logger.Logger
}

// NewConfigRepository instantiates a PostgreSQL implementation of config
// repository.
func NewConfigRepository(db *sqlx.DB, log logger.Logger) bootstrap.ConfigRepository {
	return &configRepository{db: db, log: log}
}

func (cr configRepository) Save(cfg bootstrap.Config, connections []string) (string, error) {
	q := `INSERT INTO configs (mainflux_thing, owner, name, client_cert, client_key, ca_cert, mainflux_key, external_id, external_key, content, state)
		  VALUES (:mainflux_thing, :owner, :name, :client_cert, :client_key, :ca_cert, :mainflux_key, :external_id, :external_key, :content, :state)`

	tx, err := cr.db.Beginx()
	if err != nil {
		return "", err
	}

	dbcfg := toDBConfig(cfg)

	if _, err := tx.NamedExec(q, dbcfg); err != nil {
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

	q = "DELETE FROM unknown_configs WHERE external_id = :external_id AND external_key = :external_key"

	if _, err := tx.NamedExec(q, dbcfg); err != nil {
		cr.rollback("Failed to remove from unknown", tx, err)

		return "", err
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config save", tx, err)
	}

	return cfg.MFThing, nil
}

func (cr configRepository) RetrieveByID(key, id string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, external_id, external_key, name, content, state 
		  FROM configs 
		  WHERE mainflux_thing = $1 AND owner = $2`

	dbcfg := dbConfig{
		MFThing: id,
		Owner:   key,
	}

	if err := cr.db.QueryRowx(q, id, key).StructScan(&dbcfg); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, bootstrap.ErrNotFound
		}

		return empty, err
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
		 INNER JOIN connections conn
		 ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
		 WHERE conn.config_id = :mainflux_thing AND conn.config_owner = :owner`

	rows, err := cr.db.NamedQuery(q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, err
	}
	defer rows.Close()

	chans := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, err
		}
		dbch.Owner = nullString(dbcfg.Owner)

		ch, err := toChannel(dbch)
		if err != nil {
			return bootstrap.Config{}, err
		}
		chans = append(chans, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.MFChannels = chans

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

func (cr configRepository) RetrieveByExternalID(externalID string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, external_key, owner, name, client_cert, client_key, ca_cert, content, state 
		  FROM configs 
		  WHERE external_id = $1`
	dbcfg := dbConfig{
		ExternalID: externalID,
	}

	if err := cr.db.QueryRowx(q, externalID).StructScan(&dbcfg); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, bootstrap.ErrNotFound
		}
		return empty, err
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
		 INNER JOIN connections conn
		 ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
		 WHERE conn.config_id = :mainflux_thing AND conn.config_owner = :owner`

	rows, err := cr.db.NamedQuery(q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, err
	}
	defer rows.Close()

	channels := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, err
		}

		ch, err := toChannel(dbch)
		if err != nil {
			cr.log.Error(fmt.Sprintf("Failed to deserialize channel due to %s", err))
			return bootstrap.Config{}, err
		}

		channels = append(channels, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.MFChannels = channels

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

func (cr configRepository) UpdateCert(key, thingKey, clientCert, clientKey, caCert string) error {
	q := `UPDATE configs SET client_cert = $1, client_key = $2, ca_cert = $3 WHERE mainflux_key = $4 AND owner = $5`

	res, err := cr.db.Exec(q, clientCert, clientKey, caCert, thingKey, key)
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
	tx, err := cr.db.Beginx()
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
	var channels []bootstrap.Channel
	if len(ids) == 0 {
		return channels, nil
	}

	q := "SELECT mainflux_channel, name, metadata FROM channels WHERE owner = $1 AND mainflux_channel = ANY ($2)"
	rows, err := cr.db.Queryx(q, key, pq.Array(ids))
	if err != nil {
		return []bootstrap.Channel{}, err
	}

	for rows.Next() {
		var dbch dbChannel
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channels due to %s", err))
			return []bootstrap.Channel{}, err
		}

		ch, err := toChannel(dbch)
		if err != nil {
			cr.log.Error(fmt.Sprintf("Failed to deserialize channel due to %s", err))
			return []bootstrap.Channel{}, err
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
	dbch, err := toDBChannel("", channel)
	if err != nil {
		return err
	}

	q := `UPDATE channels SET name = :name, metadata = :metadata WHERE mainflux_channel = :mainflux_channel`
	_, err = cr.db.NamedExec(q, dbch)

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

func (cr configRepository) rollback(content string, tx *sqlx.Tx, err error) {
	cr.log.Error(fmt.Sprintf("%s %s", content, err))

	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

func insertChannels(key string, channels []bootstrap.Channel, tx *sqlx.Tx) error {
	if len(channels) == 0 {
		return nil
	}

	var chans []dbChannel
	for _, ch := range channels {
		dbch, err := toDBChannel(key, ch)
		if err != nil {
			return err
		}
		chans = append(chans, dbch)
	}

	q := `INSERT INTO channels (mainflux_channel, owner, name, metadata) 
		  VALUES (:mainflux_channel, :owner, :name, :metadata)`
	if _, err := tx.NamedExec(q, chans); err != nil {
		e := err
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == duplicateErr {
			e = bootstrap.ErrConflict
		}
		return e
	}

	return nil
}

func insertConnections(cfg bootstrap.Config, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner) 
		  VALUES (:config_id, :channel_id, :config_owner, :channel_owner)`
	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:       cfg.MFThing,
			Channel:      conn,
			ConfigOwner:  cfg.Owner,
			ChannelOwner: cfg.Owner,
		}
		conns = append(conns, dbconn)
	}
	_, err := tx.NamedExec(q, conns)

	return err
}

func updateConnections(key, id string, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `DELETE FROM connections
		  WHERE config_id = $1 AND config_owner = $2 AND channel_owner = $2
		  AND channel_id NOT IN ($3)`

	res, err := tx.Exec(q, id, key, pq.Array(connections))
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	q = `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner) 
		 VALUES (:config_id, :channel_id, :config_owner, :channel_owner)`

	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:       id,
			Channel:      conn,
			ConfigOwner:  key,
			ChannelOwner: key,
		}
		conns = append(conns, dbconn)
	}

	if _, err := tx.NamedExec(q, conns); err != nil {
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

type dbConfig struct {
	MFThing     string          `db:"mainflux_thing"`
	Owner       string          `db:"owner"`
	Name        sql.NullString  `db:"name"`
	ClientCert  sql.NullString  `db:"client_cert"`
	ClientKey   sql.NullString  `db:"client_key"`
	CaCert      sql.NullString  `db:"ca_cert"`
	MFKey       string          `db:"mainflux_key"`
	ExternalID  string          `db:"external_id"`
	ExternalKey string          `db:"external_key"`
	Content     sql.NullString  `db:"content"`
	State       bootstrap.State `db:"state"`
}

func toDBConfig(cfg bootstrap.Config) dbConfig {
	return dbConfig{
		MFThing:     cfg.MFThing,
		Owner:       cfg.Owner,
		Name:        nullString(cfg.Name),
		ClientCert:  nullString(cfg.ClientCert),
		ClientKey:   nullString(cfg.ClientKey),
		CaCert:      nullString(cfg.CACert),
		MFKey:       cfg.MFKey,
		ExternalID:  cfg.ExternalID,
		ExternalKey: cfg.ExternalKey,
		Content:     nullString(cfg.Content),
		State:       cfg.State,
	}
}

func toConfig(dbcfg dbConfig) bootstrap.Config {
	cfg := bootstrap.Config{
		MFThing:     dbcfg.MFThing,
		Owner:       dbcfg.Owner,
		MFKey:       dbcfg.MFKey,
		ExternalID:  dbcfg.ExternalID,
		ExternalKey: dbcfg.ExternalKey,
		State:       dbcfg.State,
	}

	if dbcfg.Name.Valid {
		cfg.Name = dbcfg.Name.String
	}

	if dbcfg.Content.Valid {
		cfg.Content = dbcfg.Content.String
	}

	if dbcfg.ClientCert.Valid {
		cfg.ClientCert = dbcfg.ClientCert.String
	}

	if dbcfg.ClientKey.Valid {
		cfg.ClientKey = dbcfg.ClientKey.String
	}

	if dbcfg.CaCert.Valid {
		cfg.CACert = dbcfg.CaCert.String
	}
	return cfg
}

type dbChannel struct {
	ID       string         `db:"mainflux_channel"`
	Name     sql.NullString `db:"name"`
	Owner    sql.NullString `db:"owner"`
	Metadata string         `db:"metadata"`
}

func toDBChannel(owner string, ch bootstrap.Channel) (dbChannel, error) {
	dbch := dbChannel{
		ID:    ch.ID,
		Name:  nullString(ch.Name),
		Owner: nullString(owner),
	}

	metadata, err := json.Marshal(ch.Metadata)
	if err != nil {
		return dbChannel{}, err
	}

	dbch.Metadata = string(metadata)
	return dbch, nil
}

func toChannel(dbch dbChannel) (bootstrap.Channel, error) {
	ch := bootstrap.Channel{
		ID: dbch.ID,
	}

	if dbch.Name.Valid {
		ch.Name = dbch.Name.String
	}

	if err := json.Unmarshal([]byte(dbch.Metadata), &ch.Metadata); err != nil {
		return bootstrap.Channel{}, err
	}

	return ch, nil
}

type dbConnection struct {
	Config       string `db:"config_id"`
	Channel      string `db:"channel_id"`
	ConfigOwner  string `db:"config_owner"`
	ChannelOwner string `db:"channel_owner"`
}
