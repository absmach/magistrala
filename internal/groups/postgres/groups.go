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

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jmoiron/sqlx"
)

var _ mggroups.Repository = (*groupRepository)(nil)

type groupRepository struct {
	db postgres.Database
}

// New instantiates a PostgreSQL implementation of group
// repository.
func New(db postgres.Database) mggroups.Repository {
	return &groupRepository{
		db: db,
	}
}

func (repo groupRepository) Save(ctx context.Context, g mggroups.Group) (mggroups.Group, error) {
	q := repo.groupQuery(ctx, g)
	dbg, err := toDBGroup(g)
	if err != nil {
		return mggroups.Group{}, err
	}
	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return mggroups.Group{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return mggroups.Group{}, err
	}

	return toGroup(dbg)
}

func (repo groupRepository) Update(ctx context.Context, g mggroups.Group) (mggroups.Group, error) {
	var query []string
	var upq string
	if g.Name != "" {
		query = append(query, "name = :name,")
	}
	if g.Description != "" {
		query = append(query, "description = :description,")
	}
	if g.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}
	g.Status = mgclients.EnabledStatus
	q := fmt.Sprintf(`UPDATE groups SET %s updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id AND status = :status
		RETURNING id, name, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, updated_at, updated_by, status, path, nlevel(path) as level`, upq)

	dbu, err := toDBGroup(g)
	if err != nil {
		return mggroups.Group{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return mggroups.Group{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return mggroups.Group{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbu = dbGroup{}
	if err := row.StructScan(&dbu); err != nil {
		return mggroups.Group{}, errors.Wrap(err, repoerr.ErrUpdateEntity)
	}
	return toGroup(dbu)
}

func (repo groupRepository) ChangeStatus(ctx context.Context, group mggroups.Group) (mggroups.Group, error) {
	qc := `UPDATE groups SET status = :status, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
	RETURNING id, name, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, updated_at, updated_by, status, path, nlevel(path) as level;`

	dbg, err := toDBGroup(group)
	if err != nil {
		return mggroups.Group{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.db.NamedQueryContext(ctx, qc, dbg)
	if err != nil {
		return mggroups.Group{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()
	if ok := row.Next(); !ok {
		return mggroups.Group{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return mggroups.Group{}, errors.Wrap(err, repoerr.ErrUpdateEntity)
	}

	return toGroup(dbg)
}

func (repo groupRepository) RetrieveByID(ctx context.Context, id string) (mggroups.Group, error) {
	q := `SELECT id, name, domain_id, COALESCE(parent_id, '') AS parent_id, description, metadata, created_at, updated_at, updated_by, status, path, nlevel(path) as level FROM groups
	    WHERE id = :id`

	dbg := dbGroup{
		ID: id,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return mggroups.Group{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbg = dbGroup{}
	if row.Next() {
		if err := row.StructScan(&dbg); err != nil {
			return mggroups.Group{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
	}

	return toGroup(dbg)
}

func (repo groupRepository) RetrieveAll(ctx context.Context, gm mggroups.Page) (mggroups.Page, error) {
	var query, q, count string

	if gm.ParentID != "" {
		q, count = buildHierachy(gm)
		query = buildQuery(gm, "AND")

		q = fmt.Sprintf(" %s %s ORDER BY g.created_at LIMIT :limit OFFSET :offset;", q, query)
	}
	if gm.ParentID == "" {
		q = `SELECT DISTINCT g.id, g.domain_id, COALESCE(g.parent_id, '') AS parent_id, g.name, g.description,
		g.metadata, g.created_at, g.updated_at, g.updated_by, g.status, nlevel(path) as level, g.path FROM groups g`
		query = buildQuery(gm, "WHERE")

		q = fmt.Sprintf(" %s %s ORDER BY g.created_at LIMIT :limit OFFSET :offset;", q, query)
	}

	dbPage, err := toDBGroupPage(gm)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	items, err := repo.processRows(rows)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	cq := "SELECT COUNT(*) FROM groups g"
	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	if count != "" {
		cq = count
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	page := gm
	page.Groups = items
	page.Total = total

	return page, nil
}

func (repo groupRepository) RetrieveByIDs(ctx context.Context, gm mggroups.Page, ids ...string) (mggroups.Page, error) {
	var cq, query, q, count string
	if (len(ids) == 0) && (gm.PageMeta.DomainID == "") {
		return mggroups.Page{PageMeta: mggroups.PageMeta{Offset: gm.Offset, Limit: gm.Limit}}, nil
	}
	query = buildQuery(gm, "WHERE", ids...)
	cq = `SELECT COUNT(*) FROM groups g`

	if query != "" {
		cq = fmt.Sprintf(" %s %s", cq, query)
	}

	q = fmt.Sprintf("SELECT DISTINCT g.id, g.domain_id, COALESCE(g.parent_id, '') AS parent_id, g.name, g.description, g.metadata, g.created_at, g.updated_at, g.updated_by, g.status, g.path, nlevel(g.path) as level FROM groups g %s ORDER BY g.created_at LIMIT :limit OFFSET :offset;", query)

	if gm.ParentID != "" {
		q, count = buildHierachy(gm)
		query = buildQuery(gm, "AND", ids...)
		q = fmt.Sprintf("%s %s ORDER BY g.created_at LIMIT :limit OFFSET :offset;", q, query)
	}

	if count != "" {
		cq = count
	}
	dbPage, err := toDBGroupPage(gm)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	items, err := repo.processRows(rows)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return mggroups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	page := gm
	page.Groups = items
	page.Total = total

	return page, nil
}

func (repo groupRepository) AssignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error {
	if len(groupIDs) == 0 {
		return nil
	}
	if parentGroupID == "" {
		return repoerr.ErrCreateEntity
	}

	var updateColumns []string
	for _, groupID := range groupIDs {
		updateColumns = append(updateColumns, fmt.Sprintf("('%s', '%s') ", groupID, parentGroupID))
	}
	uc := strings.Join(updateColumns, ",")
	query := fmt.Sprintf(`
			UPDATE groups AS g SET
				parent_id = u.parent_group_id,
				path = text2ltree(COALESCE(u.parent_group_id, '') || '.' || ltree2text(g.path))
			FROM (VALUES
				%s
			) AS u(id, parent_group_id)
			WHERE g.id = u.id;
	`, uc)

	row, err := repo.db.QueryContext(ctx, query)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	return nil
}

func (repo groupRepository) UnassignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) error {
	if len(groupIDs) == 0 {
		return nil
	}
	var updateColumns []string
	for _, groupID := range groupIDs {
		updateColumns = append(updateColumns, fmt.Sprintf("('%s', '%s') ", groupID, parentGroupID))
	}
	uc := strings.Join(updateColumns, ",")
	query := fmt.Sprintf(`
			UPDATE groups AS g SET
				parent_id = NULL,
				path = text2ltree(TRIM(BOTH '.' FROM REGEXP_REPLACE(ltree2text(g.path), '(^|\\.)' || u.parent_group_id || '(\\.|$)', '.', 'g')))
			FROM (VALUES
				%s
			) AS u(id, parent_group_id)
			WHERE g.id = u.id;
	`, uc)

	row, err := repo.db.QueryContext(ctx, query)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	return nil
}

func (repo groupRepository) Delete(ctx context.Context, groupID string) error {
	q := "DELETE FROM groups AS g WHERE g.id = $1;"

	result, err := repo.db.ExecContext(ctx, q, groupID)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func buildHierachy(gm mggroups.Page) (query, cq string) {
	switch {
	case gm.Direction >= 0: // ancestors
		query = fmt.Sprintf(`SELECT g.id, COALESCE(g.parent_id, '') AS parent_id, g.domain_id, g.name, g.description, g.metadata, g.created_at, g.updated_at, g.updated_by, g.status, g.path, nlevel(g.path) as level FROM groups parent, groups g WHERE parent.id = '%s' AND g.path @> parent.path AND nlevel(parent.path) - nlevel(g.path) <= %d`, gm.ParentID, gm.Level)
		cq = fmt.Sprintf(`SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = '%s' AND g.path @> parent.path`, gm.ParentID)

	case gm.Direction < 0: // descendants
		query = fmt.Sprintf(`SELECT g.id, COALESCE(g.parent_id, '') AS parent_id, g.domain_id, g.name, g.description, g.metadata, g.created_at, g.updated_at, g.updated_by, g.status, g.path, nlevel(g.path) as level FROM groups parent, groups g WHERE parent.id = '%s' AND g.path <@ parent.path AND nlevel(g.path) - nlevel(parent.path) < %d`, gm.ParentID, gm.Level)
		cq = fmt.Sprintf(`SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = '%s' AND g.path <@ parent.path`, gm.ParentID)
	}
	return
}

func buildQuery(gm mggroups.Page, q string, ids ...string) string {
	queries := []string{}

	if len(ids) > 0 {
		queries = append(queries, fmt.Sprintf(" g.id in ('%s') ", strings.Join(ids, "', '")))
	}
	if gm.Name != "" {
		queries = append(queries, "g.name ILIKE '%' || :name || '%'")
	}
	if gm.PageMeta.ID != "" {
		queries = append(queries, "g.id ILIKE '%' || :id || '%'")
	}
	if gm.Status != mgclients.AllStatus {
		queries = append(queries, "g.status = :status")
	}
	if gm.DomainID != "" {
		queries = append(queries, "g.domain_id = :domain_id")
	}
	if len(gm.Metadata) > 0 {
		queries = append(queries, "g.metadata @> :metadata")
	}
	if len(queries) > 0 {
		return fmt.Sprintf("%s %s", q, strings.Join(queries, " AND "))
	}

	return ""
}

type dbGroup struct {
	ID          string           `db:"id"`
	ParentID    *string          `db:"parent_id,omitempty"`
	DomainID    string           `db:"domain_id,omitempty"`
	Name        string           `db:"name"`
	Description string           `db:"description,omitempty"`
	Level       int              `db:"level"`
	Path        string           `db:"path,omitempty"`
	Metadata    []byte           `db:"metadata,omitempty"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy   *string          `db:"updated_by,omitempty"`
	Status      mgclients.Status `db:"status"`
}

func toDBGroup(g mggroups.Group) (dbGroup, error) {
	data := []byte("{}")
	if len(g.Metadata) > 0 {
		b, err := json.Marshal(g.Metadata)
		if err != nil {
			return dbGroup{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	var parentID *string
	if g.Parent != "" {
		parentID = &g.Parent
	}
	var updatedAt sql.NullTime
	if !g.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{Time: g.UpdatedAt, Valid: true}
	}
	var updatedBy *string
	if g.UpdatedBy != "" {
		updatedBy = &g.UpdatedBy
	}
	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		ParentID:    parentID,
		DomainID:    g.Domain,
		Description: g.Description,
		Metadata:    data,
		Path:        g.Path,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		Status:      g.Status,
	}, nil
}

func toGroup(g dbGroup) (mggroups.Group, error) {
	var metadata mgclients.Metadata
	if g.Metadata != nil {
		if err := json.Unmarshal(g.Metadata, &metadata); err != nil {
			return mggroups.Group{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}
	var parentID string
	if g.ParentID != nil {
		parentID = *g.ParentID
	}
	var updatedAt time.Time
	if g.UpdatedAt.Valid {
		updatedAt = g.UpdatedAt.Time
	}
	var updatedBy string
	if g.UpdatedBy != nil {
		updatedBy = *g.UpdatedBy
	}

	return mggroups.Group{
		ID:          g.ID,
		Name:        g.Name,
		Parent:      parentID,
		Domain:      g.DomainID,
		Description: g.Description,
		Metadata:    metadata,
		Level:       g.Level,
		Path:        g.Path,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		CreatedAt:   g.CreatedAt,
		Status:      g.Status,
	}, nil
}

func toDBGroupPage(pm mggroups.Page) (dbGroupPage, error) {
	level := mggroups.MaxLevel
	if pm.Level < mggroups.MaxLevel {
		level = pm.Level
	}
	data := []byte("{}")
	if len(pm.Metadata) > 0 {
		b, err := json.Marshal(pm.Metadata)
		if err != nil {
			return dbGroupPage{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	return dbGroupPage{
		ID:       pm.ID,
		Name:     pm.Name,
		Metadata: data,
		Path:     pm.Path,
		Level:    level,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		ParentID: pm.ParentID,
		DomainID: pm.DomainID,
		Status:   pm.Status,
	}, nil
}

type dbGroupPage struct {
	ClientID string           `db:"client_id"`
	ID       string           `db:"id"`
	Name     string           `db:"name"`
	ParentID string           `db:"parent_id"`
	DomainID string           `db:"domain_id"`
	Metadata []byte           `db:"metadata"`
	Path     string           `db:"path"`
	Level    uint64           `db:"level"`
	Total    uint64           `db:"total"`
	Limit    uint64           `db:"limit"`
	Offset   uint64           `db:"offset"`
	Subject  string           `db:"subject"`
	Action   string           `db:"action"`
	Status   mgclients.Status `db:"status"`
}

func (repo groupRepository) processRows(rows *sqlx.Rows) ([]mggroups.Group, error) {
	var items []mggroups.Group
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return items, err
		}
		group, err := toGroup(dbg)
		if err != nil {
			return items, err
		}
		items = append(items, group)
	}
	return items, nil
}

func (repo groupRepository) groupQuery(c context.Context, g mggroups.Group) string {
	var query, path string
	switch {
	case g.Parent != "":
		parent, err := repo.RetrieveByID(c, g.Parent)
		if err != nil {
			query = `INSERT INTO groups (name, description, id, domain_id, parent_id, metadata, created_at, status, path)
			VALUES (:name, :description, :id, :domain_id, :parent_id, :metadata, :created_at, :status, :id)
			RETURNING id, name, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, status, path, nlevel(path) as level;`
			return query
		}
		path = parent.Path + "." + g.ID
		if len(strings.Split(path, ".")) > mggroups.MaxPathLength {
			path = parent.ID + "." + g.ID
		}

		query = fmt.Sprintf(`INSERT INTO groups (name, description, id, domain_id, parent_id, metadata, created_at, status, path)
		VALUES (:name, :description, :id, :domain_id, :parent_id, :metadata, :created_at, :status, '%s')
		RETURNING id, name, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, status, path, nlevel(path) as level;`, path)

	default:
		query = `INSERT INTO groups (name, description, id, domain_id, parent_id, metadata, created_at, status, path)
		VALUES (:name, :description, :id, :domain_id, :parent_id, :metadata, :created_at, :status, :id)
		RETURNING id, name, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, status, path, nlevel(path) as level;`
	}

	return query
}
