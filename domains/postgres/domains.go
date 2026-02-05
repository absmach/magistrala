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
	"github.com/absmach/supermq/pkg/roles"
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
	eh errors.Handler
	rolesPostgres.Repository
}

// NewRepository instantiates a PostgreSQL
// implementation of Domain repository.
func NewRepository(db postgres.Database) domains.Repository {
	rmsvcRepo := rolesPostgres.NewRepository(db, policies.DomainType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	errHandlerOptions := []errors.HandlerOption{
		postgres.WithDuplicateErrors(NewDuplicateErrors()),
	}
	return &domainRepo{
		db:         db,
		eh:         postgres.NewErrorHandler(errHandlerOptions...),
		Repository: rmsvcRepo,
	}
}

func (repo domainRepo) SaveDomain(ctx context.Context, d domains.Domain) (dd domains.Domain, err error) {
	q := `INSERT INTO domains (id, name, tags, route, metadata, created_at, updated_at, updated_by, created_by, status)
	VALUES (:id, :name, :tags, :route, :metadata, :created_at, :updated_at, :updated_by, :created_by, :status)
	RETURNING id, name, tags, route, metadata, created_at, updated_at, updated_by, created_by, status;`

	dbd, err := toDBDomain(d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrCreateEntity, errors.ErrRollbackTx)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return domains.Domain{}, repoerr.ErrNotFound
	}

	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrFailedOpDB, err)
	}

	domain, err := toDomain(dbd)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return domain, nil
}

