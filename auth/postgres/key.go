// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

var (
	errSave     = errors.New("failed to save key in database")
	errRetrieve = errors.New("failed to retrieve key from database")
	errDelete   = errors.New("failed to delete key from database")
)
var _ auth.KeyRepository = (*repo)(nil)

type repo struct {
	db postgres.Database
}

// New instantiates a PostgreSQL implementation of key repository.
func New(db postgres.Database) auth.KeyRepository {
	return &repo{
		db: db,
	}
}

func (kr *repo) Save(ctx context.Context, key auth.Key) (string, error) {
	q := `INSERT INTO keys (id, type, issuer_id, subject, issued_at, expires_at)
	      VALUES (:id, :type, :issuer_id, :subject, :issued_at, :expires_at)`

	dbKey := toDBKey(key)
	if _, err := kr.db.NamedExecContext(ctx, q, dbKey); err != nil {
		return "", postgres.HandleError(errSave, err)
	}

	return dbKey.ID, nil
}

func (kr *repo) Retrieve(ctx context.Context, issuerID, id string) (auth.Key, error) {
	q := `SELECT id, type, issuer_id, subject, issued_at, expires_at FROM keys WHERE issuer_id = $1 AND id = $2`
	key := dbKey{}
	if err := kr.db.QueryRowxContext(ctx, q, issuerID, id).StructScan(&key); err != nil {
		if err == sql.ErrNoRows {
			return auth.Key{}, repoerr.ErrNotFound
		}

		return auth.Key{}, postgres.HandleError(errRetrieve, err)
	}

	return toKey(key), nil
}

func (kr *repo) Remove(ctx context.Context, issuerID, id string) error {
	q := `DELETE FROM keys WHERE issuer_id = :issuer_id AND id = :id`
	key := dbKey{
		ID:     id,
		Issuer: issuerID,
	}
	if _, err := kr.db.NamedExecContext(ctx, q, key); err != nil {
		return errors.Wrap(errDelete, err)
	}

	return nil
}

type dbKey struct {
	ID        string       `db:"id"`
	Type      uint32       `db:"type"`
	Issuer    string       `db:"issuer_id"`
	Subject   string       `db:"subject"`
	IssuedAt  time.Time    `db:"issued_at"`
	ExpiresAt sql.NullTime `db:"expires_at,omitempty"`
}

func toDBKey(key auth.Key) dbKey {
	ret := dbKey{
		ID:       key.ID,
		Type:     uint32(key.Type),
		Issuer:   key.Issuer,
		Subject:  key.Subject,
		IssuedAt: key.IssuedAt,
	}
	if !key.ExpiresAt.IsZero() {
		ret.ExpiresAt = sql.NullTime{Time: key.ExpiresAt, Valid: true}
	}

	return ret
}

func toKey(key dbKey) auth.Key {
	ret := auth.Key{
		ID:       key.ID,
		Type:     auth.KeyType(key.Type),
		Issuer:   key.Issuer,
		Subject:  key.Subject,
		IssuedAt: key.IssuedAt,
	}
	if key.ExpiresAt.Valid {
		ret.ExpiresAt = key.ExpiresAt.Time
	}

	return ret
}
