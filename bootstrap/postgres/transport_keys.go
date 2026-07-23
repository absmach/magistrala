// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ bootstrap.DomainTransportKeyRepository = (*transportKeyRepository)(nil)

type transportKeyRepository struct {
	db postgres.Database
}

func NewDomainTransportKeyRepository(db postgres.Database) bootstrap.DomainTransportKeyRepository {
	return &transportKeyRepository{db: db}
}

func (tr transportKeyRepository) Create(ctx context.Context, key bootstrap.DomainTransportKey) error {
	q := `INSERT INTO domain_transport_keys
		(domain_id, key_id, encrypted_secret, wrapping_key_id, status, created_at, updated_at, retire_at)
		VALUES (:domain_id, :key_id, :encrypted_secret, :wrapping_key_id, :status, :created_at, :updated_at, :retire_at)`
	if _, err := tr.db.NamedExecContext(ctx, q, toDBTransportKey(key)); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return repoerr.ErrConflict
		}
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

func (tr transportKeyRepository) RetrieveCurrent(ctx context.Context, domainID string) (bootstrap.DomainTransportKey, error) {
	return tr.retrieve(ctx, domainID, "", true)
}

func (tr transportKeyRepository) Retrieve(ctx context.Context, domainID, keyID string) (bootstrap.DomainTransportKey, error) {
	return tr.retrieve(ctx, domainID, keyID, false)
}

func (tr transportKeyRepository) retrieve(ctx context.Context, domainID, keyID string, current bool) (bootstrap.DomainTransportKey, error) {
	q := `SELECT domain_id, key_id, encrypted_secret, wrapping_key_id, status, created_at, updated_at, retire_at
		FROM domain_transport_keys WHERE domain_id = :domain_id AND key_id = :key_id`
	params := dbTransportKey{DomainID: domainID, KeyID: keyID}
	if current {
		q = `SELECT domain_id, key_id, encrypted_secret, wrapping_key_id, status, created_at, updated_at, retire_at
			FROM domain_transport_keys WHERE domain_id = :domain_id AND status = 'active'`
	}
	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return bootstrap.DomainTransportKey{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return bootstrap.DomainTransportKey{}, repoerr.ErrNotFound
	}
	var stored dbTransportKey
	if err := rows.StructScan(&stored); err != nil {
		return bootstrap.DomainTransportKey{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return fromDBTransportKey(stored), nil
}

func (tr transportKeyRepository) Rotate(ctx context.Context, oldKeyID string, next bootstrap.DomainTransportKey, retireAt time.Time) error {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	defer func() { _ = tx.Rollback() }()

	updated, err := tx.ExecContext(ctx, `UPDATE domain_transport_keys
		SET status = 'retiring', retire_at = $1, updated_at = $2
		WHERE domain_id = $3 AND key_id = $4 AND status = 'active'`, retireAt, next.UpdatedAt, next.DomainID, oldKeyID)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	if count != 1 {
		return repoerr.ErrNotFound
	}
	q := `INSERT INTO domain_transport_keys
		(domain_id, key_id, encrypted_secret, wrapping_key_id, status, created_at, updated_at, retire_at)
		VALUES (:domain_id, :key_id, :encrypted_secret, :wrapping_key_id, :status, :created_at, :updated_at, :retire_at)`
	if _, err := tx.NamedExecContext(ctx, q, toDBTransportKey(next)); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	return nil
}

func (tr transportKeyRepository) ConsumeRequestID(ctx context.Context, domainID, keyID, requestID string, expiresAt time.Time) error {
	if _, err := tr.db.ExecContext(ctx, `DELETE FROM secure_bootstrap_requests WHERE expires_at <= $1`, time.Now().UTC()); err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	_, err := tr.db.ExecContext(ctx, `INSERT INTO secure_bootstrap_requests
		(domain_id, key_id, request_id, expires_at) VALUES ($1, $2, $3, $4)`, domainID, keyID, requestID, expiresAt)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return repoerr.ErrConflict
		}
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

type dbTransportKey struct {
	DomainID        string       `db:"domain_id"`
	KeyID           string       `db:"key_id"`
	EncryptedSecret string       `db:"encrypted_secret"`
	WrappingKeyID   string       `db:"wrapping_key_id"`
	Status          string       `db:"status"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
	RetireAt        sql.NullTime `db:"retire_at"`
}

func toDBTransportKey(key bootstrap.DomainTransportKey) dbTransportKey {
	retireAt := sql.NullTime{}
	if key.RetireAt != nil {
		retireAt = sql.NullTime{Time: *key.RetireAt, Valid: true}
	}
	return dbTransportKey{
		DomainID: key.DomainID, KeyID: key.KeyID, EncryptedSecret: key.EncryptedSecret,
		WrappingKeyID: key.WrappingKeyID, Status: key.Status, CreatedAt: key.CreatedAt,
		UpdatedAt: key.UpdatedAt, RetireAt: retireAt,
	}
}

func fromDBTransportKey(key dbTransportKey) bootstrap.DomainTransportKey {
	result := bootstrap.DomainTransportKey{
		DomainID: key.DomainID, KeyID: key.KeyID, EncryptedSecret: key.EncryptedSecret,
		WrappingKeyID: key.WrappingKeyID, Status: key.Status, CreatedAt: key.CreatedAt,
		UpdatedAt: key.UpdatedAt,
	}
	if key.RetireAt.Valid {
		result.RetireAt = &key.RetireAt.Time
	}
	return result
}
