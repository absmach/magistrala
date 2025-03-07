// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
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

// HandleError handles the error and returns a wrapped error.
// It checks the error code and returns a specific error.
func HandleError(wrapper, err error) error {
	pqErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pqErr.Code {
		case errDuplicate:
			return errors.Wrap(repoerr.ErrConflict, err)
		case errInvalid, errInvalidChar, errTruncation, errUntranslatable:
			return errors.Wrap(repoerr.ErrMalformedEntity, err)
		case errFK:
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
	}

	return errors.Wrap(wrapper, err)
}
