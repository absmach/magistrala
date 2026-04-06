// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ errors.Handler = (*errHandler)(nil)

type errHandler struct {
	duplicateErrors errors.Mapper
}

func WithDuplicateErrors(mapper errors.Mapper) errors.HandlerOption {
	return func(eh *errors.Handler) {
		if h, ok := (*eh).(*errHandler); ok {
			h.duplicateErrors = mapper
		}
	}
}

func NewErrorHandler(opts ...errors.HandlerOption) errors.Handler {
	var eh errors.Handler = &errHandler{}
	for _, opt := range opts {
		opt(&eh)
	}
	return eh
}

// Handle handles the error.
func (eh errHandler) HandleError(wrapper, err error) error {
	pqErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pqErr.Code {
		case errDuplicate:
			if eh.duplicateErrors != nil {
				if knownErr, ok := eh.duplicateErrors.GetError(pqErr.ConstraintName); ok {
					return errors.Wrap(wrapper, knownErr)
				}
			}
			return errors.Wrap(wrapper, err)
		case errInvalid, errInvalidChar, errTruncation, errUntranslatable:
			return errors.Wrap(wrapper, err)
		case errFK:
			return errors.Wrap(wrapper, err)
		}
	}

	return errors.Wrap(wrapper, err)
}
