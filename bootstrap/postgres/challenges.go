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
	"github.com/jackc/pgx/v5/pgconn"
)

const maxActiveBootstrapChallenges = 5

var _ bootstrap.BootstrapChallengeRepository = (*challengeRepository)(nil)

type challengeRepository struct {
	db postgres.Database
}

func NewBootstrapChallengeRepository(db postgres.Database) bootstrap.BootstrapChallengeRepository {
	return &challengeRepository{db: db}
}

func (cr challengeRepository) Create(ctx context.Context, challenge bootstrap.BootstrapChallenge) error {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM bootstrap_challenges
		WHERE expires_at <= $1 OR (consumed_at IS NOT NULL AND consumed_at <= $2)`,
		challenge.CreatedAt, challenge.CreatedAt.Add(-time.Minute)); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM bootstrap_challenges
		WHERE challenge_id IN (
			SELECT challenge_id FROM bootstrap_challenges
			WHERE config_id = $1 AND consumed_at IS NULL AND expires_at > $2
			ORDER BY created_at DESC OFFSET $3
		)`, challenge.ConfigID, challenge.CreatedAt, maxActiveBootstrapChallenges-1); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	q := `INSERT INTO bootstrap_challenges
		(challenge_id, config_id, key_version, server_nonce, created_at, expires_at, consumed_at)
		VALUES (:challenge_id, :config_id, :key_version, :server_nonce, :created_at, :expires_at, :consumed_at)`
	if _, err := tx.NamedExecContext(ctx, q, toDBChallenge(challenge)); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return repoerr.ErrConflict
		}
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

func (cr challengeRepository) Retrieve(ctx context.Context, challengeID, configID string) (bootstrap.BootstrapChallenge, error) {
	q := `SELECT challenge_id, config_id, key_version, server_nonce, created_at, expires_at, consumed_at
		FROM bootstrap_challenges WHERE challenge_id = :challenge_id AND config_id = :config_id`
	rows, err := cr.db.NamedQueryContext(ctx, q, dbChallenge{ID: challengeID, ConfigID: configID})
	if err != nil {
		return bootstrap.BootstrapChallenge{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return bootstrap.BootstrapChallenge{}, repoerr.ErrNotFound
	}
	var stored dbChallenge
	if err := rows.StructScan(&stored); err != nil {
		return bootstrap.BootstrapChallenge{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return fromDBChallenge(stored), nil
}

func (cr challengeRepository) Consume(ctx context.Context, challengeID, configID string, now time.Time) error {
	result, err := cr.db.ExecContext(ctx, `UPDATE bootstrap_challenges SET consumed_at = $1
		WHERE challenge_id = $2 AND config_id = $3 AND consumed_at IS NULL AND expires_at > $1`,
		now, challengeID, configID)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	if count != 1 {
		return repoerr.ErrConflict
	}
	return nil
}

type dbChallenge struct {
	ID          string       `db:"challenge_id"`
	ConfigID    string       `db:"config_id"`
	KeyVersion  uint64       `db:"key_version"`
	ServerNonce []byte       `db:"server_nonce"`
	CreatedAt   time.Time    `db:"created_at"`
	ExpiresAt   time.Time    `db:"expires_at"`
	ConsumedAt  sql.NullTime `db:"consumed_at"`
}

func toDBChallenge(challenge bootstrap.BootstrapChallenge) dbChallenge {
	stored := dbChallenge{
		ID: challenge.ID, ConfigID: challenge.ConfigID, KeyVersion: challenge.KeyVersion,
		ServerNonce: challenge.ServerNonce, CreatedAt: challenge.CreatedAt, ExpiresAt: challenge.ExpiresAt,
	}
	if challenge.ConsumedAt != nil {
		stored.ConsumedAt = sql.NullTime{Time: *challenge.ConsumedAt, Valid: true}
	}
	return stored
}

func fromDBChallenge(challenge dbChallenge) bootstrap.BootstrapChallenge {
	result := bootstrap.BootstrapChallenge{
		ID: challenge.ID, ConfigID: challenge.ConfigID, KeyVersion: challenge.KeyVersion,
		ServerNonce: challenge.ServerNonce, CreatedAt: challenge.CreatedAt, ExpiresAt: challenge.ExpiresAt,
	}
	if challenge.ConsumedAt.Valid {
		result.ConsumedAt = &challenge.ConsumedAt.Time
	}
	return result
}
