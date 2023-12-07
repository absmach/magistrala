// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"errors"

	repoerror "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres error codes:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	errDuplicate  = "23505" // unique_violation
	errTruncation = "22001" // string_data_right_truncation
	errFK         = "23503" // foreign_key_violation
	errInvalid    = "22P02" // invalid_text_representation
)

func HandleError(wrapper, err error) error {
	pqErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pqErr.Code {
		case errDuplicate:
			return errors.Join(repoerror.ErrConflict, err)
		case errInvalid, errTruncation:
			return errors.Join(repoerror.ErrMalformedEntity, err)
		case errFK:
			return errors.Join(repoerror.ErrCreateEntity, err)
		}
	}

	return errors.Join(wrapper, err)
}
