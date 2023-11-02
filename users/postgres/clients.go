// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"

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
	Save(ctx context.Context, client mgclients.Client) (mgclients.Client, error)

	IsOwner(ctx context.Context, clientID string, ownerID string) error
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) Repository {
	return &clientRepo{
		ClientRepository: pgclients.ClientRepository{DB: db},
	}
}

func (repo clientRepo) Save(ctx context.Context, c mgclients.Client) (mgclients.Client, error) {
	q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, status, role)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :status, :role)
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at`
	dbc, err := pgclients.ToDBClient(c)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	row, err := repo.ClientRepository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return mgclients.Client{}, postgres.HandleError(err, errors.ErrCreateEntity)
	}

	defer row.Close()
	row.Next()
	dbc = pgclients.DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mgclients.Client{}, err
	}

	client, err := pgclients.ToClient(dbc)
	if err != nil {
		return mgclients.Client{}, err
	}

	return client, nil
}

func (repo clientRepo) IsOwner(ctx context.Context, clientID, ownerID string) error {
	q := "SELECT * FROM clients WHERE id = $1 AND owner_id = $2"

	rows, err := repo.ClientRepository.DB.QueryContext(ctx, q, clientID, ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.ErrAuthorization
		}
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.ErrAuthorization
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	return nil
}
