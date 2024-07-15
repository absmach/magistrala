// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	errSaveChannels    = errors.New("failed to insert channels to database")
	errSaveConnections = errors.New("failed to insert connections to database")
	errUpdateChannels  = errors.New("failed to update channels in bootstrap configuration database")
	errRemoveChannels  = errors.New("failed to remove channels from bootstrap configuration in database")
	errConnectThing    = errors.New("failed to connect thing in bootstrap configuration in database")
	errDisconnectThing = errors.New("failed to disconnect thing in bootstrap configuration in database")
)

const cleanupQuery = `DELETE FROM channels ch WHERE NOT EXISTS (
						 SELECT channel_id FROM connections c WHERE ch.magistrala_channel = c.channel_id);`

var _ bootstrap.ConfigRepository = (*configRepository)(nil)

type configRepository struct {
	db  postgres.Database
	log *slog.Logger
}

// NewConfigRepository instantiates a PostgreSQL implementation of config
// repository.
func NewConfigRepository(db postgres.Database, log *slog.Logger) bootstrap.ConfigRepository {
	return &configRepository{db: db, log: log}
}

func (cr configRepository) Save(ctx context.Context, cfg bootstrap.Config, chsConnIDs []string) (thingID string, err error) {
	q := `INSERT INTO configs (magistrala_thing, domain_id, name, client_cert, client_key, ca_cert, magistrala_key, external_id, external_key, content, state)
	VALUES (:magistrala_thing, :domain_id, :name, :client_cert, :client_key, :ca_cert, :magistrala_key, :external_id, :external_key, :content, :state)`

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	dbcfg := toDBConfig(cfg)

	defer func() {
		if err != nil {
			err = cr.rollback("Save method", err, tx)
		}
	}()

	if _, err := tx.NamedExec(q, dbcfg); err != nil {
		switch pgErr := err.(type) {
		case *pgconn.PgError:
			if pgErr.Code == pgerrcode.UniqueViolation {
				err = repoerr.ErrConflict
			}
		}
		return "", err
	}

	if err := insertChannels(cfg.DomainID, cfg.Channels, tx); err != nil {
		return "", errors.Wrap(errSaveChannels, err)
	}

	if err := insertConnections(ctx, cfg, chsConnIDs, tx); err != nil {
		return "", errors.Wrap(errSaveConnections, err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return "", commitErr
	}

	return cfg.ThingID, nil
}

func (cr configRepository) RetrieveByID(ctx context.Context, domainID, id string) (bootstrap.Config, error) {
	q := `SELECT magistrala_thing, magistrala_key, external_id, external_key, name, content, state, client_cert, ca_cert
		  FROM configs
		  WHERE magistrala_thing = :magistrala_thing AND domain_id = :domain_id`

	dbcfg := dbConfig{
		ThingID:  id,
		DomainID: domainID,
	}
	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrNotFound, err)
		}

		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	if ok := row.Next(); !ok {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, err
	}

	q = `SELECT magistrala_channel, name, metadata FROM channels ch
		 INNER JOIN connections conn
		 ON ch.magistrala_channel = conn.channel_id AND ch.domain_id = conn.domain_id
		 WHERE conn.config_id = :magistrala_thing AND conn.domain_id = :domain_id`

	rows, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	chans := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		dbch.DomainID = nullString(dbcfg.DomainID)

		ch, err := toChannel(dbch)
		if err != nil {
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		chans = append(chans, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.Channels = chans

	return cfg, nil
}

func (cr configRepository) RetrieveAll(ctx context.Context, domainID string, thingIDs []string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	search, params := buildRetrieveQueryParams(domainID, thingIDs, filter)
	n := len(params)

	q := `SELECT magistrala_thing, magistrala_key, external_id, external_key, name, content, state
		  FROM configs %s ORDER BY magistrala_thing LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, search, n+1, n+2)

	rows, err := cr.db.QueryContext(ctx, q, append(params, limit, offset)...)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}
	defer rows.Close()

	var name, content sql.NullString
	configs := []bootstrap.Config{}

	for rows.Next() {
		c := bootstrap.Config{DomainID: domainID}
		if err := rows.Scan(&c.ThingID, &c.ThingKey, &c.ExternalID, &c.ExternalKey, &name, &content, &c.State); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return bootstrap.ConfigsPage{}
		}

		c.Name = name.String
		c.Content = content.String
		configs = append(configs, c)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM configs %s`, search)

	var total uint64
	if err := cr.db.QueryRowxContext(ctx, q, params...).Scan(&total); err != nil {
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

func (cr configRepository) RetrieveByExternalID(ctx context.Context, externalID string) (bootstrap.Config, error) {
	q := `SELECT magistrala_thing, magistrala_key, external_key, domain_id, name, client_cert, client_key, ca_cert, content, state
		  FROM configs
		  WHERE external_id = :external_id`
	dbcfg := dbConfig{
		ExternalID: externalID,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	if ok := row.Next(); !ok {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q = `SELECT magistrala_channel, name, metadata FROM channels ch
     INNER JOIN connections conn
     ON ch.magistrala_channel = conn.channel_id AND ch.domain_id = conn.domain_id
     WHERE conn.config_id = :magistrala_thing AND conn.domain_id = :domain_id`

	rows, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	channels := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			cr.log.Error(fmt.Sprintf("Failed to deserialize channel due to %s", err))
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		channels = append(channels, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.Channels = channels

	return cfg, nil
}

func (cr configRepository) Update(ctx context.Context, cfg bootstrap.Config) error {
	q := `UPDATE configs SET name = :name, content = :content WHERE magistrala_thing = :magistrala_thing AND domain_id = :domain_id `

	dbcfg := dbConfig{
		Name:     nullString(cfg.Name),
		Content:  nullString(cfg.Content),
		ThingID:  cfg.ThingID,
		DomainID: cfg.DomainID,
	}

	res, err := cr.db.NamedExecContext(ctx, q, dbcfg)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (cr configRepository) UpdateCert(ctx context.Context, domainID, thingID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	q := `UPDATE configs SET client_cert = :client_cert, client_key = :client_key, ca_cert = :ca_cert WHERE magistrala_thing = :magistrala_thing AND domain_id = :domain_id
	RETURNING magistrala_thing, client_cert, client_key, ca_cert`

	dbcfg := dbConfig{
		ThingID:    thingID,
		ClientCert: nullString(clientCert),
		DomainID:   domainID,
		ClientKey:  nullString(clientKey),
		CaCert:     nullString(caCert),
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	if ok := row.Next(); !ok {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, err
	}

	return toConfig(dbcfg), nil
}

func (cr configRepository) UpdateConnections(ctx context.Context, domainID, id string, channels []bootstrap.Channel, connections []string) (err error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	defer func() {
		if err != nil {
			err = cr.rollback("UpdateConnections method", err, tx)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				err = commitErr
			}
		}
	}()

	if err = insertChannels(domainID, channels, tx); err != nil {
		err = errors.Wrap(repoerr.ErrUpdateEntity, err)
		return err
	}

	if err = updateConnections(domainID, id, connections, tx); err != nil {
		if e, ok := err.(*pgconn.PgError); ok {
			if e.Code == pgerrcode.ForeignKeyViolation {
				err = repoerr.ErrNotFound
			}
		}
		err = errors.Wrap(repoerr.ErrUpdateEntity, err)
		return err
	}

	return nil
}

func (cr configRepository) Remove(ctx context.Context, domainID, id string) error {
	q := `DELETE FROM configs WHERE magistrala_thing = :magistrala_thing AND domain_id = :domain_id`
	dbcfg := dbConfig{
		ThingID:  id,
		DomainID: domainID,
	}

	if _, err := cr.db.NamedExecContext(ctx, q, dbcfg); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	if _, err := cr.db.ExecContext(ctx, cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}

	return nil
}

func (cr configRepository) ChangeState(ctx context.Context, domainID, id string, state bootstrap.State) error {
	q := `UPDATE configs SET state = :state WHERE magistrala_thing = :magistrala_thing AND domain_id = :domain_id;`

	dbcfg := dbConfig{
		ThingID:  id,
		State:    state,
		DomainID: domainID,
	}

	res, err := cr.db.NamedExecContext(ctx, q, dbcfg)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (cr configRepository) ListExisting(ctx context.Context, domainID string, ids []string) ([]bootstrap.Channel, error) {
	var channels []bootstrap.Channel
	if len(ids) == 0 {
		return channels, nil
	}

	var chans pgtype.TextArray
	if err := chans.Set(ids); err != nil {
		return []bootstrap.Channel{}, err
	}

	q := "SELECT magistrala_channel, name, metadata FROM channels WHERE domain_id = $1 AND magistrala_channel = ANY ($2)"
	rows, err := cr.db.QueryxContext(ctx, q, domainID, chans)
	if err != nil {
		return []bootstrap.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	for rows.Next() {
		var dbch dbChannel
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channels due to %s", err))
			return []bootstrap.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
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

func (cr configRepository) RemoveThing(ctx context.Context, id string) error {
	q := `DELETE FROM configs WHERE magistrala_thing = $1`
	_, err := cr.db.ExecContext(ctx, q, id)

	if _, err := cr.db.ExecContext(ctx, cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr configRepository) UpdateChannel(ctx context.Context, c bootstrap.Channel) error {
	dbch, err := toDBChannel("", c)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	q := `UPDATE channels SET name = :name, metadata = :metadata, updated_at = :updated_at, updated_by = :updated_by
			WHERE magistrala_channel = :magistrala_channel`
	if _, err = cr.db.NamedExecContext(ctx, q, dbch); err != nil {
		return errors.Wrap(errUpdateChannels, err)
	}
	return nil
}

func (cr configRepository) RemoveChannel(ctx context.Context, id string) error {
	q := `DELETE FROM channels WHERE magistrala_channel = $1`
	if _, err := cr.db.ExecContext(ctx, q, id); err != nil {
		return errors.Wrap(errRemoveChannels, err)
	}
	return nil
}

func (cr configRepository) ConnectThing(ctx context.Context, channelID, thingID string) error {
	q := `UPDATE configs SET state = $1 WHERE EXISTS (
		SELECT 1 FROM connections WHERE config_id = $2 AND channel_id = $3)`
	result, err := cr.db.ExecContext(ctx, q, bootstrap.Active, thingID, channelID)
	if err != nil {
		return errors.Wrap(errConnectThing, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr configRepository) DisconnectThing(ctx context.Context, channelID, thingID string) error {
	q := `UPDATE configs SET state = $1 WHERE EXISTS (
		SELECT 1 FROM connections WHERE config_id = $2 AND channel_id = $3)`
	_, err := cr.db.ExecContext(ctx, q, bootstrap.Inactive, thingID, channelID)
	if err != nil {
		return errors.Wrap(errDisconnectThing, err)
	}
	return nil
}

func buildRetrieveQueryParams(domainID string, thingIDs []string, filter bootstrap.Filter) (string, []interface{}) {
	params := []interface{}{}
	queries := []string{}

	if len(thingIDs) != 0 {
		queries = append(queries, fmt.Sprintf("magistrala_thing IN ('%s')", strings.Join(thingIDs, "','")))
	} else if domainID != "" {
		params = append(params, domainID)
		queries = append(queries, fmt.Sprintf("domain_id = $%d", len(params)))
	}

	// Adjust the starting point for placeholders based on the current length of params
	counter := len(params) + 1
	for k, v := range filter.FullMatch {
		params = append(params, v)
		queries = append(queries, fmt.Sprintf("%s = $%d", k, counter))
		counter++
	}
	for k, v := range filter.PartialMatch {
		params = append(params, v)
		queries = append(queries, fmt.Sprintf("LOWER(%s) LIKE '%%' || $%d || '%%'", k, counter))
		counter++
	}

	if len(queries) > 0 {
		return "WHERE " + strings.Join(queries, " AND "), params
	}
	return "", params
}

func (cr configRepository) rollback(content string, defErr error, tx *sqlx.Tx) error {
	if err := tx.Rollback(); err != nil {
		return errors.Wrap(defErr, errors.Wrap(errors.New("failed to rollback at "+content), err))
	}

	return defErr
}

func insertChannels(domainID string, channels []bootstrap.Channel, tx *sqlx.Tx) error {
	if len(channels) == 0 {
		return nil
	}

	var chans []dbChannel
	for _, ch := range channels {
		dbch, err := toDBChannel(domainID, ch)
		if err != nil {
			return err
		}
		chans = append(chans, dbch)
	}
	q := `INSERT INTO channels (magistrala_channel, domain_id, name, metadata, parent_id, description, created_at, updated_at, updated_by, status)
		  VALUES (:magistrala_channel, :domain_id, :name, :metadata, :parent_id, :description, :created_at, :updated_at, :updated_by, :status)`
	if _, err := tx.NamedExec(q, chans); err != nil {
		e := err
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == pgerrcode.UniqueViolation {
			e = repoerr.ErrConflict
		}
		return e
	}

	return nil
}

func insertConnections(_ context.Context, cfg bootstrap.Config, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `INSERT INTO connections (config_id, channel_id, domain_id)
      VALUES (:config_id, :channel_id, :domain_id)`

	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:   cfg.ThingID,
			Channel:  conn,
			DomainID: cfg.DomainID,
		}
		conns = append(conns, dbconn)
	}
	_, err := tx.NamedExec(q, conns)

	return err
}

func updateConnections(domainID, id string, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `DELETE FROM connections
		  WHERE config_id = $1 AND domain_id = $2
		  AND channel_id NOT IN ($3)`

	var conn pgtype.TextArray
	if err := conn.Set(connections); err != nil {
		return err
	}

	res, err := tx.Exec(q, id, domainID, conn)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	q = `INSERT INTO connections (config_id, channel_id, domain_id)
		 VALUES (:config_id, :channel_id, :domain_id)`

	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:   id,
			Channel:  conn,
			DomainID: domainID,
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

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}

	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}

type dbConfig struct {
	ThingID     string          `db:"magistrala_thing"`
	DomainID    string          `db:"domain_id"`
	Name        sql.NullString  `db:"name"`
	ClientCert  sql.NullString  `db:"client_cert"`
	ClientKey   sql.NullString  `db:"client_key"`
	CaCert      sql.NullString  `db:"ca_cert"`
	ThingKey    string          `db:"magistrala_key"`
	ExternalID  string          `db:"external_id"`
	ExternalKey string          `db:"external_key"`
	Content     sql.NullString  `db:"content"`
	State       bootstrap.State `db:"state"`
}

func toDBConfig(cfg bootstrap.Config) dbConfig {
	return dbConfig{
		ThingID:     cfg.ThingID,
		DomainID:    cfg.DomainID,
		Name:        nullString(cfg.Name),
		ClientCert:  nullString(cfg.ClientCert),
		ClientKey:   nullString(cfg.ClientKey),
		CaCert:      nullString(cfg.CACert),
		ThingKey:    cfg.ThingKey,
		ExternalID:  cfg.ExternalID,
		ExternalKey: cfg.ExternalKey,
		Content:     nullString(cfg.Content),
		State:       cfg.State,
	}
}

func toConfig(dbcfg dbConfig) bootstrap.Config {
	cfg := bootstrap.Config{
		ThingID:     dbcfg.ThingID,
		DomainID:    dbcfg.DomainID,
		ThingKey:    dbcfg.ThingKey,
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
	ID          string         `db:"magistrala_channel"`
	Name        sql.NullString `db:"name"`
	DomainID    sql.NullString `db:"domain_id"`
	Metadata    string         `db:"metadata"`
	Parent      sql.NullString `db:"parent_id,omitempty"`
	Description string         `db:"description,omitempty"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at,omitempty"`
	UpdatedBy   sql.NullString `db:"updated_by,omitempty"`
	Status      clients.Status `db:"status"`
}

func toDBChannel(domainID string, ch bootstrap.Channel) (dbChannel, error) {
	dbch := dbChannel{
		ID:          ch.ID,
		Name:        nullString(ch.Name),
		DomainID:    nullString(domainID),
		Parent:      nullString(ch.Parent),
		Description: ch.Description,
		CreatedAt:   ch.CreatedAt,
		UpdatedAt:   nullTime(ch.UpdatedAt),
		UpdatedBy:   nullString(ch.UpdatedBy),
		Status:      ch.Status,
	}

	metadata, err := json.Marshal(ch.Metadata)
	if err != nil {
		return dbChannel{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	dbch.Metadata = string(metadata)
	return dbch, nil
}

func toChannel(dbch dbChannel) (bootstrap.Channel, error) {
	ch := bootstrap.Channel{
		ID:          dbch.ID,
		Description: dbch.Description,
		CreatedAt:   dbch.CreatedAt,
		Status:      dbch.Status,
	}

	if dbch.Name.Valid {
		ch.Name = dbch.Name.String
	}
	if dbch.DomainID.Valid {
		ch.DomainID = dbch.DomainID.String
	}
	if dbch.Parent.Valid {
		ch.Parent = dbch.Parent.String
	}
	if dbch.UpdatedBy.Valid {
		ch.UpdatedBy = dbch.UpdatedBy.String
	}
	if dbch.UpdatedAt.Valid {
		ch.UpdatedAt = dbch.UpdatedAt.Time
	}

	if err := json.Unmarshal([]byte(dbch.Metadata), &ch.Metadata); err != nil {
		return bootstrap.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return ch, nil
}

type dbConnection struct {
	Config   string `db:"config_id"`
	Channel  string `db:"channel_id"`
	DomainID string `db:"domain_id"`
}
