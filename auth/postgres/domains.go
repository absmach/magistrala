// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/jackc/pgtype"
	"github.com/jmoiron/sqlx"
)

var _ auth.DomainsRepository = (*domainRepo)(nil)

type domainRepo struct {
	db postgres.Database
}

// NewDomainRepository instantiates a PostgreSQL
// implementation of Domain repository.
func NewDomainRepository(db postgres.Database) auth.DomainsRepository {
	return &domainRepo{
		db: db,
	}
}

func (repo domainRepo) Save(ctx context.Context, d auth.Domain) (ad auth.Domain, err error) {
	q := `INSERT INTO domains (id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status)
	VALUES (:id, :name, :tags, :alias, :metadata, :created_at, :updated_at, :updated_by, :created_by, :status)
	RETURNING id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status;`

	dbd, err := toDBDomains(d)
	if err != nil {
		return auth.Domain{}, errors.Wrap(repoerr.ErrCreateEntity, repoerr.ErrRollbackTx)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return auth.Domain{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return auth.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	domain, err := toDomain(dbd)
	if err != nil {
		return auth.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return domain, nil
}

// RetrieveByID retrieves Domain by its unique ID.
func (repo domainRepo) RetrieveByID(ctx context.Context, id string) (auth.Domain, error) {
	q := `SELECT d.id as id, d.name as name, d.tags as tags,  d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
        FROM domains d WHERE d.id = :id`

	dbdp := dbDomainsPage{
		ID: id,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbdp)
	if err != nil {
		if err == sql.ErrNoRows {
			return auth.Domain{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return auth.Domain{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	defer row.Close()
	row.Next()
	dbd := dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return auth.Domain{}, errors.Wrap(errors.ErrNotFound, err)
	}

	return toDomain(dbd)
}

func (repo domainRepo) RetrievePermissions(ctx context.Context, subject, id string) ([]string, error) {
	q := `SELECT pc.relation as relation
	FROM domains as d
	JOIN policies pc
	ON pc.object_id = d.id
	WHERE d.id = $1
	AND pc.subject_id = $2
	`

	rows, err := repo.db.QueryxContext(ctx, q, id, subject)
	if err != nil {
		return []string{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	domains, err := repo.processRows(rows)
	if err != nil {
		return []string{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	permissions := []string{}
	for _, domain := range domains {
		if domain.Permission != "" {
			permissions = append(permissions, domain.Permission)
		}
	}
	return permissions, nil
}

// RetrieveAllByIDs retrieves for given Domain IDs .
func (repo domainRepo) RetrieveAllByIDs(ctx context.Context, pm auth.Page) (auth.DomainsPage, error) {
	var q string
	if len(pm.IDs) <= 0 {
		return auth.DomainsPage{}, nil
	}
	query, err := buildPageQuery(pm)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}
	if query == "" {
		return auth.DomainsPage{}, nil
	}

	q = `SELECT d.id as id, d.name as name, d.tags as tags, d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
	FROM domains d`
	q = fmt.Sprintf("%s %s  LIMIT %d OFFSET %d;", q, query, pm.Limit, pm.Offset)

	dbPage, err := toDBClientsPage(pm)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	domains, err := repo.processRows(rows)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	cq := "SELECT COUNT(*) FROM domains d"
	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	pm.Total = total
	return auth.DomainsPage{
		Page:    pm,
		Domains: domains,
	}, nil
}

// ListDomains list domains of user.
func (repo domainRepo) ListDomains(ctx context.Context, pm auth.Page) (auth.DomainsPage, error) {
	var q string
	query, err := buildPageQuery(pm)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}
	if query == "" {
		return auth.DomainsPage{}, nil
	}

	q = `SELECT d.id as id, d.name as name, d.tags as tags, d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status, pc.relation as relation
	FROM domains as d
	JOIN policies pc
	ON pc.object_id = d.id`

	// The service sends the user ID in the pagemeta subject field, which filters domains by joining with the policies table.
	// For SuperAdmins, access to domains is granted without the policies filter.
	// If the user making the request is a super admin, the service will assign an empty value to the pagemeta subject field.
	// In the repository, when the pagemeta subject is empty, the query should be constructed without applying the policies filter.
	if pm.SubjectID == "" {
		q = `SELECT d.id as id, d.name as name, d.tags as tags, d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
		FROM domains as d`
	}

	q = fmt.Sprintf("%s %s LIMIT %d OFFSET %d", q, query, pm.Limit, pm.Offset)

	dbPage, err := toDBClientsPage(pm)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	domains, err := repo.processRows(rows)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	cq := "SELECT COUNT(*) FROM domains d JOIN policies pc ON pc.object_id = d.id"
	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return auth.DomainsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	pm.Total = total
	return auth.DomainsPage{
		Page:    pm,
		Domains: domains,
	}, nil
}

// Update updates the client name and metadata.
func (repo domainRepo) Update(ctx context.Context, id, userID string, dr auth.DomainReq) (auth.Domain, error) {
	var query []string
	var upq string
	var ws string = "AND status = :status"
	d := auth.Domain{ID: id}
	if dr.Name != nil && *dr.Name != "" {
		query = append(query, "name = :name, ")
		d.Name = *dr.Name
	}
	if dr.Metadata != nil {
		query = append(query, "metadata = :metadata, ")
		d.Metadata = *dr.Metadata
	}
	if dr.Tags != nil {
		query = append(query, "tags = :tags, ")
		d.Tags = *dr.Tags
	}
	if dr.Status != nil {
		ws = ""
		query = append(query, "status = :status, ")
		d.Status = *dr.Status
	}
	if dr.Alias != nil {
		query = append(query, "alias = :alias, ")
		d.Alias = *dr.Alias
	}
	d.UpdatedAt = time.Now()
	d.UpdatedBy = userID
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}
	q := fmt.Sprintf(`UPDATE domains SET %s  updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id %s
        RETURNING id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status;`,
		upq, ws)

	dbd, err := toDBDomains(d)
	if err != nil {
		return auth.Domain{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}
	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return auth.Domain{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	// defer row.Close()
	row.Next()
	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return auth.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	domain, err := toDomain(dbd)
	if err != nil {
		return auth.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return domain, nil
}

// Delete delete domain from database.
func (repo domainRepo) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`
		DELETE FROM
			domains
		WHERE
			id = '%s'
		;`, id)

	row, err := repo.db.NamedQueryContext(ctx, q, nil)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	defer row.Close()

	return nil
}

// SavePolicies save policies in domains database.
func (repo domainRepo) SavePolicies(ctx context.Context, pcs ...auth.Policy) error {
	q := `INSERT INTO policies (subject_type, subject_id, subject_relation, relation, object_type, object_id)
	VALUES (:subject_type, :subject_id, :subject_relation, :relation, :object_type, :object_id)
	RETURNING subject_type, subject_id, subject_relation, relation, object_type, object_id;`

	dbpc := toDBPolicies(pcs...)
	row, err := repo.db.NamedQueryContext(ctx, q, dbpc)
	if err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	return nil
}

// CheckPolicy check policy in domains database.
func (repo domainRepo) CheckPolicy(ctx context.Context, pc auth.Policy) error {
	q := `
		SELECT
			subject_type, subject_id, subject_relation, relation, object_type, object_id FROM policies
		WHERE
			subject_type = :subject_type
			AND subject_id = :subject_id
			AND	subject_relation = :subject_relation
			AND relation = :relation
			AND object_type = :object_type
			AND object_id = :object_id
		LIMIT 1
	`
	dbpc := toDBPolicy(pc)
	row, err := repo.db.NamedQueryContext(ctx, q, dbpc)
	if err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()
	row.Next()
	if err := row.StructScan(&dbpc); err != nil {
		return err
	}
	return nil
}

// DeletePolicies delete policies from domains database.
func (repo domainRepo) DeletePolicies(ctx context.Context, pcs ...auth.Policy) (err error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(apiutil.ErrRollbackTx, errRollback)
			}
		}
	}()

	for _, pc := range pcs {
		q := `
			DELETE FROM
				policies
			WHERE
				subject_type = :subject_type
				AND subject_id = :subject_id
				AND subject_relation = :subject_relation
				AND relation = :relation
				AND object_type = :object_type
				AND object_id = :object_id
			;`

		dbpc := toDBPolicy(pc)
		row, err := tx.NamedQuery(q, dbpc)
		if err != nil {
			return postgres.HandleError(repoerr.ErrRemoveEntity, err)
		}
		defer row.Close()
	}
	return tx.Commit()
}

func (repo domainRepo) processRows(rows *sqlx.Rows) ([]auth.Domain, error) {
	var items []auth.Domain
	for rows.Next() {
		dbd := dbDomain{}
		if err := rows.StructScan(&dbd); err != nil {
			return items, err
		}
		d, err := toDomain(dbd)
		if err != nil {
			return items, err
		}
		items = append(items, d)
	}
	return items, nil
}

type dbDomain struct {
	ID         string           `db:"id"`
	Name       string           `db:"name"`
	Metadata   []byte           `db:"metadata,omitempty"`
	Tags       pgtype.TextArray `db:"tags,omitempty"`
	Alias      *string          `db:"alias,omitempty"`
	Status     auth.Status      `db:"status"`
	Permission string           `db:"relation"`
	CreatedBy  string           `db:"created_by"`
	CreatedAt  time.Time        `db:"created_at"`
	UpdatedBy  *string          `db:"updated_by,omitempty"`
	UpdatedAt  sql.NullTime     `db:"updated_at,omitempty"`
}

func toDBDomains(d auth.Domain) (dbDomain, error) {
	data := []byte("{}")
	if len(d.Metadata) > 0 {
		b, err := json.Marshal(d.Metadata)
		if err != nil {
			return dbDomain{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(d.Tags); err != nil {
		return dbDomain{}, err
	}
	var alias *string
	if d.Alias != "" {
		alias = &d.Alias
	}

	var updatedBy *string
	if d.UpdatedBy != "" {
		updatedBy = &d.UpdatedBy
	}
	var updatedAt sql.NullTime
	if d.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: d.UpdatedAt, Valid: true}
	}

	return dbDomain{
		ID:         d.ID,
		Name:       d.Name,
		Metadata:   data,
		Tags:       tags,
		Alias:      alias,
		Status:     d.Status,
		Permission: d.Permission,
		CreatedBy:  d.CreatedBy,
		CreatedAt:  d.CreatedAt,
		UpdatedBy:  updatedBy,
		UpdatedAt:  updatedAt,
	}, nil
}

func toDomain(d dbDomain) (auth.Domain, error) {
	var metadata clients.Metadata
	if d.Metadata != nil {
		if err := json.Unmarshal([]byte(d.Metadata), &metadata); err != nil {
			return auth.Domain{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range d.Tags.Elements {
		tags = append(tags, e.String)
	}
	var alias string
	if d.Alias != nil {
		alias = *d.Alias
	}
	var updatedBy string
	if d.UpdatedBy != nil {
		updatedBy = *d.UpdatedBy
	}
	var updatedAt time.Time
	if d.UpdatedAt.Valid {
		updatedAt = d.UpdatedAt.Time
	}

	return auth.Domain{
		ID:         d.ID,
		Name:       d.Name,
		Metadata:   metadata,
		Tags:       tags,
		Alias:      alias,
		Permission: d.Permission,
		Status:     d.Status,
		CreatedBy:  d.CreatedBy,
		CreatedAt:  d.CreatedAt,
		UpdatedBy:  updatedBy,
		UpdatedAt:  updatedAt,
	}, nil
}

type dbDomainsPage struct {
	Total      uint64      `db:"total"`
	Limit      uint64      `db:"limit"`
	Offset     uint64      `db:"offset"`
	Order      string      `db:"order"`
	Dir        string      `db:"dir"`
	Name       string      `db:"name"`
	Permission string      `db:"permission"`
	ID         string      `db:"id"`
	IDs        []string    `db:"ids"`
	Metadata   []byte      `db:"metadata"`
	Tag        string      `db:"tag"`
	Status     auth.Status `db:"status"`
	SubjectID  string      `db:"subject_id"`
}

func toDBClientsPage(pm auth.Page) (dbDomainsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbDomainsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	return dbDomainsPage{
		Total:      pm.Total,
		Limit:      pm.Limit,
		Offset:     pm.Offset,
		Order:      pm.Order,
		Dir:        pm.Dir,
		Name:       pm.Name,
		Permission: pm.Permission,
		ID:         pm.ID,
		IDs:        pm.IDs,
		Metadata:   data,
		Tag:        pm.Tag,
		Status:     pm.Status,
		SubjectID:  pm.SubjectID,
	}, nil
}

func buildPageQuery(pm auth.Page) (string, error) {
	var query []string
	var emq string

	if pm.ID != "" {
		query = append(query, "d.id = :id")
	}

	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("d.id IN ('%s')", strings.Join(pm.IDs, "','")))
	}

	if pm.Status != auth.AllStatus {
		query = append(query, "d.status = :status")
	} else {
		query = append(query, fmt.Sprintf("d.status < %s", auth.AllStatus))
	}

	if pm.Name != "" {
		query = append(query, "d.name = :name")
	}

	if pm.SubjectID != "" {
		query = append(query, "pc.subject_id = :subject_id")
	}

	if pm.Permission != "" {
		query = append(query, "pc.relation = :permission")
	}

	if pm.Tag != "" {
		query = append(query, ":tag = ANY(d.tags)")
	}

	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrViewEntity, err)
	}
	if mq != "" {
		query = append(query, mq)
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}

type dbPolicy struct {
	SubjectType     string `db:"subject_type,omitempty"`
	SubjectID       string `db:"subject_id,omitempty"`
	SubjectRelation string `db:"subject_relation,omitempty"`
	Relation        string `db:"relation,omitempty"`
	ObjectType      string `db:"object_type,omitempty"`
	ObjectID        string `db:"object_id,omitempty"`
}

func toDBPolicies(pcs ...auth.Policy) []dbPolicy {
	var dbpcs []dbPolicy
	for _, pc := range pcs {
		dbpcs = append(dbpcs, dbPolicy{
			SubjectType:     pc.SubjectType,
			SubjectID:       pc.SubjectID,
			SubjectRelation: pc.SubjectRelation,
			Relation:        pc.Relation,
			ObjectType:      pc.ObjectType,
			ObjectID:        pc.ObjectID,
		})
	}
	return dbpcs
}

func toDBPolicy(pc auth.Policy) dbPolicy {
	return dbPolicy{
		SubjectType:     pc.SubjectType,
		SubjectID:       pc.SubjectID,
		SubjectRelation: pc.SubjectRelation,
		Relation:        pc.Relation,
		ObjectType:      pc.ObjectType,
		ObjectID:        pc.ObjectID,
	}
}
