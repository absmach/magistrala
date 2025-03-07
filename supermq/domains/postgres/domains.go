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

	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/postgres"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	"github.com/jackc/pgtype"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ domains.Repository = (*domainRepo)(nil)

const (
	rolesTableNamePrefix = "domains"
	entityTableName      = "domains"
	entityIDColumnName   = "id"
)

type domainRepo struct {
	db postgres.Database
	rolesPostgres.Repository
}

// NewRepository instantiates a PostgreSQL
// implementation of Domain repository.
func NewRepository(db postgres.Database) domains.Repository {
	rmsvcRepo := rolesPostgres.NewRepository(db, policies.DomainType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	return &domainRepo{
		db:         db,
		Repository: rmsvcRepo,
	}
}

func (repo domainRepo) SaveDomain(ctx context.Context, d domains.Domain) (dd domains.Domain, err error) {
	q := `INSERT INTO domains (id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status)
	VALUES (:id, :name, :tags, :alias, :metadata, :created_at, :updated_at, :updated_by, :created_by, :status)
	RETURNING id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status;`

	dbd, err := toDBDomain(d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrCreateEntity, errors.ErrRollbackTx)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return domains.Domain{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	domain, err := toDomain(dbd)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return domain, nil
}

// RetrieveDomainByID retrieves Domain by its unique ID.
func (repo domainRepo) RetrieveDomainByID(ctx context.Context, id string) (domains.Domain, error) {
	q := `SELECT d.id as id, d.name as name, d.tags as tags,  d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
        FROM domains d WHERE d.id = :id`

	dbdp := dbDomainsPage{
		ID: id,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbdp)
	if err != nil {
		return domains.Domain{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbd := dbDomain{}
	if rows.Next() {
		if err = rows.StructScan(&dbd); err != nil {
			return domains.Domain{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		domain, err := toDomain(dbd)
		if err != nil {
			return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return domain, nil
	}
	return domains.Domain{}, repoerr.ErrNotFound
}

func (repo domainRepo) RetrieveDomainByUserAndID(ctx context.Context, userID, id string) (domains.Domain, error) {
	q := repo.userDomainsBaseQuery() +
		`SELECT
			d.id as id,
			d.name as name,
			d.tags as tags,
			d.alias as alias,
			d.metadata as metadata,
			d.status as status,
			d.role_id AS role_id,
			d.role_name AS role_name,
			d.actions AS actions,
			d.created_at as created_at,
			d.updated_at as updated_at,
			d.updated_by as updated_by,
			d.created_by as created_by
		FROM
			domains d
		WHERE d.id = :id
		`

	dbdp := dbDomainsPage{
		ID:     id,
		UserID: userID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbdp)
	if err != nil {
		return domains.Domain{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbd := dbDomain{}
	if rows.Next() {
		if err = rows.StructScan(&dbd); err != nil {
			return domains.Domain{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		domain, err := toDomain(dbd)
		if err != nil {
			return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return domain, nil
	}
	return domains.Domain{}, repoerr.ErrNotFound
}

// RetrieveAllByIDs retrieves for given Domain IDs .
func (repo domainRepo) RetrieveAllDomainsByIDs(ctx context.Context, pm domains.Page) (domains.DomainsPage, error) {
	var q string
	if len(pm.IDs) == 0 {
		return domains.DomainsPage{}, nil
	}
	query, err := buildPageQuery(pm)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	q = `SELECT d.id as id, d.name as name, d.tags as tags, d.alias as alias, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
	FROM domains d`
	q = fmt.Sprintf("%s %s  LIMIT %d OFFSET %d;", q, query, pm.Limit, pm.Offset)

	dbPage, err := toDBDomainsPage(pm)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	doms, err := repo.processRows(rows)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	cq := "SELECT COUNT(*) FROM domains d"
	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	return domains.DomainsPage{
		Total:   total,
		Offset:  pm.Offset,
		Limit:   pm.Limit,
		Domains: doms,
	}, nil
}

// ListDomains list domains of user.
func (repo domainRepo) ListDomains(ctx context.Context, pm domains.Page) (domains.DomainsPage, error) {
	query, err := buildPageQuery(pm)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}
	squery := applyOrdering(query, pm)

	q := `SELECT
			d.id as id,
			d.name as name,
			d.tags as tags,
			d.alias as alias,
			d.metadata as metadata,
			d.created_at as created_at,
			d.updated_at as updated_at,
			d.updated_by as updated_by,
			d.created_by as created_by,
			d.status as status
		FROM
			domains as d
		%s
		LIMIT :limit OFFSET :offset`

	if pm.UserID != "" {
		q = repo.userDomainsBaseQuery() +
			`
			SELECT
				d.id as id,
				d.name as name,
				d.tags as tags,
				d.alias as alias,
				d.metadata as metadata,
				d.status as status,
				d.role_id AS role_id,
				d.role_name AS role_name,
				d.actions AS actions,
				d.created_at as created_at,
				d.updated_at as updated_at,
				d.updated_by as updated_by,
				d.created_by as created_by
			FROM
				domains d
			%s
			LIMIT :limit OFFSET :offset
			`
	}

	q = fmt.Sprintf(q, squery)

	dbPage, err := toDBDomainsPage(pm)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	doms, err := repo.processRows(rows)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	cq := `SELECT COUNT(*)
		FROM domains as d %s`

	if pm.UserID != "" {
		cq = repo.userDomainsBaseQuery() + cq
	}

	if query != "" {
		cq = fmt.Sprintf(cq, query)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	return domains.DomainsPage{
		Total:   total,
		Offset:  pm.Offset,
		Limit:   pm.Limit,
		Domains: doms,
	}, nil
}

// UpdateDomain updates the client name and metadata.
func (repo domainRepo) UpdateDomain(ctx context.Context, id string, dr domains.DomainReq) (domains.Domain, error) {
	var query []string
	var upq string
	d := domains.Domain{ID: id}
	if dr.Name != nil && *dr.Name != "" {
		query = append(query, "name = :name")
		d.Name = *dr.Name
	}
	if dr.Metadata != nil {
		query = append(query, "metadata = :metadata")
		d.Metadata = *dr.Metadata
	}
	if dr.Tags != nil {
		query = append(query, "tags = :tags")
		d.Tags = *dr.Tags
	}
	if dr.Status != nil {
		query = append(query, "status = :status")
		d.Status = *dr.Status
	}
	if dr.Alias != nil {
		query = append(query, "alias = :alias")
		d.Alias = *dr.Alias
	}
	d.UpdatedAt = time.Now()
	if dr.UpdatedAt != nil {
		query = append(query, "updated_at = :updated_at")
		d.UpdatedAt = *dr.UpdatedAt
	}
	if dr.UpdatedBy != nil {
		query = append(query, "updated_by = :updated_by")
		d.UpdatedAt = *dr.UpdatedAt
	}
	if len(query) > 0 {
		upq = strings.Join(query, ", ")
	}
	q := fmt.Sprintf(`UPDATE domains SET %s
        WHERE id = :id
        RETURNING id, name, tags, alias, metadata, created_at, updated_at, updated_by, created_by, status;`,
		upq)

	dbd, err := toDBDomain(d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return domains.Domain{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	// defer row.Close()
	row.Next()
	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	domain, err := toDomain(dbd)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return domain, nil
}

// Delete delete domain from database.
func (repo domainRepo) DeleteDomain(ctx context.Context, id string) error {
	q := "DELETE FROM domains WHERE id = $1;"

	res, err := repo.db.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo domainRepo) userDomainsBaseQuery() string {
	return `
		with domains AS (
			SELECT
				d.id as id,
				d.name as name,
				d.tags as tags,
				d.alias as alias,
				d.metadata as metadata,
				d.created_at as created_at,
				d.updated_at as updated_at,
				d.updated_by as updated_by,
				d.created_by as created_by,
				d.status as status,
				dr.entity_id AS entity_id,
				drm.member_id AS member_id,
				dr.id AS role_id,
				dr."name" AS role_name,
				array_agg(dra."action") AS actions
			FROM
				domains_role_members drm
			JOIN
				domains_role_actions dra ON dra.role_id = drm.role_id
			JOIN
				domains_roles dr ON dr.id = drm.role_id
			JOIN
				"domains" d ON d.id = dr.entity_id
			WHERE
				drm.member_id = :member_id
			GROUP BY
				dr.entity_id, drm.member_id, dr.id, dr."name", d.id
		)`
}

func (repo domainRepo) processRows(rows *sqlx.Rows) ([]domains.Domain, error) {
	var items []domains.Domain
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

func applyOrdering(emq string, pm domains.Page) string {
	switch pm.Order {
	case "name", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY d.%s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}

type dbDomain struct {
	ID        string           `db:"id"`
	Name      string           `db:"name"`
	Metadata  []byte           `db:"metadata,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Alias     *string          `db:"alias,omitempty"`
	Status    domains.Status   `db:"status"`
	RoleID    string           `db:"role_id"`
	RoleName  string           `db:"role_name"`
	Actions   pq.StringArray   `db:"actions"`
	CreatedBy string           `db:"created_by"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
}

func toDBDomain(d domains.Domain) (dbDomain, error) {
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
		ID:        d.ID,
		Name:      d.Name,
		Metadata:  data,
		Tags:      tags,
		Alias:     alias,
		Status:    d.Status,
		RoleID:    d.RoleID,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}, nil
}

func toDomain(d dbDomain) (domains.Domain, error) {
	var metadata domains.Metadata
	if d.Metadata != nil {
		if err := json.Unmarshal([]byte(d.Metadata), &metadata); err != nil {
			return domains.Domain{}, errors.Wrap(errors.ErrMalformedEntity, err)
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

	return domains.Domain{
		ID:        d.ID,
		Name:      d.Name,
		Metadata:  metadata,
		Tags:      tags,
		Alias:     alias,
		RoleID:    d.RoleID,
		RoleName:  d.RoleName,
		Actions:   d.Actions,
		Status:    d.Status,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}, nil
}

type dbDomainsPage struct {
	Total    uint64         `db:"total"`
	Limit    uint64         `db:"limit"`
	Offset   uint64         `db:"offset"`
	Order    string         `db:"order"`
	Dir      string         `db:"dir"`
	Name     string         `db:"name"`
	RoleID   string         `db:"role_id"`
	RoleName string         `db:"role_name"`
	Actions  pq.StringArray `db:"actions"`
	ID       string         `db:"id"`
	IDs      []string       `db:"ids"`
	Metadata []byte         `db:"metadata"`
	Tag      string         `db:"tag"`
	Status   domains.Status `db:"status"`
	UserID   string         `db:"member_id"`
}

func toDBDomainsPage(pm domains.Page) (dbDomainsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbDomainsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbDomainsPage{
		Total:    pm.Total,
		Limit:    pm.Limit,
		Offset:   pm.Offset,
		Order:    pm.Order,
		Dir:      pm.Dir,
		Name:     pm.Name,
		RoleID:   pm.RoleID,
		RoleName: pm.RoleName,
		Actions:  pm.Actions,
		ID:       pm.ID,
		IDs:      pm.IDs,
		Metadata: data,
		Tag:      pm.Tag,
		Status:   pm.Status,
		UserID:   pm.UserID,
	}, nil
}

func buildPageQuery(pm domains.Page) (string, error) {
	var query []string
	var emq string

	if pm.ID != "" {
		query = append(query, "d.id = :id")
	}

	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("d.id IN ('%s')", strings.Join(pm.IDs, "','")))
	}

	if (pm.Status >= domains.EnabledStatus) && (pm.Status < domains.AllStatus) {
		query = append(query, "d.status = :status")
	} else {
		query = append(query, fmt.Sprintf("d.status < %d", domains.AllStatus))
	}

	if pm.Name != "" {
		query = append(query, "d.name = :name")
	}

	if pm.UserID != "" {
		if pm.RoleName != "" {
			query = append(query, "d.role_name = :role_name")
		}

		if pm.RoleID != "" {
			query = append(query, "d.role_id = :role_id")
		}

		if len(pm.Actions) != 0 {
			query = append(query, "d.actions @> :actions")
		}
	}

	if pm.Tag != "" {
		query = append(query, ":tag = ANY(d.tags)")
	}

	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrViewEntity, err)
	}
	if mq != "" {
		query = append(query, mq)
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}
