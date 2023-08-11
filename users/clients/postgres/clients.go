// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"

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
	Save(ctx context.Context, client mfclients.Client) (mfclients.Client, error)
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) Repository {
	return &clientRepo{
		ClientRepository: pgclients.ClientRepository{DB: db},
	}
}

func (repo clientRepo) Save(ctx context.Context, c mfclients.Client) (mfclients.Client, error) {
	q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, status, role)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :status, :role)
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at`
	dbc, err := pgclients.ToDBClient(c)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	row, err := repo.ClientRepository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return mfclients.Client{}, postgres.HandleError(err, errors.ErrCreateEntity)
	}

	defer row.Close()
	row.Next()
	dbc = pgclients.DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mfclients.Client{}, err
	}

	client, err := pgclients.ToClient(dbc)
	if err != nil {
		return mfclients.Client{}, err
	}

	return client, nil
}
