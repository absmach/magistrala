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

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgconn"
)

const jsonNull = "null"

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

func (cr configRepository) Save(ctx context.Context, cfg bootstrap.Config) (string, error) {
	q := `INSERT INTO configs (id, domain_id, name, client_cert, client_key, ca_cert, external_id, external_key, content, status, profile_id, render_context)
	VALUES (:id, :domain_id, :name, :client_cert, :client_key, :ca_cert, :external_id, :external_key, :content, :status, :profile_id, :render_context)`

	dbcfg, err := toDBConfig(cfg)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	if _, err := cr.db.NamedExecContext(ctx, q, dbcfg); err != nil {
		switch pgErr := err.(type) {
		case *pgconn.PgError:
			if pgErr.Code == pgerrcode.UniqueViolation {
				return "", repoerr.ErrConflict
			}
		}
		return "", errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return cfg.ID, nil
}

func (cr configRepository) RetrieveByID(ctx context.Context, domainID, id string) (bootstrap.Config, error) {
	q := `SELECT id, external_id, name, content, status, client_cert, client_key, ca_cert, profile_id, render_context
		  FROM configs
		  WHERE id = :id AND domain_id = :domain_id`

	dbcfg := dbConfig{
		ID:       id,
		DomainID: domainID,
	}
	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	if !row.Next() {
		return bootstrap.Config{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, err
	}

	return toConfig(dbcfg), nil
}

func (cr configRepository) RetrieveAll(ctx context.Context, domainID string, clientIDs []string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	search, params := buildRetrieveQueryParams(domainID, clientIDs, filter)
	n := len(params)

	q := `SELECT id, external_id, name, content, status, profile_id, render_context
		  FROM configs %s ORDER BY id LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, search, n+1, n+2)

	rows, err := cr.db.QueryContext(ctx, q, append(params, limit, offset)...)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}
	defer rows.Close()

	var name, content, profileID sql.NullString
	var renderContext []byte
	configs := []bootstrap.Config{}

	for rows.Next() {
		c := bootstrap.Config{DomainID: domainID}
		if err := rows.Scan(&c.ID, &c.ExternalID, &name, &content, &c.Status, &profileID, &renderContext); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return bootstrap.ConfigsPage{}
		}

		c.Name = name.String
		c.Content = content.String
		if profileID.Valid {
			c.ProfileID = profileID.String
		}
		if len(renderContext) > 0 && string(renderContext) != jsonNull {
			if err := json.Unmarshal(renderContext, &c.RenderContext); err != nil {
				cr.log.Error(fmt.Sprintf("Failed to decode render context due to %s", err))
				return bootstrap.ConfigsPage{}
			}
		}
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
	q := `SELECT id, external_key, domain_id, name, client_cert, client_key, ca_cert, content, status, profile_id, render_context
		  FROM configs
		  WHERE external_id = :external_id`
	dbcfg := dbConfig{
		ExternalID: externalID,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	if !row.Next() {
		return bootstrap.Config{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return toConfig(dbcfg), nil
}

func (cr configRepository) Update(ctx context.Context, cfg bootstrap.Config) error {
	q := `UPDATE configs SET name = :name, content = :content WHERE id = :id AND domain_id = :domain_id `

	dbcfg := dbConfig{
		Name:     nullString(cfg.Name),
		Content:  nullString(cfg.Content),
		ID:       cfg.ID,
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

func (cr configRepository) AssignProfile(ctx context.Context, domainID, id, profileID string) error {
	q := `UPDATE configs SET profile_id = :profile_id WHERE id = :id AND domain_id = :domain_id`

	dbcfg := dbConfig{
		ID:        id,
		DomainID:  domainID,
		ProfileID: nullString(profileID),
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

func (cr configRepository) UpdateCert(ctx context.Context, domainID, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	q := `UPDATE configs SET client_cert = :client_cert, client_key = :client_key, ca_cert = :ca_cert WHERE id = :id AND domain_id = :domain_id
	RETURNING id, client_cert, client_key, ca_cert, domain_id`

	dbcfg := dbConfig{
		ID:         clientID,
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

func (cr configRepository) Remove(ctx context.Context, domainID, id string) error {
	q := `DELETE FROM configs WHERE id = :id AND domain_id = :domain_id`
	dbcfg := dbConfig{
		ID:       id,
		DomainID: domainID,
	}

	if _, err := cr.db.NamedExecContext(ctx, q, dbcfg); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (cr configRepository) ChangeStatus(ctx context.Context, domainID, id string, status bootstrap.Status) error {
	q := `UPDATE configs SET status = :status WHERE id = :id AND domain_id = :domain_id;`

	dbcfg := dbConfig{
		ID:       id,
		Status:   status,
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

func (cr configRepository) RemoveClient(ctx context.Context, id string) error {
	q := `DELETE FROM configs WHERE id = $1`
	_, err := cr.db.ExecContext(ctx, q, id)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func buildRetrieveQueryParams(domainID string, clientIDs []string, filter bootstrap.Filter) (string, []any) {
	params := []any{}
	queries := []string{}

	if len(clientIDs) != 0 {
		var arr pgtype.TextArray
		if err := arr.Set(clientIDs); err != nil {
			return "", nil
		}
		params = append(params, arr)
		queries = append(queries, fmt.Sprintf("id = ANY($%d)", len(params)))
	} else if domainID != "" {
		params = append(params, domainID)
		queries = append(queries, fmt.Sprintf("domain_id = $%d", len(params)))
	}

	counter := len(params) + 1
	for k, v := range filter.FullMatch {
		if k == "status" {
			status, err := bootstrap.ToStatus(v)
			if err != nil {
				return "", nil
			}
			params = append(params, status)
			queries = append(queries, fmt.Sprintf("%s = $%d", k, counter))
			counter++
			continue
		}
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

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type dbConfig struct {
	DomainID      string           `db:"domain_id"`
	ID            string           `db:"id"`
	Name          sql.NullString   `db:"name"`
	ClientCert    sql.NullString   `db:"client_cert"`
	ClientKey     sql.NullString   `db:"client_key"`
	CaCert        sql.NullString   `db:"ca_cert"`
	ExternalID    string           `db:"external_id"`
	ExternalKey   string           `db:"external_key"`
	Content       sql.NullString   `db:"content"`
	Status        bootstrap.Status `db:"status"`
	ProfileID     sql.NullString   `db:"profile_id"`
	RenderContext []byte           `db:"render_context"`
}

func toDBConfig(cfg bootstrap.Config) (dbConfig, error) {
	renderContext, err := json.Marshal(cfg.RenderContext)
	if err != nil {
		return dbConfig{}, err
	}

	return dbConfig{
		ID:            cfg.ID,
		DomainID:      cfg.DomainID,
		Name:          nullString(cfg.Name),
		ClientCert:    nullString(cfg.ClientCert),
		ClientKey:     nullString(cfg.ClientKey),
		CaCert:        nullString(cfg.CACert),
		ExternalID:    cfg.ExternalID,
		ExternalKey:   cfg.ExternalKey,
		Content:       nullString(cfg.Content),
		Status:        cfg.Status,
		ProfileID:     nullString(cfg.ProfileID),
		RenderContext: renderContext,
	}, nil
}

func toConfig(dbcfg dbConfig) bootstrap.Config {
	cfg := bootstrap.Config{
		ID:          dbcfg.ID,
		DomainID:    dbcfg.DomainID,
		ExternalID:  dbcfg.ExternalID,
		ExternalKey: dbcfg.ExternalKey,
		Status:      dbcfg.Status,
	}
	if dbcfg.ProfileID.Valid {
		cfg.ProfileID = dbcfg.ProfileID.String
	}

	if dbcfg.Name.Valid {
		cfg.Name = dbcfg.Name.String
	}
	if dbcfg.Content.Valid {
		cfg.Content = dbcfg.Content.String
	}
	if len(dbcfg.RenderContext) > 0 && string(dbcfg.RenderContext) != jsonNull {
		_ = json.Unmarshal(dbcfg.RenderContext, &cfg.RenderContext)
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
