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
	q := `INSERT INTO configs (id, workspace_id, name, client_cert, client_key, ca_cert, external_id, external_key, bootstrap_key_version, content, status, profile_id, render_context)
	VALUES (:id, :workspace_id, :name, :client_cert, :client_key, :ca_cert, :external_id, :external_key, :bootstrap_key_version, :content, :status, :profile_id, :render_context)`

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

func (cr configRepository) RetrieveByID(ctx context.Context, workspaceID, id string) (bootstrap.Config, error) {
	q := `SELECT id, workspace_id, external_id, external_key, bootstrap_key_version, name, content, status, client_cert, client_key, ca_cert, profile_id, render_context
		  FROM configs
		  WHERE id = :id AND workspace_id = :workspace_id`

	dbcfg := dbConfig{
		ID:          id,
		WorkspaceID: workspaceID,
	}
	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return bootstrap.Config{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, err
	}

	cfg, err := toConfig(dbcfg)
	if err != nil {
		return bootstrap.Config{}, err
	}
	return cfg, nil
}

func (cr configRepository) RetrieveAll(ctx context.Context, workspaceID string, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	search, params, err := buildRetrieveQueryParams(workspaceID, filter)
	if err != nil {
		return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	n := len(params)

	// workspace_id is selected rather than stamped from the argument: an
	// empty workspaceID means "every workspace" (used by startup Atom
	// reconciliation), and each row must carry its own tenant so that the
	// external key decrypts against the right associated data.
	q := `SELECT id, workspace_id, external_id, external_key, bootstrap_key_version, name, content, status, profile_id, render_context
		  FROM configs %s ORDER BY id LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, search, n+1, n+2)

	rows, err := cr.db.QueryContext(ctx, q, append(params, limit, offset)...)
	if err != nil {
		return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var name, content, profileID sql.NullString
	var renderContext []byte
	configs := []bootstrap.Config{}

	for rows.Next() {
		c := bootstrap.Config{}
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.ExternalID, &c.ExternalKey, &c.BootstrapKeyVersion, &name, &content, &c.Status, &profileID, &renderContext); err != nil {
			return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c.Name = name.String
		c.Content = content.String
		if profileID.Valid {
			c.ProfileID = profileID.String
		}
		if len(renderContext) > 0 && string(renderContext) != jsonNull {
			if err := json.Unmarshal(renderContext, &c.RenderContext); err != nil {
				return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
			}
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM configs %s`, search)

	var total uint64
	if err := cr.db.QueryRowxContext(ctx, q, params...).Scan(&total); err != nil {
		return bootstrap.ConfigsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return bootstrap.ConfigsPage{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Configs: configs,
	}, nil
}

func (cr configRepository) RetrieveByExternalID(ctx context.Context, externalID string) (bootstrap.Config, error) {
	q := `SELECT id, external_key, bootstrap_key_version, workspace_id, name, client_cert, client_key, ca_cert, content, status, profile_id, render_context
		  FROM configs
		  WHERE external_id = :external_id`
	dbcfg := dbConfig{
		ExternalID: externalID,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbcfg)
	if err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return bootstrap.Config{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbcfg); err != nil {
		return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	cfg, err := toConfig(dbcfg)
	if err != nil {
		return bootstrap.Config{}, err
	}
	return cfg, nil
}

func (cr configRepository) Update(ctx context.Context, cfg bootstrap.Config) error {
	// Update backs a PATCH, so only the columns the caller actually supplied
	// are written. Assigning every column unconditionally would let a request
	// carrying just {"name": ...} clear the content and render_context the
	// device template depends on.
	var set []string
	if cfg.Name != "" {
		set = append(set, "name = :name")
	}
	if cfg.Content != "" {
		set = append(set, "content = :content")
	}
	var renderContext []byte
	if cfg.RenderContext != nil {
		marshalled, err := json.Marshal(cfg.RenderContext)
		if err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		renderContext = marshalled
		set = append(set, "render_context = :render_context")
	}
	// A new external key is rotated in together with its version bump.
	if cfg.ExternalKey != "" {
		set = append(set, "external_key = :external_key")
		set = append(set, "bootstrap_key_version = bootstrap_key_version + 1")
	}

	// Nothing to change: still verify the config exists in this workspace so
	// the caller gets ErrNotFound rather than a silent success.
	if len(set) == 0 {
		_, err := cr.RetrieveByID(ctx, cfg.WorkspaceID, cfg.ID)
		return err
	}

	q := fmt.Sprintf(`UPDATE configs SET %s WHERE id = :id AND workspace_id = :workspace_id`,
		strings.Join(set, ", "))

	dbcfg := dbConfig{
		Name:          nullString(cfg.Name),
		ExternalKey:   cfg.ExternalKey,
		Content:       nullString(cfg.Content),
		RenderContext: renderContext,
		ID:            cfg.ID,
		WorkspaceID:   cfg.WorkspaceID,
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

func (cr configRepository) AssignProfile(ctx context.Context, workspaceID, id, profileID string) error {
	q := `UPDATE configs SET profile_id = :profile_id WHERE id = :id AND workspace_id = :workspace_id`

	dbcfg := dbConfig{
		ID:          id,
		WorkspaceID: workspaceID,
		ProfileID:   nullString(profileID),
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

func (cr configRepository) UpdateCert(ctx context.Context, workspaceID, id, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	q := `UPDATE configs SET client_cert = :client_cert, client_key = :client_key, ca_cert = :ca_cert WHERE id = :id AND workspace_id = :workspace_id
	RETURNING id, client_cert, client_key, ca_cert, workspace_id`

	dbcfg := dbConfig{
		ID:          id,
		ClientCert:  nullString(clientCert),
		WorkspaceID: workspaceID,
		ClientKey:   nullString(clientKey),
		CaCert:      nullString(caCert),
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

	cfg, err := toConfig(dbcfg)
	if err != nil {
		return bootstrap.Config{}, err
	}
	return cfg, nil
}

func (cr configRepository) Remove(ctx context.Context, workspaceID, id string) error {
	q := `DELETE FROM configs WHERE id = :id AND workspace_id = :workspace_id`
	dbcfg := dbConfig{
		ID:          id,
		WorkspaceID: workspaceID,
	}

	res, err := cr.db.NamedExecContext(ctx, q, dbcfg)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	// Matching no row means the config does not exist in this workspace.
	// Reporting success would both answer 204 for a foreign ID and let the
	// caller tear down an Atom projection whose config is still live.
	if cnt == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (cr configRepository) ChangeStatus(ctx context.Context, workspaceID, id string, status bootstrap.Status) error {
	q := `UPDATE configs SET status = :status WHERE id = :id AND workspace_id = :workspace_id;`

	dbcfg := dbConfig{
		ID:          id,
		Status:      status,
		WorkspaceID: workspaceID,
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

// buildRetrieveQueryParams renders the WHERE clause for a config listing. An
// unparseable filter is reported as an error rather than degrading into an
// unfiltered query: dropping the predicates would also drop the workspace_id
// scope and leak enrollments across tenants.
func buildRetrieveQueryParams(workspaceID string, filter bootstrap.Filter) (string, []any, error) {
	params := []any{}
	queries := []string{}

	if workspaceID != "" {
		params = append(params, workspaceID)
		queries = append(queries, fmt.Sprintf("workspace_id = $%d", len(params)))
	}

	counter := len(params) + 1
	for k, v := range filter.FullMatch {
		if k == "status" {
			status, err := bootstrap.ToStatus(v)
			if err != nil {
				return "", nil, err
			}
			if status == bootstrap.AllStatus {
				continue
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
		return "WHERE " + strings.Join(queries, " AND "), params, nil
	}
	return "", params, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type dbConfig struct {
	WorkspaceID         string           `db:"workspace_id"`
	ID                  string           `db:"id"`
	Name                sql.NullString   `db:"name"`
	ClientCert          sql.NullString   `db:"client_cert"`
	ClientKey           sql.NullString   `db:"client_key"`
	CaCert              sql.NullString   `db:"ca_cert"`
	ExternalID          string           `db:"external_id"`
	ExternalKey         string           `db:"external_key"`
	Content             sql.NullString   `db:"content"`
	Status              bootstrap.Status `db:"status"`
	ProfileID           sql.NullString   `db:"profile_id"`
	RenderContext       []byte           `db:"render_context"`
	BootstrapKeyVersion uint64           `db:"bootstrap_key_version"`
}

func toDBConfig(cfg bootstrap.Config) (dbConfig, error) {
	renderContext, err := json.Marshal(cfg.RenderContext)
	if err != nil {
		return dbConfig{}, err
	}

	return dbConfig{
		ID:                  cfg.ID,
		WorkspaceID:         cfg.WorkspaceID,
		Name:                nullString(cfg.Name),
		ClientCert:          nullString(cfg.ClientCert),
		ClientKey:           nullString(cfg.ClientKey),
		CaCert:              nullString(cfg.CACert),
		ExternalID:          cfg.ExternalID,
		ExternalKey:         cfg.ExternalKey,
		Content:             nullString(cfg.Content),
		Status:              cfg.Status,
		ProfileID:           nullString(cfg.ProfileID),
		RenderContext:       renderContext,
		BootstrapKeyVersion: cfg.BootstrapKeyVersion,
	}, nil
}

func toConfig(dbcfg dbConfig) (bootstrap.Config, error) {
	cfg := bootstrap.Config{
		ID:                  dbcfg.ID,
		WorkspaceID:         dbcfg.WorkspaceID,
		ExternalID:          dbcfg.ExternalID,
		ExternalKey:         dbcfg.ExternalKey,
		Status:              dbcfg.Status,
		BootstrapKeyVersion: dbcfg.BootstrapKeyVersion,
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
		if err := json.Unmarshal(dbcfg.RenderContext, &cfg.RenderContext); err != nil {
			return bootstrap.Config{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
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
	return cfg, nil
}
