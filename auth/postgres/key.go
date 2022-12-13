package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errSave     = errors.New("failed to save key in database")
	errRetrieve = errors.New("failed to retrieve key from database")
	errDelete   = errors.New("failed to delete key from database")
)
var _ auth.KeyRepository = (*repo)(nil)

type repo struct {
	db Database
}

// New instantiates a PostgreSQL implementation of key repository.
func New(db Database) auth.KeyRepository {
	return &repo{
		db: db,
	}
}

func (kr repo) Save(ctx context.Context, key auth.Key) (string, error) {
	q := `INSERT INTO keys (id, type, issuer_id, subject, issued_at, expires_at)
	      VALUES (:id, :type, :issuer_id, :subject, :issued_at, :expires_at)`

	dbKey := toDBKey(key)
	if _, err := kr.db.NamedExecContext(ctx, q, dbKey); err != nil {

		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return "", errors.Wrap(errors.ErrConflict, err)
		}

		return "", errors.Wrap(errSave, err)
	}

	return dbKey.ID, nil
}

func (kr repo) Retrieve(ctx context.Context, issuerID, id string) (auth.Key, error) {
	q := `SELECT id, type, issuer_id, subject, issued_at, expires_at FROM keys WHERE issuer_id = $1 AND id = $2`
	key := dbKey{}
	if err := kr.db.QueryRowxContext(ctx, q, issuerID, id).StructScan(&key); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return auth.Key{}, errors.Wrap(errors.ErrNotFound, err)
		}

		return auth.Key{}, errors.Wrap(errRetrieve, err)
	}

	return toKey(key), nil
}

func (kr repo) Remove(ctx context.Context, issuerID, id string) error {
	q := `DELETE FROM keys WHERE issuer_id = :issuer_id AND id = :id`
	key := dbKey{
		ID:       id,
		IssuerID: issuerID,
	}
	if _, err := kr.db.NamedExecContext(ctx, q, key); err != nil {
		return errors.Wrap(errDelete, err)
	}

	return nil
}

type dbKey struct {
	ID        string       `db:"id"`
	Type      uint32       `db:"type"`
	IssuerID  string       `db:"issuer_id"`
	Subject   string       `db:"subject"`
	Revoked   bool         `db:"revoked"`
	IssuedAt  time.Time    `db:"issued_at"`
	ExpiresAt sql.NullTime `db:"expires_at"`
}

func toDBKey(key auth.Key) dbKey {
	ret := dbKey{
		ID:       key.ID,
		Type:     key.Type,
		IssuerID: key.IssuerID,
		Subject:  key.Subject,
		IssuedAt: key.IssuedAt,
	}
	if !key.ExpiresAt.IsZero() {
		ret.ExpiresAt = sql.NullTime{Time: key.ExpiresAt, Valid: true}
	}

	return ret
}

func toKey(key dbKey) auth.Key {
	ret := auth.Key{
		ID:       key.ID,
		Type:     key.Type,
		IssuerID: key.IssuerID,
		Subject:  key.Subject,
		IssuedAt: key.IssuedAt,
	}
	if key.ExpiresAt.Valid {
		ret.ExpiresAt = key.ExpiresAt.Time
	}

	return ret
}
