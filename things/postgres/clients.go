// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/internal/postgres"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
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

	// Delete deletes client with given id
	Delete(ctx context.Context, id string) error
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
		q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags, identity, secret, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

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
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
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

func (repo clientRepo) Delete(ctx context.Context, id string) error {
	q := "DELETE FROM clients AS c  WHERE c.id = $1 ;"

	result, err := repo.DB.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}
