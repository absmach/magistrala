// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
)

var _ certs.Repository = (*certsRepository)(nil)

// Cert holds info on expiration date for specific cert issued for specific Thing.
type Cert struct {
	ThingID string
	Serial  string
	Expire  time.Time
}

type certsRepository struct {
	db  *sqlx.DB
	log logger.Logger
}

// NewRepository instantiates a PostgreSQL implementation of certs
// repository.
func NewRepository(db *sqlx.DB, log logger.Logger) certs.Repository {
	return &certsRepository{db: db, log: log}
}

func (cr certsRepository) RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, owner_id, serial, expire FROM certs WHERE owner_id = $1 ORDER BY expire LIMIT $2 OFFSET $3;`
	rows, err := cr.db.Query(q, ownerID, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return certs.Page{}, err
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		c := certs.Cert{}
		if err := rows.Scan(&c.ThingID, &c.OwnerID, &c.Serial, &c.Expire); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return certs.Page{}, err

		}
		certificates = append(certificates, c)
	}

	q = `SELECT COUNT(*) FROM certs WHERE owner_id = $1`
	var total uint64
	if err := cr.db.QueryRow(q, ownerID).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count certs due to %s", err))
		return certs.Page{}, err
	}

	return certs.Page{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Certs:  certificates,
	}, nil
}

func (cr certsRepository) Save(ctx context.Context, cert certs.Cert) (string, error) {
	q := `INSERT INTO certs (thing_id, owner_id, serial, expire) VALUES (:thing_id, :owner_id, :serial, :expire)`

	tx, err := cr.db.Beginx()
	if err != nil {
		return "", errors.Wrap(errors.ErrCreateEntity, err)
	}

	dbcrt := toDBCert(cert)

	if _, err := tx.NamedExec(q, dbcrt); err != nil {
		e := err
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			e = errors.New("error conflict")
		}

		cr.rollback("Failed to insert a Cert", tx, err)

		return "", errors.Wrap(errors.ErrCreateEntity, e)
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config save", tx, err)
	}

	return cert.Serial, nil
}

func (cr certsRepository) Remove(ctx context.Context, ownerID, serial string) error {
	if _, err := cr.RetrieveBySerial(ctx, ownerID, serial); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	q := `DELETE FROM certs WHERE serial = :serial`
	var c certs.Cert
	c.Serial = serial
	dbcrt := toDBCert(c)
	if _, err := cr.db.NamedExecContext(ctx, q, dbcrt); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (cr certsRepository) RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, owner_id, serial, expire FROM certs WHERE owner_id = $1 AND thing_id = $2 ORDER BY expire LIMIT $3 OFFSET $4;`
	rows, err := cr.db.Query(q, ownerID, thingID, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return certs.Page{}, err
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		c := certs.Cert{}
		if err := rows.Scan(&c.ThingID, &c.OwnerID, &c.Serial, &c.Expire); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return certs.Page{}, err

		}
		certificates = append(certificates, c)
	}

	q = `SELECT COUNT(*) FROM certs WHERE owner_id = $1 AND thing_id = $2`
	var total uint64
	if err := cr.db.QueryRow(q, ownerID, thingID).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count certs due to %s", err))
		return certs.Page{}, err
	}

	return certs.Page{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Certs:  certificates,
	}, nil
}

func (cr certsRepository) RetrieveBySerial(ctx context.Context, ownerID, serialID string) (certs.Cert, error) {
	q := `SELECT thing_id, owner_id, serial, expire FROM certs WHERE owner_id = $1 AND serial = $2`
	var dbcrt dbCert
	var c certs.Cert

	if err := cr.db.QueryRowxContext(ctx, q, ownerID, serialID).StructScan(&dbcrt); err != nil {

		pqErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pqErr.Code {
			return c, errors.Wrap(errors.ErrNotFound, err)
		}

		return c, errors.Wrap(errors.ErrViewEntity, err)
	}
	c = toCert(dbcrt)

	return c, nil
}

func (cr certsRepository) rollback(content string, tx *sqlx.Tx, err error) {
	cr.log.Error(fmt.Sprintf("%s %s", content, err))

	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

type dbCert struct {
	ThingID string    `db:"thing_id"`
	Serial  string    `db:"serial"`
	Expire  time.Time `db:"expire"`
	OwnerID string    `db:"owner_id"`
}

func toDBCert(c certs.Cert) dbCert {
	return dbCert{
		ThingID: c.ThingID,
		OwnerID: c.OwnerID,
		Serial:  c.Serial,
		Expire:  c.Expire,
	}
}

func toCert(cdb dbCert) certs.Cert {
	var c certs.Cert
	c.OwnerID = cdb.OwnerID
	c.ThingID = cdb.ThingID
	c.Serial = cdb.Serial
	c.Expire = cdb.Expire
	return c
}
