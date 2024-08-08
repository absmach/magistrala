// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
)

var _ auth.TokenRepository = (*tokenRepo)(nil)

type tokenRepo struct {
	db postgres.Database
}

// NewTokensRepository instantiates a PostgreSQL implementation of tokens repository.
func NewTokensRepository(db postgres.Database) auth.TokenRepository {
	return &tokenRepo{
		db: db,
	}
}

func (repo *tokenRepo) Save(ctx context.Context, id string) error {
	q := `INSERT INTO tokens (id) VALUES ($1);`

	result, err := repo.db.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	if rows, err := result.RowsAffected(); rows == 0 {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *tokenRepo) Contains(ctx context.Context, id string) bool {
	q := `SELECT * FROM tokens WHERE id = $1;`

	rows, err := repo.db.QueryContext(ctx, q, id)
	if err != nil {
		return false
	}
	defer rows.Close()

	if rows.Next() {
		id := ""
		if err = rows.Scan(&id); err != nil {
			return false
		}

		return true
	}

	return false
}