// RetrieveDomainByIDWithRoles retrieves Domain by its unique ID along with member roles.
func (repo domainRepo) RetrieveDomainByIDWithRoles(ctx context.Context, id string, memberID string) (domains.Domain, error) {
	q := `
	WITH all_roles AS (
		SELECT
			d.id AS domain_id,
			drm.member_id AS member_id,
			dr.id AS role_id,
			dr."name" AS role_name,
			jsonb_agg(DISTINCT all_actions."action") AS actions,
			'direct' AS access_type,
			'' AS access_provider_path,
			'' AS access_provider_id
		FROM
			domains d
		JOIN
				domains_roles dr ON
			dr.entity_id = d.id
		JOIN
				domains_role_members drm ON
			dr.id = drm.role_id
		JOIN
				domains_role_actions dra ON
			dr.id = dra.role_id
		JOIN
				domains_role_actions all_actions ON
			dr.id = all_actions.role_id
		WHERE
			d.id = :id
			AND
			drm.member_id = :member_id
		GROUP BY
			d.id,
			dr.id,
			dr."name",
			drm.member_id
	),
	final_roles AS (
		SELECT
			ar.domain_id,
			ar.member_id,
			jsonb_agg(
				jsonb_build_object(
					'role_id', ar.role_id,
					'role_name', ar.role_name,
					'actions', ar.actions,
					'access_type', ar.access_type,
					'access_provider_path', ar.access_provider_path,
					'access_provider_id', ar.access_provider_id
				)
			) AS roles
		FROM
			all_roles ar
		GROUP BY
			ar.domain_id,
			ar.member_id
	)
	SELECT
		d.id,
		d.name,
		d.tags,
		d.route,
		d.metadata,
		d.created_at,
		d.updated_at,
		d.updated_by,
		d.created_by,
		d.status,
		fr.member_id,
		fr.roles
	FROM
		domains d
	JOIN final_roles fr ON
		d.id = fr.domain_id
	`

	dbdp := dbDomainsPage{
		ID:     id,
		UserID: memberID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbdp)
	if err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbd := dbDomain{}
	if rows.Next() {
		if err = rows.StructScan(&dbd); err != nil {
			return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		domain, err := toDomain(dbd)
		if err != nil {
			return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return domain, nil
	}
	return domains.Domain{}, repoerr.ErrNotFound
}

// RetrieveDomainByID retrieves Domain by its unique ID.
func (repo domainRepo) RetrieveDomainByID(ctx context.Context, id string) (domains.Domain, error) {
	q := `SELECT d.id as id, d.name as name, d.tags as tags,  d.route as route, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
        FROM domains d WHERE d.id = :id`

	dbdp := dbDomainsPage{
		ID: id,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbdp)
	if err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbd := dbDomain{}
	if rows.Next() {
		if err = rows.StructScan(&dbd); err != nil {
			return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		domain, err := toDomain(dbd)
		if err != nil {
			return domains.Domain{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return domain, nil
	}
	return domains.Domain{}, repoerr.ErrNotFound
}

// RetrieveDomainByRoute retrieves Domain by its unique route.
func (repo domainRepo) RetrieveDomainByRoute(ctx context.Context, route string) (domains.Domain, error) {
	q := `SELECT d.id as id, d.name as name, d.tags as tags,  d.route as route, d.metadata as metadata, d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by, d.created_by as created_by, d.status as status
		FROM domains d WHERE d.route = :route`

	dbdom := dbDomain{
		Route: &route,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbdom)
	if err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbd := dbDomain{}
	if rows.Next() {
		if err = rows.StructScan(&dbd); err != nil {
			return domains.Domain{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
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

	baseQ := `SELECT d.id as id, d.name as name, d.tags as tags, d.route as route, d.metadata as metadata,
		d.created_at as created_at, d.updated_at as updated_at, d.updated_by as updated_by,
		d.created_by as created_by, d.status as status FROM domains d`

	squery := applyOrdering(query, pm)

	q = fmt.Sprintf("%s %s  LIMIT %d OFFSET %d;", baseQ, squery, pm.Limit, pm.Offset)

	dbPage, err := toDBDomainsPage(pm)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	doms, err := repo.processRows(rows)
	if err != nil {
		return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	cq := "SELECT COUNT(*) FROM domains d"
	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
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
			d.route as route,
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
				d.route as route,
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

	var doms []domains.Domain
	if !pm.OnlyTotal {
		rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
		defer rows.Close()

		doms, err = repo.processRows(rows)
		if err != nil {
			return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
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
		return domains.DomainsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
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
	d.UpdatedAt = time.Now().UTC()
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
		RETURNING id, name, tags, route, metadata, created_at, updated_at, updated_by, created_by, status;`, upq)

	dbd, err := toDBDomain(d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbd)
	if err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	if !row.Next() {
		return domains.Domain{}, repoerr.ErrNotFound
	}

	dbd = dbDomain{}
	if err := row.StructScan(&dbd); err != nil {
		return domains.Domain{}, repo.eh.HandleError(repoerr.ErrFailedOpDB, err)
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
		return repo.eh.HandleError(repoerr.ErrRemoveEntity, err)
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
				d.route as route,
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
	col := "COALESCE(d.updated_at, d.created_at)"

	switch pm.Order {
	case "name":
		col = "d.name"
	case "created_at":
		col = "d.created_at"
	case "updated_at", "":
		col = "COALESCE(d.updated_at, d.created_at)"
	}

	dir := pm.Dir
	if dir != api.AscDir && dir != api.DescDir {
		dir = api.DescDir
	}

	return fmt.Sprintf("%s ORDER BY %s %s, d.id %s", emq, col, dir, dir)
}

type dbDomain struct {
	ID        string           `db:"id"`
	Name      string           `db:"name"`
	Metadata  []byte           `db:"metadata,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Route     *string          `db:"route,omitempty"`
	Status    domains.Status   `db:"status"`
	RoleID    string           `db:"role_id"`
	RoleName  string           `db:"role_name"`
	Actions   pq.StringArray   `db:"actions"`
	CreatedBy string           `db:"created_by"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	MemberID  string           `db:"member_id,omitempty"`
	Roles     json.RawMessage  `db:"roles,omitempty"`
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
	var route *string
	if d.Route != "" {
		route = &d.Route
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
		Route:     route,
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
	var route string
	if d.Route != nil {
		route = *d.Route
	}
	var updatedBy string
	if d.UpdatedBy != nil {
		updatedBy = *d.UpdatedBy
	}
	var updatedAt time.Time
	if d.UpdatedAt.Valid {
		updatedAt = d.UpdatedAt.Time.UTC()
	}

	var mra []roles.MemberRoleActions
	if d.Roles != nil {
		if err := json.Unmarshal(d.Roles, &mra); err != nil {
			return domains.Domain{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	return domains.Domain{
		ID:        d.ID,
		Name:      d.Name,
		Metadata:  metadata,
		Tags:      tags,
		Route:     route,
		RoleID:    d.RoleID,
		RoleName:  d.RoleName,
		Actions:   d.Actions,
		Status:    d.Status,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt.UTC(),
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
		MemberID:  d.MemberID,
		Roles:     mra,
	}, nil
}

type dbDomainsPage struct {
	Total       uint64           `db:"total"`
	Limit       uint64           `db:"limit"`
	Offset      uint64           `db:"offset"`
	Order       string           `db:"order"`
	Dir         string           `db:"dir"`
	Name        string           `db:"name"`
	RoleID      string           `db:"role_id"`
	RoleName    string           `db:"role_name"`
	Actions     pq.StringArray   `db:"actions"`
	ID          string           `db:"id"`
	IDs         []string         `db:"ids"`
	Metadata    []byte           `db:"metadata"`
	Tags        pgtype.TextArray `db:"tags"`
	Status      domains.Status   `db:"status"`
	UserID      string           `db:"member_id"`
	CreatedFrom time.Time        `db:"created_from"`
	CreatedTo   time.Time        `db:"created_to"`
}

func toDBDomainsPage(pm domains.Page) (dbDomainsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbDomainsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	var tags pgtype.TextArray
	if err := tags.Set(pm.Tags.Elements); err != nil {
		return dbDomainsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbDomainsPage{
		Total:       pm.Total,
		Limit:       pm.Limit,
		Offset:      pm.Offset,
		Order:       pm.Order,
		Dir:         pm.Dir,
		Name:        pm.Name,
		RoleID:      pm.RoleID,
		RoleName:    pm.RoleName,
		Actions:     pm.Actions,
		ID:          pm.ID,
		IDs:         pm.IDs,
		Metadata:    data,
		Tags:        tags,
		Status:      pm.Status,
		UserID:      pm.UserID,
		CreatedFrom: pm.CreatedFrom,
		CreatedTo:   pm.CreatedTo,
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
		query = append(query, "d.name ILIKE '%' || :name || '%'")
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

	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case domains.AndOp:
			query = append(query, "tags @> :tags")
		default: // OR
			query = append(query, "tags && :tags")
		}
	}

	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrViewEntity, err)
	}
	if mq != "" {
		query = append(query, mq)
	}

	if !pm.CreatedFrom.IsZero() {
		query = append(query, "d.created_at >= :created_from")
	}
	if !pm.CreatedTo.IsZero() {
		query = append(query, "d.created_at <= :created_to")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}
