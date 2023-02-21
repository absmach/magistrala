package postgres

import (
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mainflux/mainflux/pkg/errors"
)

type Operation int

const (
	Create Operation = iota
	View
	Update
	Remove
)

func CheckError(err error, op Operation) error {
	if pErr, ok := err.(*pgconn.PgError); ok {
		switch pErr.Code {
		case pgerrcode.UniqueViolation:
			return errors.Wrap(errors.ErrConflict, err)
		case pgerrcode.InvalidTextRepresentation:
			return errors.Wrap(errors.ErrMalformedEntity, err)
		case pgerrcode.ForeignKeyViolation:
			return errors.Wrap(errors.ErrConflict, err)
		case pgerrcode.StringDataRightTruncationDataException:
			return errors.Wrap(errors.ErrMalformedEntity, err)
		}

		switch op {
		case Create:
			return errors.Wrap(errors.ErrCreateEntity, pErr)
		case View:
			return errors.Wrap(errors.ErrViewEntity, pErr)
		case Update:
			return errors.Wrap(errors.ErrUpdateEntity, pErr)
		case Remove:
			return errors.Wrap(errors.ErrRemoveEntity, pErr)
		}
	}
	return err
}
