// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/users"
)

// AddUserVerification adds new verification for given user id and email.
func (repo *userRepo) AddUserVerification(ctx context.Context, uv users.UserVerification) error {
	q := `INSERT INTO users_verifications (user_id, email, otp, created_at, expires_at )
		VALUES (:user_id, :email, :otp, :created_at, :expires_at );`
	dbuv := toDBUserVerification(uv)
	if _, err := repo.Repository.DB.NamedExecContext(ctx, q, dbuv); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

// RetrieveUserVerification retrieves verification details of given user id and email.
func (repo *userRepo) RetrieveUserVerification(ctx context.Context, userID, email string) (users.UserVerification, error) {
	dbuv := dbUserVerification{
		UserID: userID,
		Email:  email,
	}
	q := `SELECT user_id, email, otp, created_at, expires_at , used_at FROM users_verifications WHERE user_id = :user_id AND email = :email ORDER BY created_at DESC LIMIT 1 `

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbuv)
	if err != nil {
		return users.UserVerification{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	if !row.Next() {
		return users.UserVerification{}, repoerr.ErrNotFound
	}

	defer row.Close()

	if err := row.StructScan(&dbuv); err != nil {
		return users.UserVerification{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return toUserVerification(dbuv), nil
}

// UpdateUserVerification update user verification details for the given user id and email.
func (repo *userRepo) UpdateUserVerification(ctx context.Context, uv users.UserVerification) error {
	q := `UPDATE users_verifications SET otp = :otp, used_at = :used_at WHERE user_id = :user_id AND email = :email`
	dbuv := toDBUserVerification(uv)
	res, err := repo.Repository.DB.NamedExecContext(ctx, q, dbuv)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	if rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

type dbUserVerification struct {
	UserID    string         `db:"user_id"`
	Email     string         `db:"email"`
	OTP       sql.NullString `db:"otp"`
	CreatedAt sql.NullTime   `db:"created_at"`
	ExpiresAt sql.NullTime   `db:"expires_at"`
	UsedAt    sql.NullTime   `db:"used_at"`
}

func toDBUserVerification(uv users.UserVerification) dbUserVerification {
	var otp sql.NullString
	if uv.OTP != "" {
		otp = sql.NullString{String: uv.OTP, Valid: true}
	}
	var createdAt sql.NullTime
	if !uv.CreatedAt.IsZero() {
		createdAt = sql.NullTime{Time: uv.CreatedAt, Valid: true}
	}
	var expiresAt sql.NullTime
	if !uv.ExpiresAt.IsZero() {
		expiresAt = sql.NullTime{Time: uv.ExpiresAt, Valid: true}
	}
	var usedAt sql.NullTime
	if !uv.UsedAt.IsZero() {
		usedAt = sql.NullTime{Time: uv.UsedAt, Valid: true}
	}

	return dbUserVerification{
		UserID:    uv.UserID,
		Email:     uv.Email,
		OTP:       otp,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		UsedAt:    usedAt,
	}
}

func toUserVerification(dbuv dbUserVerification) users.UserVerification {
	var createdAt time.Time
	if dbuv.CreatedAt.Valid {
		createdAt = dbuv.CreatedAt.Time.UTC()
	}

	var expiresAt time.Time
	if dbuv.ExpiresAt.Valid {
		expiresAt = dbuv.ExpiresAt.Time.UTC()
	}

	var usedAt time.Time
	if dbuv.UsedAt.Valid {
		usedAt = dbuv.UsedAt.Time.UTC()
	}

	return users.UserVerification{
		UserID:    dbuv.UserID,
		Email:     dbuv.Email,
		OTP:       dbuv.OTP.String,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		UsedAt:    usedAt,
	}
}
