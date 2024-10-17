// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/internal/api"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/things"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ mgclients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	pgclients.Repository
}

// Repository is the interface that wraps the basic methods for
// a client repository.
//
//go:generate mockery --name Repository --output=../mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	mgclients.Repository

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client ...mgclients.Client) ([]mgclients.Client, error)

	// RetrieveBySecret retrieves a client based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error)
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) Repository {
	return &clientRepo{
		Repository: pgclients.Repository{DB: db},
	}
}

func (repo clientRepo) Save(ctx context.Context, cs ...mgclients.Client) ([]mgclients.Client, error) {
	tx, err := repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return []mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	var clients []mgclients.Client

	for _, cli := range cs {
		q := `INSERT INTO clients (id, name, tags, domain_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :domain_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags, identity, secret, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`

		dbcli, err := pgclients.ToDBClient(cli)
		if err != nil {
			return []mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		row, err := repo.DB.NamedQueryContext(ctx, q, dbcli)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return []mgclients.Client{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
			return []mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		defer row.Close()

		if row.Next() {
			dbcli = pgclients.DBClient{}
			if err := row.StructScan(&dbcli); err != nil {
				return []mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}

			client, err := pgclients.ToClient(dbcli)
			if err != nil {
				return []mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}
			clients = append(clients, client)
		}
	}
	if err = tx.Commit(); err != nil {
		return []mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return clients, nil
}

func (repo clientRepo) RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = :secret AND status = %d`, mgclients.EnabledStatus)

	dbc := pgclients.DBClient{
		Secret: key,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return mgclients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = pgclients.DBClient{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return mgclients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		client, err := pgclients.ToClient(dbc)
		if err != nil {
			return mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return client, nil
	}

	return mgclients.Client{}, repoerr.ErrNotFound
}

type DBThing struct {
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Identity  string           `db:"identity"`
	Domain    string           `db:"domain_id"`
	Secret    string           `db:"secret"`
	Metadata  []byte           `db:"metadata,omitempty"`
	CreatedAt time.Time        `db:"created_at,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    things.Status    `db:"status,omitempty"`
}

func ToDBThing(c things.Thing) (DBThing, error) {
	data := []byte("{}")
	if len(c.Metadata) > 0 {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return DBThing{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(c.Tags); err != nil {
		return DBThing{}, err
	}
	var updatedBy *string
	if c.UpdatedBy != "" {
		updatedBy = &c.UpdatedBy
	}
	var updatedAt sql.NullTime
	if c.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: c.UpdatedAt, Valid: true}
	}

	return DBThing{
		ID:        c.ID,
		Name:      c.Name,
		Tags:      tags,
		Domain:    c.Domain,
		Identity:  c.Credentials.Identity,
		Secret:    c.Credentials.Secret,
		Metadata:  data,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
	}, nil
}

func ToThing(t DBThing) (things.Thing, error) {
	var metadata things.Metadata
	if t.Metadata != nil {
		if err := json.Unmarshal([]byte(t.Metadata), &metadata); err != nil {
			return things.Thing{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range t.Tags.Elements {
		tags = append(tags, e.String)
	}
	var updatedBy string
	if t.UpdatedBy != nil {
		updatedBy = *t.UpdatedBy
	}
	var updatedAt time.Time
	if t.UpdatedAt.Valid {
		updatedAt = t.UpdatedAt.Time
	}

	thg := things.Thing{
		ID:     t.ID,
		Name:   t.Name,
		Tags:   tags,
		Domain: t.Domain,
		Credentials: things.Credentials{
			Identity: t.Identity,
			Secret:   t.Secret,
		},
		Metadata:  metadata,
		CreatedAt: t.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    t.Status,
	}
	return thg, nil
}

func ToDBThingsPage(pm things.Page) (dbThingsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbThingsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbThingsPage{
		Name:     pm.Name,
		Identity: pm.Identity,
		Id:       pm.Id,
		Metadata: data,
		Domain:   pm.Domain,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		Status:   pm.Status,
		Tag:      pm.Tag,
	}, nil
}

type dbThingsPage struct {
	Total    uint64        `db:"total"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
	Name     string        `db:"name"`
	Id       string        `db:"id"`
	Domain   string        `db:"domain_id"`
	Identity string        `db:"identity"`
	Metadata []byte        `db:"metadata"`
	Tag      string        `db:"tag"`
	Status   things.Status `db:"status"`
	GroupID  string        `db:"group_id"`
}

func PageQuery(pm things.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var query []string
	if pm.Name != "" {
		query = append(query, "name ILIKE '%' || :name || '%'")
	}
	if pm.Identity != "" {
		query = append(query, "identity ILIKE '%' || :identity || '%'")
	}
	if pm.Id != "" {
		query = append(query, "id ILIKE '%' || :id || '%'")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}
	// If there are search params presents, use search and ignore other options.
	// Always combine role with search params, so len(query) > 1.
	if len(query) > 1 {
		return fmt.Sprintf("WHERE %s", strings.Join(query, " AND ")), nil
	}

	if mq != "" {
		query = append(query, mq)
	}

	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("id IN ('%s')", strings.Join(pm.IDs, "','")))
	}
	if pm.Status != things.AllStatus {
		query = append(query, "c.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "c.domain_id = :domain_id")
	}
	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}
	return emq, nil
}

func applyOrdering(emq string, pm things.Page) string {
	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}
