package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errSave     = errors.New("failed to save key in database")
	errRetrieve = errors.New("failed to retrieve key from database")
	errDelete   = errors.New("failed to delete key from database")
)
var _ authn.KeyRepository = (*repo)(nil)

const (
	errDuplicate = "unique_violation"
	errInvalid   = "invalid_text_representation"
)

type repo struct {
	db Database
}

// New instantiates a PostgreSQL implementation of key repository.
func New(db Database) authn.KeyRepository {
	return &repo{
		db: db,
	}
}

func (kr repo) Save(ctx context.Context, key authn.Key) (string, error) {
	q := `INSERT INTO keys (id, type, issuer_id, subject, issued_at, expires_at)
	      VALUES (:id, :type, :issuer_id, :subject, :issued_at, :expires_at)`

	dbKey := toDBKey(key)
	if _, err := kr.db.NamedExecContext(ctx, q, dbKey); err != nil {

		pqErr, ok := err.(*pq.Error)
		if ok {
			if pqErr.Code.Name() == errDuplicate {
				return "", errors.Wrap(authn.ErrConflict, pqErr)
			}
		}

		return "", errors.Wrap(errSave, err)
	}

	return dbKey.ID, nil
}

func (kr repo) Retrieve(ctx context.Context, issuerID, id string) (authn.Key, error) {
	q := `SELECT id, type, issuer_id, subject, issued_at, expires_at FROM keys WHERE issuer_id = $1 AND id = $2`
	key := dbKey{}
	if err := kr.db.QueryRowxContext(ctx, q, issuerID, id).StructScan(&key); err != nil {
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return authn.Key{}, errors.Wrap(authn.ErrNotFound, err)
		}

		return authn.Key{}, errors.Wrap(errRetrieve, err)
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

func toDBKey(key authn.Key) dbKey {
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

func toKey(key dbKey) authn.Key {
	ret := authn.Key{
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
