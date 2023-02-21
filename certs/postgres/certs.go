// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/mainflux/mainflux/certs"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/sqlxt"
	"github.com/mainflux/mainflux/pkg/errors"
)

var errInvalidRevocationTime = errors.New("invalid revocation time")

var _ certs.Repository = (*certsRepository)(nil)

// Cert holds info on expiration date for specific cert issued for specific Thing.
type Cert struct {
	ThingID string
	Serial  string
	Expire  time.Time
}

type certsRepository struct {
	db sqlxt.Database
}

// NewRepository instantiates a PostgreSQL implementation of certs
// repository.
func NewRepository(db sqlxt.Database) certs.Repository {
	return &certsRepository{
		db: db,
	}
}

func (cr certsRepository) Save(ctx context.Context, cert certs.Cert) error {

	q := `INSERT INTO certs
			(id, name, owner_id, thing_id, serial, private_key, certificate, ca_chain, issuing_ca, ttl, expire)
		VALUES
			(:id, :name, :owner_id, :thing_id, :serial, :private_key, :certificate, :ca_chain, :issuing_ca, :ttl, :expire)
		`
	dbc, err := CertToDbCert(cert)
	if err != nil {
		return err
	}
	if _, err, txErr := cr.db.NamedCUDContext(ctx, q, dbc); err != nil || txErr != nil {
		err = pgClient.CheckError(err, pgClient.Create)
		return errors.Wrap(err, txErr)
	}
	return nil
}

func (cr certsRepository) Update(ctx context.Context, certID string, cert certs.Cert) error {
	q := `
		UPDATE
			certs
		SET
			serial = :serial,
			private_key = :private_key,
			certificate = :certificate,
			ca_chain = :ca_chain,
			issuing_ca = :issuing_ca,
			expire = :expire,
			revocation = :revocation
		WHERE id = :id AND owner_id = :owner_id
	`
	dbc, err := CertToDbCert(cert)
	if err != nil {
		return err
	}
	if _, err, txErr := cr.db.NamedCUDContext(ctx, q, dbc); err != nil || txErr != nil {
		err = pgClient.CheckError(err, pgClient.Update)
		return errors.Wrap(err, txErr)
	}
	return nil
}

func (cr certsRepository) Remove(ctx context.Context, ownerID, certID string) error {
	q := `DELETE FROM certs WHERE id = :id`

	dbc, err := CertToDbCert(certs.Cert{ID: certID})
	if err != nil {
		return err
	}
	if _, err, txErr := cr.db.NamedCUDContext(ctx, q, dbc); err != nil || txErr != nil {
		err = pgClient.CheckError(err, pgClient.Remove)
		return errors.Wrap(err, txErr)
	}
	return nil
}

func (cr certsRepository) Retrieve(ctx context.Context, ownerID, certID, thingID, serial, name string, status certs.Status, offset uint64, limit int64) (certs.Page, error) {
	q := `
	SELECT
		id, name, owner_id, thing_id, serial, private_key, certificate, ca_chain, issuing_ca, ttl, expire, revocation
	FROM
		certs
	WHERE owner_id = :owner_id
		%s
	ORDER BY expire %s;
	`

	q = fmt.Sprintf(q, whereClause(certID, thingID, serial, name, status), orderClause(limit))

	params := map[string]interface{}{
		"limit":    limit,
		"offset":   offset,
		"owner_id": ownerID,
		"id":       certID,
		"thing_id": thingID,
		"serial":   serial,
		"name":     name,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return certs.Page{}, pgClient.CheckError(err, pgClient.View)
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		dbcs := dbCert{}
		if err := rows.StructScan(&dbcs); err != nil {
			return certs.Page{}, pgClient.CheckError(err, pgClient.View)
		}
		certificates = append(certificates, dbcs.ToCert())
	}
	if len(certificates) < 1 {
		return certs.Page{}, errors.ErrNotFound
	}

	total, err := cr.RetrieveCount(ctx, ownerID, certID, thingID, serial, name, status)
	if err != nil {
		return certs.Page{}, err
	}

	return certs.Page{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Certs:  certificates,
	}, nil
}

func (cr certsRepository) RetrieveCount(ctx context.Context, ownerID, certID, thingID, serial, name string, status certs.Status) (uint64, error) {
	qc := `
	SELECT
		COUNT(*)
	FROM
		certs
	WHERE owner_id = :owner_id
		%s
	;
	`
	params := map[string]interface{}{
		"owner_id": ownerID,
		"id":       certID,
		"thing_id": thingID,
		"serial":   serial,
		"name":     name,
	}
	qc = fmt.Sprintf(qc, whereClause(certID, thingID, serial, name, status))
	total, err := cr.db.NamedTotalQueryContext(ctx, qc, params)
	if err != nil {
		return 0, pgClient.CheckError(err, pgClient.View)
	}
	return total, nil
}

