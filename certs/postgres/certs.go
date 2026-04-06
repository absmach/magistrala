// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres error codes:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	errDuplicate      = "23505" // unique_violation
	errTruncation     = "22001" // string_data_right_truncation
	errFK             = "23503" // foreign_key_violation
	errInvalid        = "22P02" // invalid_text_representation
	errUntranslatable = "22P05" // untranslatable_character
	errInvalidChar    = "22021" // character_not_in_repertoire
)

var (
	ErrConflict        = errors.New("entity already exists")
	ErrMalformedEntity = errors.New("malformed entity")
	ErrCreateEntity    = errors.New("failed to create entity")
	ErrNotFound        = errors.New("entity not found")
)

type certsRepo struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) certs.Repository {
	return certsRepo{
		db: db,
	}
}

// SaveCertEntityMapping saves the mapping between certificate serial number and entity ID.
func (repo certsRepo) SaveCertEntityMapping(ctx context.Context, serialNumber, entityID string) error {
	q := `INSERT INTO cert_entity_mappings (serial_number, entity_id) VALUES ($1, $2)`
	_, err := repo.db.ExecContext(ctx, q, serialNumber, entityID)
	if err != nil {
		return handleError(ErrCreateEntity, err)
	}
	return nil
}

// GetEntityIDBySerial retrieves the entity ID for a given certificate serial number.
func (repo certsRepo) GetEntityIDBySerial(ctx context.Context, serialNumber string) (string, error) {
	q := `SELECT entity_id FROM cert_entity_mappings WHERE serial_number = $1`
	var entityID string
	if err := repo.db.QueryRowxContext(ctx, q, serialNumber).Scan(&entityID); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(ErrNotFound, err)
		}
		return "", handleError(ErrNotFound, err)
	}
	return entityID, nil
}

// ListCertsByEntityID lists all certificate serial numbers for a given entity ID.
func (repo certsRepo) ListCertsByEntityID(ctx context.Context, entityID string) ([]string, error) {
	q := `SELECT serial_number FROM cert_entity_mappings WHERE entity_id = $1 ORDER BY created_at DESC`
	rows, err := repo.db.QueryContext(ctx, q, entityID)
	if err != nil {
		return nil, handleError(ErrNotFound, err)
	}
	defer rows.Close()

	var serialNumbers []string
	for rows.Next() {
		var serialNumber string
		if err := rows.Scan(&serialNumber); err != nil {
			return nil, handleError(ErrNotFound, err)
		}
		serialNumbers = append(serialNumbers, serialNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, handleError(ErrNotFound, err)
	}

	return serialNumbers, nil
}

// RemoveCertEntityMapping removes the mapping between certificate and entity ID.
func (repo certsRepo) RemoveCertEntityMapping(ctx context.Context, serialNumber string) error {
	q := `DELETE FROM cert_entity_mappings WHERE serial_number = $1`
	result, err := repo.db.ExecContext(ctx, q, serialNumber)
	if err != nil {
		return handleError(ErrNotFound, err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}

	return nil
}

func handleError(wrapper, err error) error {
	pqErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pqErr.Code {
		case errDuplicate:
			return errors.Wrap(ErrConflict, err)
		case errInvalid, errInvalidChar, errTruncation, errUntranslatable:
			return errors.Wrap(ErrMalformedEntity, err)
		case errFK:
			return errors.Wrap(ErrCreateEntity, err)
		}
	}

	return errors.Wrap(wrapper, err)
}
