//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/users"
)

var _ users.UserRepository = (*userRepository)(nil)

const errDuplicate = "unique_violation"

type userRepository struct {
	db *sql.DB
}

// New instantiates a PostgreSQL implementation of user
// repository.
func New(db *sql.DB) users.UserRepository {
	return &userRepository{db}
}

func (ur userRepository) Save(user users.User) error {
	q := `INSERT INTO users (email, password) VALUES ($1, $2)`

	if _, err := ur.db.Exec(q, user.Email, user.Password); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && errDuplicate == pqErr.Code.Name() {
			return users.ErrConflict
		}
		return err
	}

	return nil
}

func (ur userRepository) RetrieveByID(email string) (users.User, error) {
	q := `SELECT password FROM users WHERE email = $1`

	user := users.User{}
	if err := ur.db.QueryRow(q, email).Scan(&user.Password); err != nil {
		if err == sql.ErrNoRows {
			return user, users.ErrNotFound
		}
		return user, err
	}

	user.Email = email

	return user, nil
}