func (cr certsRepository) RetrieveThingCerts(ctx context.Context, thingID string) (certs.Page, error) {
	q := `
	SELECT
		id, name, owner_id, thing_id, serial, private_key, certificate, ca_chain, issuing_ca, ttl, expire, revocation
	FROM
		certs
	WHERE thing_id = :thing_id
	ORDER BY expire;
	`

	params := certs.Cert{ThingID: thingID}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return certs.Page{}, pgClient.CheckError(err, pgClient.View)
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		dbcs := dbCert{}
		if err := rows.StructScan(&dbcs); err != nil {
			return certs.Page{}, pgClient.CheckError(err, pgClient.View)
		}
		certificates = append(certificates, dbcs.ToCert())
	}

	qc := `
	SELECT
		COUNT(*)
	FROM
		certs
	WHERE thing_id = :thing_id
	`
	total, err := cr.db.NamedTotalQueryContext(ctx, qc, params)
	if err != nil {
		return certs.Page{}, pgClient.CheckError(err, pgClient.View)
	}

	return certs.Page{
		Total:  total,
		Limit:  0,
		Offset: 0,
		Certs:  certificates,
	}, nil
}

func (cr certsRepository) RemoveThingCerts(ctx context.Context, thingID string) error {
	q := `DELETE FROM certs WHERE thing_id = thingID`
	dbc, err := CertToDbCert(certs.Cert{ThingID: thingID})
	if err != nil {
		return err
	}
	if _, err, txErr := cr.db.NamedCUDContext(ctx, q, dbc); err != nil || txErr != nil {
		err = pgClient.CheckError(err, pgClient.Remove)
		return errors.Wrap(err, txErr)
	}
	return nil
}

type dbCert struct {
	ID          string       `db:"id"`
	Name        string       `db:"name"`
	OwnerID     string       `db:"owner_id"`
	ThingID     string       `db:"thing_id"`
	Serial      string       `db:"serial"`
	Certificate string       `db:"certificate"`
	PrivateKey  string       `db:"private_key"`
	CAChain     string       `db:"ca_chain"`
	IssuingCA   string       `db:"issuing_ca"`
	TTL         string       `db:"ttl"`
	Expire      time.Time    `db:"expire"`
	Revocation  sql.NullTime `db:"revocation"`
}

func (c *dbCert) ToCert() certs.Cert {
	var rev time.Time
	if c.Revocation.Valid {
		rev = c.Revocation.Time
	}
	return certs.Cert{
		ID:          c.ID,
		Name:        c.Name,
		OwnerID:     c.OwnerID,
		ThingID:     c.ThingID,
		Serial:      c.Serial,
		Certificate: c.Certificate,
		PrivateKey:  c.PrivateKey,
		CAChain:     c.CAChain,
		IssuingCA:   c.IssuingCA,
		TTL:         c.TTL,
		Expire:      c.Expire,
		Revocation:  rev,
	}
}

func CertToDbCert(c certs.Cert) (dbCert, error) {
	var revokeTime sql.NullTime
	if !c.Revocation.IsZero() {
		if err := revokeTime.Scan(c.Revocation); err != nil {
			return dbCert{}, errors.Wrap(errInvalidRevocationTime, err)
		}
	}
	fmt.Println(revokeTime)
	return dbCert{
		ID:          c.ID,
		Name:        c.Name,
		OwnerID:     c.OwnerID,
		ThingID:     c.ThingID,
		Serial:      c.Serial,
		Certificate: c.Certificate,
		PrivateKey:  c.PrivateKey,
		CAChain:     c.CAChain,
		IssuingCA:   c.IssuingCA,
		TTL:         c.TTL,
		Expire:      c.Expire,
		Revocation:  revokeTime,
	}, nil
}

func whereClause(certID, thingID, serial, name string, status certs.Status) string {
	var clause []string
	if certID != "" {
		clause = append(clause, " id = :id ")
	}

	if thingID != "" {
		clause = append(clause, " thing_id = :thing_id ")
	}

	if serial != "" {
		clause = append(clause, " serial = :serial ")
	}

	if name != "" {
		clause = append(clause, " name = :name ")
	}

	if sf := statusFilter(status); sf != "" {
		clause = append(clause, sf)
	}

	c := strings.Join(clause, " AND ")
	if c != "" {
		c = " AND " + c
	}
	return c
}

func orderClause(limit int64) string {
	var clause []string
	if limit >= 0 {
		clause = append(clause, " LIMIT :limit ")
	}
	clause = append(clause, " OFFSET :offset ")
	return strings.Join(clause, "  ")
}

func statusFilter(status certs.Status) string {
	var filterQuery string
	switch status {
	case certs.ActiveCerts:
		filterQuery = "revocation is NULL"
	case certs.RevokedCerts:
		filterQuery = "revocation is NOT NULL"
	case certs.AllCerts:
		fallthrough
	default:
		filterQuery = ""
	}
	return filterQuery
}
