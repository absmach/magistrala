// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/users"
)

var _ users.UserRepository = (*userRepository)(nil)

const errDuplicate = "unique_violation"

type userRepository struct {
	db Database
}

// New instantiates a PostgreSQL implementation of user
// repository.
func New(db Database) users.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (ur userRepository) Save(ctx context.Context, user users.User) error {
	q := `INSERT INTO users (email, password, metadata) VALUES (:email, :password, :metadata)`

	dbu := toDBUser(user)
	if _, err := ur.db.NamedExecContext(ctx, q, dbu); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && errDuplicate == pqErr.Code.Name() {
			return users.ErrConflict
		}
		return err
	}

	return nil
}

func (ur userRepository) Update(ctx context.Context, user users.User) error {
	q := `UPDATE users SET(email, password, metadata) VALUES (:email, :password, :metadata) WHERE email = :email`

	dbu := toDBUser(user)
	if _, err := ur.db.NamedExecContext(ctx, q, dbu); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && errDuplicate == pqErr.Code.Name() {
			return users.ErrConflict
		}
		return err
	}

	return nil
}

func (ur userRepository) RetrieveByID(ctx context.Context, email string) (users.User, error) {
	q := `SELECT password, metadata FROM users WHERE email = $1`

	dbu := dbUser{
		Email: email,
	}
	if err := ur.db.QueryRowxContext(ctx, q, email).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return users.User{}, users.ErrNotFound
		}
		return users.User{}, err
	}

	user := toUser(dbu)

	return user, nil
}

func (ur userRepository) UpdatePassword(ctx context.Context, email, password string) error {
	q := `UPDATE users SET  password = :password  WHERE email = :email`

	db := dbUser{
		Email:    email,
		Password: password,
	}

	if _, err := ur.db.NamedExecContext(ctx, q, db); err != nil {
		return err
	}

	return nil
}

// dbMetadata type for handling metadata properly in database/sql
type dbMetadata map[string]interface{}

// Scan - Implement the database/sql scanner interface
func (m *dbMetadata) Scan(value interface{}) error {
	if value == nil {
		m = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		m = &dbMetadata{}
		return users.ErrScanMetadata
	}

	if err := json.Unmarshal(b, m); err != nil {
		m = &dbMetadata{}
		return err
	}

	return nil
}

// Value Implements valuer
func (m dbMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, err
}

type dbUser struct {
	Email    string     `db:"email"`
	Password string     `db:"password"`
	Metadata dbMetadata `db:"metadata"`
}

func toDBUser(u users.User) dbUser {
	return dbUser{
		Email:    u.Email,
		Password: u.Password,
		Metadata: u.Metadata,
	}
}

func toUser(dbu dbUser) users.User {
	return users.User{
		Email:    dbu.Email,
		Password: dbu.Password,
		Metadata: dbu.Metadata,
	}
}
