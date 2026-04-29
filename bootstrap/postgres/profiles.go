// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ bootstrap.ProfileRepository = (*profileRepository)(nil)

type profileRepository struct {
	db  postgres.Database
	log *slog.Logger
}

// NewProfileRepository instantiates a PostgreSQL implementation of ProfileRepository.
func NewProfileRepository(db postgres.Database, log *slog.Logger) bootstrap.ProfileRepository {
	return &profileRepository{db: db, log: log}
}

func (pr profileRepository) Save(ctx context.Context, p bootstrap.Profile) (bootstrap.Profile, error) {
	q := `INSERT INTO profiles (id, domain_id, name, description, template_format, content_template, defaults, version, created_at, updated_at)
		  VALUES (:id, :domain_id, :name, :description, :template_format, :content_template, :defaults, :version, :created_at, :updated_at)`

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	dbp, err := toDBProfile(p)
	if err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	if _, err = pr.db.NamedExecContext(ctx, q, dbp); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return bootstrap.Profile{}, repoerr.ErrConflict
		}
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return p, nil
}

func (pr profileRepository) RetrieveByID(ctx context.Context, domainID, id string) (bootstrap.Profile, error) {
	q := `SELECT id, domain_id, name, description, template_format, content_template, defaults, version, created_at, updated_at
		  FROM profiles WHERE id = $1 AND domain_id = $2`

	var dbp dbProfile
	if err := pr.db.QueryRowxContext(ctx, q, id, domainID).StructScan(&dbp); err != nil {
		if err == sql.ErrNoRows {
			return bootstrap.Profile{}, repoerr.ErrNotFound
		}
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return toProfile(dbp)
}

func (pr profileRepository) RetrieveAll(ctx context.Context, domainID string, offset, limit uint64) (bootstrap.ProfilesPage, error) {
	q := `SELECT id, domain_id, name, description, template_format, content_template, defaults, version, created_at, updated_at
		  FROM profiles WHERE domain_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := pr.db.QueryxContext(ctx, q, domainID, limit, offset)
	if err != nil {
		return bootstrap.ProfilesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var profiles []bootstrap.Profile
	for rows.Next() {
		var dbp dbProfile
		if err := rows.StructScan(&dbp); err != nil {
			pr.log.Error(fmt.Sprintf("failed to scan profile row: %s", err))
			return bootstrap.ProfilesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		p, err := toProfile(dbp)
		if err != nil {
			return bootstrap.ProfilesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		profiles = append(profiles, p)
	}

	var total uint64
	if err := pr.db.QueryRowxContext(ctx, `SELECT COUNT(*) FROM profiles WHERE domain_id = $1`, domainID).Scan(&total); err != nil {
		return bootstrap.ProfilesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return bootstrap.ProfilesPage{
		Total:    total,
		Offset:   offset,
		Limit:    limit,
		Profiles: profiles,
	}, nil
}

func (pr profileRepository) Update(ctx context.Context, p bootstrap.Profile) error {
	q := `UPDATE profiles SET name = :name, description = :description, template_format = :template_format,
		  content_template = :content_template, defaults = :defaults, version = version + 1, updated_at = :updated_at
		  WHERE id = :id AND domain_id = :domain_id`

	p.UpdatedAt = time.Now().UTC()
	dbp, err := toDBProfile(p)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	res, err := pr.db.NamedExecContext(ctx, q, dbp)
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

func (pr profileRepository) Delete(ctx context.Context, domainID, id string) error {
	q := `DELETE FROM profiles WHERE id = $1 AND domain_id = $2`
	if _, err := pr.db.ExecContext(ctx, q, id, domainID); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

// dbProfile is the database representation of a Profile.
type dbProfile struct {
	ID              string         `db:"id"`
	DomainID        string         `db:"domain_id"`
	Name            string         `db:"name"`
	Description     sql.NullString `db:"description"`
	TemplateFormat  string         `db:"template_format"`
	ContentTemplate sql.NullString `db:"content_template"`
	Defaults        []byte         `db:"defaults"`
	Version         int            `db:"version"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func toDBProfile(p bootstrap.Profile) (dbProfile, error) {
	defaults, err := json.Marshal(p.Defaults)
	if err != nil {
		return dbProfile{}, err
	}
	return dbProfile{
		ID:              p.ID,
		DomainID:        p.DomainID,
		Name:            p.Name,
		Description:     nullString(p.Description),
		TemplateFormat:  string(p.TemplateFormat),
		ContentTemplate: nullString(p.ContentTemplate),
		Defaults:        defaults,
		Version:         p.Version,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}, nil
}

func toProfile(dbp dbProfile) (bootstrap.Profile, error) {
	p := bootstrap.Profile{
		ID:             dbp.ID,
		DomainID:       dbp.DomainID,
		Name:           dbp.Name,
		TemplateFormat: bootstrap.TemplateFormat(dbp.TemplateFormat),
		Version:        dbp.Version,
		CreatedAt:      dbp.CreatedAt,
		UpdatedAt:      dbp.UpdatedAt,
	}
	if dbp.Description.Valid {
		p.Description = dbp.Description.String
	}
	if dbp.ContentTemplate.Valid {
		p.ContentTemplate = dbp.ContentTemplate.String
	}
	if len(dbp.Defaults) > 0 && string(dbp.Defaults) != "null" {
		if err := json.Unmarshal(dbp.Defaults, &p.Defaults); err != nil {
			return bootstrap.Profile{}, err
		}
	}
	return p, nil
}
