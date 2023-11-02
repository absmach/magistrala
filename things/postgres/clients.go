// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/absmach/magistrala/internal/postgres"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
)

var _ mgclients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	pgclients.ClientRepository
}

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
		ClientRepository: pgclients.ClientRepository{DB: db},
	}
}

func (repo clientRepo) Save(ctx context.Context, cs ...mgclients.Client) ([]mgclients.Client, error) {
	tx, err := repo.ClientRepository.DB.BeginTxx(ctx, nil)
	if err != nil {
		return []mgclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	var clients []mgclients.Client

	for _, cli := range cs {
		q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags, identity, secret, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

		dbcli, err := pgclients.ToDBClient(cli)
		if err != nil {
			return []mgclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		row, err := repo.ClientRepository.DB.NamedQueryContext(ctx, q, dbcli)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return []mgclients.Client{}, postgres.HandleError(err, errors.ErrCreateEntity)
			}
			return []mgclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		defer row.Close()
		row.Next()
		dbcli = pgclients.DBClient{}
		if err := row.StructScan(&dbcli); err != nil {
			return []mgclients.Client{}, err
		}

		client, err := pgclients.ToClient(dbcli)
		if err != nil {
			return []mgclients.Client{}, err
		}
		clients = append(clients, client)
	}
	if err = tx.Commit(); err != nil {
		return []mgclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return clients, nil
}

func (repo clientRepo) RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = $1 AND status = %d`, mgclients.EnabledStatus)

	dbc := pgclients.DBClient{
		Secret: key,
	}

	if err := repo.DB.QueryRowxContext(ctx, q, key).StructScan(&dbc); err != nil {
		if err == sql.ErrNoRows {
			return mgclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return mgclients.Client{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	return pgclients.ToClient(dbc)
}
