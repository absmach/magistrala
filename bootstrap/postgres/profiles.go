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
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
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
	q := `INSERT INTO profiles (id, domain_id, name, description, content_format, content_template, defaults, binding_slots, version, created_at, updated_at)
		  VALUES (:id, :domain_id, :name, :description, :content_format, :content_template, :defaults, :binding_slots, :version, :created_at, :updated_at)`

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	dbp, err := toDBProfile(p)
	if err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	if _, err = pr.db.NamedExecContext(ctx, q, dbp); err != nil {
		return bootstrap.Profile{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return p, nil
}

func (pr profileRepository) RetrieveByID(ctx context.Context, domainID, id string) (bootstrap.Profile, error) {
	q := `SELECT id, domain_id, name, description, content_format, content_template, defaults, binding_slots, version, created_at, updated_at
		  FROM profiles WHERE id = :id AND domain_id = :domain_id`

	rows, err := pr.db.NamedQueryContext(ctx, q, dbProfile{ID: id, DomainID: domainID})
	if err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return bootstrap.Profile{}, repoerr.ErrNotFound
	}
	var dbp dbProfile
	if err := rows.StructScan(&dbp); err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return toProfile(dbp)
}

func (pr profileRepository) RetrieveAll(ctx context.Context, domainID string, offset, limit uint64, name string) (bootstrap.ProfilesPage, error) {
	dbPage := dbProfilesPage{DomainID: domainID, Offset: offset, Limit: limit, Name: name}
	pageQuery := profilesPageQuery(dbPage)
	q := fmt.Sprintf(`SELECT id, domain_id, name, description, content_format, content_template, defaults, binding_slots, version, created_at, updated_at
		  FROM profiles %s`, pageQuery)
	q = applyProfilesOrdering(q)
	q = fmt.Sprintf(`%s LIMIT :limit OFFSET :offset`, q)

	rows, err := pr.db.NamedQueryContext(ctx, q, dbPage)
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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM profiles %s`, pageQuery)
	total, err := postgres.Total(ctx, pr.db, cq, dbPage)
	if err != nil {
		return bootstrap.ProfilesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return bootstrap.ProfilesPage{
		Total:    total,
		Offset:   offset,
		Limit:    limit,
		Profiles: profiles,
	}, nil
}

type dbProfilesPage struct {
	DomainID string `db:"domain_id"`
	Offset   uint64 `db:"offset"`
	Limit    uint64 `db:"limit"`
	Name     string `db:"name"`
}

func profilesPageQuery(pm dbProfilesPage) string {
	var query []string
	query = append(query, "domain_id = :domain_id")
	if pm.Name != "" {
		query = append(query, "name ILIKE '%' || :name || '%'")
	}
	return fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
}

func applyProfilesOrdering(q string) string {
	return fmt.Sprintf("%s ORDER BY created_at DESC", q)
}

func (pr profileRepository) Update(ctx context.Context, p bootstrap.Profile) (bootstrap.Profile, error) {
	var query []string
	var upq string
	if p.Name != "" {
		query = append(query, "name = :name,")
	}
	if p.Description != "" {
		query = append(query, "description = :description,")
	}
	if p.ContentFormat != "" {
		query = append(query, "content_format = :content_format,")
	}
	if p.ContentTemplate != "" {
		query = append(query, "content_template = :content_template,")
	}
	if p.Defaults != nil {
		query = append(query, "defaults = :defaults,")
	}
	if p.BindingSlots != nil {
		query = append(query, "binding_slots = :binding_slots,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE profiles SET %s version = version + 1, updated_at = :updated_at
		  WHERE id = :id AND domain_id = :domain_id
		  RETURNING id, domain_id, name, description, content_format, content_template, defaults, binding_slots, version, created_at, updated_at`,
		upq)

	p.UpdatedAt = time.Now().UTC()
	dbp, err := toDBProfile(p)
	if err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, dbp)
	if err != nil {
		return bootstrap.Profile{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return bootstrap.Profile{}, repoerr.ErrNotFound
	}
	var updated dbProfile
	if err := rows.StructScan(&updated); err != nil {
		return bootstrap.Profile{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return toProfile(updated)
}

func (pr profileRepository) Delete(ctx context.Context, domainID, id string) error {
	q := `DELETE FROM profiles WHERE id = :id AND domain_id = :domain_id`
	if _, err := pr.db.NamedExecContext(ctx, q, dbProfile{ID: id, DomainID: domainID}); err != nil {
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
	ContentFormat   string         `db:"content_format"`
	ContentTemplate sql.NullString `db:"content_template"`
	Defaults        []byte         `db:"defaults"`
	BindingSlots    []byte         `db:"binding_slots"`
	Version         int            `db:"version"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func toDBProfile(p bootstrap.Profile) (dbProfile, error) {
	defaults, err := json.Marshal(p.Defaults)
	if err != nil {
		return dbProfile{}, err
	}
	bindingSlots, err := json.Marshal(p.BindingSlots)
	if err != nil {
		return dbProfile{}, err
	}
	return dbProfile{
		ID:              p.ID,
		DomainID:        p.DomainID,
		Name:            p.Name,
		Description:     nullString(p.Description),
		ContentFormat:   string(p.ContentFormat),
		ContentTemplate: nullString(p.ContentTemplate),
		Defaults:        defaults,
		BindingSlots:    bindingSlots,
		Version:         p.Version,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}, nil
}

func toProfile(dbp dbProfile) (bootstrap.Profile, error) {
	p := bootstrap.Profile{
		ID:            dbp.ID,
		DomainID:      dbp.DomainID,
		Name:          dbp.Name,
		ContentFormat: bootstrap.ContentFormat(dbp.ContentFormat),
		Version:       dbp.Version,
		CreatedAt:     dbp.CreatedAt,
		UpdatedAt:     dbp.UpdatedAt,
	}
	if dbp.Description.Valid {
		p.Description = dbp.Description.String
	}
	if dbp.ContentTemplate.Valid {
		p.ContentTemplate = dbp.ContentTemplate.String
	}
	if len(dbp.Defaults) > 0 && string(dbp.Defaults) != jsonNull {
		if err := json.Unmarshal(dbp.Defaults, &p.Defaults); err != nil {
			return bootstrap.Profile{}, err
		}
	}
	if len(dbp.BindingSlots) > 0 && string(dbp.BindingSlots) != jsonNull {
		if err := json.Unmarshal(dbp.BindingSlots, &p.BindingSlots); err != nil {
			return bootstrap.Profile{}, err
		}
	}
	return p, nil
}
