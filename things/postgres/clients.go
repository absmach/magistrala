// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mainflux/mainflux/internal/postgres"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	pgclients "github.com/mainflux/mainflux/pkg/clients/postgres"
	"github.com/mainflux/mainflux/pkg/errors"
)

var _ mfclients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	pgclients.ClientRepository
}

type Repository interface {
	mfclients.Repository

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client ...mfclients.Client) ([]mfclients.Client, error)

	// RetrieveBySecret retrieves a client based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (mfclients.Client, error)
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) Repository {
	return &clientRepo{
		ClientRepository: pgclients.ClientRepository{DB: db},
	}
}

func (repo clientRepo) Save(ctx context.Context, cs ...mfclients.Client) ([]mfclients.Client, error) {
	tx, err := repo.ClientRepository.DB.BeginTxx(ctx, nil)
	if err != nil {
		return []mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}
	var clients []mfclients.Client

	for _, cli := range cs {
		q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags, identity, secret, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

		dbcli, err := pgclients.ToDBClient(cli)
		if err != nil {
			return []mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		row, err := repo.ClientRepository.DB.NamedQueryContext(ctx, q, dbcli)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return []mfclients.Client{}, postgres.HandleError(err, errors.ErrCreateEntity)
			}
			return []mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
		}

		defer row.Close()
		row.Next()
		dbcli = pgclients.DBClient{}
		if err := row.StructScan(&dbcli); err != nil {
			return []mfclients.Client{}, err
		}

		client, err := pgclients.ToClient(dbcli)
		if err != nil {
			return []mfclients.Client{}, err
		}
		clients = append(clients, client)
	}
	if err = tx.Commit(); err != nil {
		return []mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return clients, nil
}

func (repo clientRepo) RetrieveBySecret(ctx context.Context, key string) (mfclients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = $1 AND status = %d`, mfclients.EnabledStatus)

	dbc := pgclients.DBClient{
		Secret: key,
	}

	if err := repo.DB.QueryRowxContext(ctx, q, key).StructScan(&dbc); err != nil {
		if err == sql.ErrNoRows {
			return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return mfclients.Client{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	return pgclients.ToClient(dbc)
}
