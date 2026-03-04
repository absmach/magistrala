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
	groups "github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/internal/nullable"
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

var _ groups.Repository = (*groupRepository)(nil)

const (
	rolesTableNamePrefix = "groups"
	entityTableName      = "groups"
	entityIDColumnName   = "id"
)

var (
	errParentGroupID   = errors.New("parent group id is empty")
	errParentGroupPath = errors.New("parent group path is empty")
	errParentSuffix    = errors.New("parent group path doesn't have parent id suffix")
)

type groupRepository struct {
	db postgres.Database
	eh errors.Handler
	rolesPostgres.Repository
}

// New instantiates a PostgreSQL implementation of group
// repository.
func New(db postgres.Database) groups.Repository {
	roleRepo := rolesPostgres.NewRepository(db, policies.GroupType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	errHandlerOptions := []errors.HandlerOption{
		postgres.WithDuplicateErrors(NewDuplicateErrors()),
	}
	return &groupRepository{
		db:         db,
		eh:         postgres.NewErrorHandler(errHandlerOptions...),
		Repository: roleRepo,
	}
}

func (repo groupRepository) Save(ctx context.Context, g groups.Group) (groups.Group, error) {
	q, err := repo.getInsertQuery(ctx, g)
	if err != nil {
		return groups.Group{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	dbg, err := toDBGroup(g)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	return toGroup(dbg)
}

func (repo groupRepository) Update(ctx context.Context, g groups.Group) (groups.Group, error) {
	var query []string
	var upq string
	if g.Name != "" {
		query = append(query, "name = :name,")
	}
	if g.Description.Valid {
		query = append(query, "description = :description,")
	}
	if g.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}
	g.Status = groups.EnabledStatus
	q := fmt.Sprintf(`UPDATE groups SET %s updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id AND status = :status
		RETURNING id, name, tags, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, updated_at, updated_by, status`, upq)

	dbu, err := toDBGroup(g)
	if err != nil {
		return groups.Group{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return groups.Group{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbu = dbGroup{}
	if err := row.StructScan(&dbu); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	return toGroup(dbu)
}

func (repo groupRepository) UpdateTags(ctx context.Context, group groups.Group) (groups.Group, error) {
	q := `UPDATE groups SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_id, '') AS parent_id, status, created_at, updated_at, updated_by`
	group.Status = groups.EnabledStatus

	dbg, err := toDBGroup(group)
	if err != nil {
		return groups.Group{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbg = dbGroup{}
	if row.Next() {
		if err := row.StructScan(&dbg); err != nil {
			return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
		}

		return toGroup(dbg)
	}

	return groups.Group{}, repoerr.ErrNotFound
}

func (repo groupRepository) ChangeStatus(ctx context.Context, group groups.Group) (groups.Group, error) {
	qc := `UPDATE groups SET status = :status, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id
	RETURNING id, name, tags, description, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, updated_at, updated_by, status`

	dbg, err := toDBGroup(group)
	if err != nil {
		return groups.Group{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	row, err := repo.db.NamedQueryContext(ctx, qc, dbg)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()
	if ok := row.Next(); !ok {
		return groups.Group{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	return toGroup(dbg)
}

func (repo groupRepository) RetrieveByID(ctx context.Context, id string) (groups.Group, error) {
	q := `SELECT id, name, tags, domain_id, COALESCE(parent_id, '') AS parent_id, description, metadata, created_at, updated_at, updated_by, status, path FROM groups
	    WHERE id = :id`

	dbg := dbGroup{
		ID: id,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbg = dbGroup{}
	if ok := row.Next(); !ok {
		return groups.Group{}, repoerr.ErrNotFound
	}
	if err := row.StructScan(&dbg); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	return toGroup(dbg)
}

func (repo groupRepository) RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (groups.Group, error) {
	query := `
	WITH selected_group AS (
    SELECT
        g.id,
        g.parent_id,
        g.domain_id,
        g.path AS parent_group_path
    FROM
        groups g
    WHERE
        g.id = :id
    LIMIT 1
	),
	selected_group_roles AS (
		SELECT
			sg.id AS group_id,
			grm.member_id AS member_id,
			gr.id AS role_id,
			gr.name AS role_name,
			jsonb_agg(DISTINCT gra.action) AS actions,
			g.path AS access_provider_path,
			gr.entity_id AS access_provider_id,
			CASE
				WHEN gr.entity_id = sg.id THEN 'direct'
				WHEN gr.entity_id = sg.parent_id THEN 'direct_group'
				ELSE 'indirect_group'
			END AS access_type
		FROM
			groups g
		JOIN
			groups_roles gr ON gr.entity_id = g.id
		JOIN
			groups_role_members grm ON gr.id = grm.role_id
		JOIN
			groups_role_actions gra ON gr.id = gra.role_id
		JOIN
			selected_group sg ON g.path @> sg.parent_group_path
		WHERE
			grm.member_id = :member_id
			AND (
				(gr.entity_id = sg.id)
				OR (gr.entity_id <> sg.id AND gra.action LIKE 'subgroup%%')
			)
		GROUP BY
			sg.id, gr.entity_id, gr.id, gr.name, g.path, grm.member_id, sg.parent_id
	),
	selected_domain_roles AS (
		SELECT
			sg.id AS group_id,
			drm.member_id AS member_id,
			dr.id AS role_id,
			dr.name AS role_name,
			jsonb_agg(DISTINCT all_actions.action) AS actions,
			''::::ltree access_provider_path,
			'domain' AS access_type,
			dr.entity_id AS access_provider_id
		FROM
			domains d
		JOIN
			selected_group sg ON sg.domain_id = d.id
		JOIN
			domains_roles dr ON dr.entity_id = d.id
		JOIN
			domains_role_members drm ON dr.id = drm.role_id
		JOIN
			domains_role_actions dra ON dr.id = dra.role_id
		JOIN
			domains_role_actions all_actions ON dr.id = all_actions.role_id
		WHERE
			drm.member_id = :member_id
			AND dra.action LIKE 'group%%'
		GROUP BY
			sg.id, dr.entity_id, dr.id, dr.name, drm.member_id
	),
	all_roles AS (
		SELECT
			sgr.group_id,
			sgr.member_id,
			sgr.role_id AS role_id,
			sgr.role_name AS role_name,
			sgr.actions AS actions,
			sgr.access_type AS access_type,
			sgr.access_provider_path AS access_provider_path,
			sgr.access_provider_id AS access_provider_id
		FROM
			selected_group_roles sgr
		UNION
		SELECT
			sdr.group_id,
			sdr.member_id,
			sdr.role_id AS role_id,
			sdr.role_name AS role_name,
			sdr.actions AS actions,
			sdr.access_type AS access_type,
			sdr.access_provider_path AS access_provider_path,
			sdr.access_provider_id AS access_provider_id
		FROM
			selected_domain_roles sdr
	),
	final_roles AS (
		SELECT
			ar.group_id,
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
		FROM all_roles ar
		GROUP BY
			ar.group_id, ar.member_id
	)
	SELECT
		g.id,
		g.parent_id,
		g.domain_id,
		g.name,
		g.tags,
		g.description,
		g.path,
		g.metadata,
		g.created_at,
		g.updated_at,
		g.updated_by,
		g.status,
		fr.member_id,
		fr.roles
	FROM groups g
		JOIN final_roles fr ON fr.group_id = g.id
	`

	parameters := map[string]any{
		"id":        id,
		"member_id": memberID,
	}
	row, err := repo.db.NamedQueryContext(ctx, query, parameters)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbg := dbGroup{}
	if !row.Next() {
		return groups.Group{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbg); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	return toGroup(dbg)
}

func (repo groupRepository) RetrieveByIDAndUser(ctx context.Context, domainID, userID, groupID string) (groups.Group, error) {
	baseQuery := repo.userGroupsBaseQuery(domainID, userID)

	dbg := dbGroup{ID: groupID}
	q := fmt.Sprintf(`%s
					SELECT
						g.id,
						g.name,
						g.domain_id,
						COALESCE(g.parent_id, '') AS parent_id,
						g.tags,
						g.description,
						g.metadata,
						g.created_at,
						g.updated_at,
						g.updated_by,
						g.status,
						g.path as path,
						g.role_id,
						g.role_name,
						g.actions,
						g.access_type,
						g.access_provider_id,
						g.access_provider_role_id,
						g.access_provider_role_name,
						g.access_provider_role_actions
					FROM
						final_groups g
					WHERE
						g.id = :id
					LIMIT 1
					;
					`,
		baseQuery)

	row, err := repo.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbg = dbGroup{}
	if ok := row.Next(); !ok {
		return groups.Group{}, repoerr.ErrNotFound
	}
	if err := row.StructScan(&dbg); err != nil {
		return groups.Group{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	return toGroup(dbg)
}

func (repo groupRepository) RetrieveAll(ctx context.Context, pm groups.PageMeta) (groups.Page, error) {
	query := buildQuery(pm)

	if pm.RootGroup {
		query += " AND nlevel(g.path) = 1 "
	}

	orderClause := ""
	var orderBy string
	switch pm.Order {
	case "name":
		orderBy = "g.name"
	case "created_at":
		orderBy = "g.created_at"
	case "updated_at":
		orderBy = "COALESCE(g.updated_at, g.created_at)"
	}

	if orderBy != "" {
		dir := pm.Dir
		if dir != api.AscDir && dir != api.DescDir {
			dir = api.DescDir
		}
		orderClause = fmt.Sprintf("ORDER BY %s %s, g.id %s", orderBy, dir, dir)
	}

	q := fmt.Sprintf(`SELECT g.id, g.domain_id, tags, COALESCE(g.parent_id, '') AS parent_id, g.name, g.description,
		g.metadata, g.created_at, g.updated_at, g.updated_by, g.status FROM groups g %s %s LIMIT :limit OFFSET :offset;`, query, orderClause)

	dbPageMeta, err := toDBGroupPageMeta(pm)
	if err != nil {
		return groups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	var items []groups.Group
	if !pm.OnlyTotal {
		rows, err := repo.db.NamedQueryContext(ctx, q, dbPageMeta)
		if err != nil {
			return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
		defer rows.Close()

		items, err = repo.processRows(rows)
		if err != nil {
			return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
	}

	cq := fmt.Sprintf(`	SELECT COUNT(*) AS total_count
						FROM (
							SELECT g.id, g.domain_id, COALESCE(g.parent_id, '') AS parent_id, g.name, g.tags, g.description,
							g.metadata, g.created_at, g.updated_at, g.updated_by, g.status FROM groups g %s
						) AS subquery;
						`, query)

	total, err := postgres.Total(ctx, repo.db, cq, dbPageMeta)
	if err != nil {
		return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	page := groups.Page{PageMeta: pm}
	page.Total = total
	page.Groups = items
	return page, nil
}

func (repo groupRepository) RetrieveByIDs(ctx context.Context, pm groups.PageMeta, ids ...string) (groups.Page, error) {
	var q string
	if (len(ids) == 0) && (pm.DomainID == "") {
		return groups.Page{PageMeta: groups.PageMeta{Offset: pm.Offset, Limit: pm.Limit}}, nil
	}
	query := buildQuery(pm, ids...)

	q = fmt.Sprintf(`SELECT DISTINCT g.id, g.domain_id, tags, COALESCE(g.parent_id, '') AS parent_id, g.name, g.tags, g.description,
		g.metadata, g.created_at, g.updated_at, g.updated_by, g.status FROM groups g %s ORDER BY g.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPageMeta, err := toDBGroupPageMeta(pm)
	if err != nil {
		return groups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbPageMeta)
	if err != nil {
		return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	items, err := repo.processRows(rows)
	if err != nil {
		return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	cq := fmt.Sprintf(`	SELECT COUNT(*) AS total_count
						FROM (
							SELECT DISTINCT g.id, g.domain_id, COALESCE(g.parent_id, '') AS parent_id, g.name, g.tags, g.description,
							g.metadata, g.created_at, g.updated_at, g.updated_by, g.status FROM groups g %s
						) AS subquery;
						`, query)

	total, err := postgres.Total(ctx, repo.db, cq, dbPageMeta)
	if err != nil {
		return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	page := groups.Page{PageMeta: pm}
	page.Total = total
	page.Groups = items
	return page, nil
}

func (repo groupRepository) RetrieveHierarchy(ctx context.Context, domainID, userID, groupID string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	var dirQuery string
	switch {
	case hm.Direction >= 0:
		dirQuery = "g.path @> (SELECT path FROM groups WHERE id = :id)"
	default:
		dirQuery = "g.path <@ (SELECT path FROM groups WHERE id = :id)"
	}

	baseQuery := repo.userGroupsBaseQuery(domainID, userID)
	query := fmt.Sprintf(`%s,
		target_hierarchy AS (
			SELECT
				g.id,
				g.parent_id,
				g.domain_id,
				g.name,
				g.tags,
				g.description,
				g.metadata,
				g.created_at,
				g.updated_at,
				g.updated_by,
				g.status,
				g.path,
				nlevel(g.path) AS level
			FROM
				groups g
			WHERE
				%s
		),
		filtered_hierarchy AS (
			SELECT
				th.*
			FROM
				target_hierarchy th
			JOIN
				final_groups fg ON th.id = fg.id
		)
		SELECT
			*
		FROM
			filtered_hierarchy
		ORDER BY path;
		`, baseQuery, dirQuery)

	parameters := map[string]any{
		"id":    groupID,
		"level": hm.Level,
	}

	rows, err := repo.db.NamedQueryContext(ctx, query, parameters)
	if err != nil {
		return groups.HierarchyPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	items, err := repo.processRows(rows)
	if err != nil {
		return groups.HierarchyPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	return groups.HierarchyPage{HierarchyPageMeta: hm, Groups: items}, nil
}

func (repo groupRepository) AssignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) (err error) {
	if len(groupIDs) == 0 {
		return nil
	}

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(err, errRollback)
			}
		}
	}()

	pq := `SELECT id, path FROM groups WHERE id = $1 LIMIT 1;`
	rows, err := tx.Queryx(pq, parentGroupID)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer rows.Close()

	pGroups, err := repo.processRows(rows)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if len(pGroups) == 0 {
		return repoerr.ErrUpdateEntity
	}
	pGroup := pGroups[0]

	if pGroup.ID == "" {
		return errors.Wrap(repoerr.ErrViewEntity, errParentGroupID)
	}
	if pGroup.Path == "" {
		return errors.Wrap(repoerr.ErrViewEntity, errParentGroupPath)
	}
	if !strings.HasSuffix(pGroup.Path, pGroup.ID) {
		return errors.Wrap(repoerr.ErrViewEntity, errParentSuffix)
	}
	sPaths := strings.Split(pGroup.Path, ".") // 021b9f24-5337-469b-abfa-586f5813dd41.bd4a1fea-6303-4dca-9628-301cd1165a8c.c7e8f389-11e9-4849-a474-e186012ddf38
	for _, sPath := range sPaths {
		for _, cgid := range groupIDs {
			if sPath == cgid {
				return errors.Wrap(repoerr.ErrUpdateEntity, errCyclicParentGroup)
			}
		}
	}

	query := `	UPDATE groups
			SET parent_id = :parent_id
			WHERE id = ANY(:children_group_ids)
			RETURNING id, path;`

	params := map[string]any{
		"parent_id":          pGroup.ID,
		"children_group_ids": groupIDs,
	}

	crows, err := tx.NamedQuery(query, params)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer crows.Close()
	cgroups, err := repo.processRows(crows)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	childrenPaths := []string{}
	for _, cg := range cgroups {
		spath := strings.Split(cg.Path, ".")
		if len(spath) > 0 {
			childrenPaths = append(childrenPaths, cg.Path)
		}
	}

	query = `UPDATE groups
				SET path = text2ltree(COALESCE($1, '') || '.' || ltree2text(path))
				WHERE path <@ ANY($2::ltree[]);`

	if _, err := tx.Exec(query, pGroup.Path, childrenPaths); err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	return nil
}

func (repo groupRepository) UnassignParentGroup(ctx context.Context, parentGroupID string, groupIDs ...string) (err error) {
	if len(groupIDs) == 0 {
		return nil
	}

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(err, errRollback)
			}
		}
	}()
	pq := `SELECT id, path FROM groups WHERE id = $1 LIMIT 1;`
	rows, err := tx.Queryx(pq, parentGroupID)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer rows.Close()

	pGroups, err := repo.processRows(rows)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if len(pGroups) == 0 {
		return repoerr.ErrUpdateEntity
	}
	pGroup := pGroups[0]

	if pGroup.ID == "" {
		return errors.Wrap(repoerr.ErrViewEntity, errParentGroupID)
	}
	if pGroup.Path == "" {
		return errors.Wrap(repoerr.ErrViewEntity, errParentGroupPath)
	}

	query := `UPDATE groups
			  SET parent_id = NULL
			  WHERE id = ANY(:children_group_ids) AND parent_id = :parent_id
			  RETURNING id, path;`

	parameters := map[string]any{
		"parent_id":          pGroup.ID,
		"children_group_ids": groupIDs,
	}
	crows, err := tx.NamedQuery(query, parameters)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer crows.Close()
	cgroups, err := repo.processRows(crows)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	childrenPaths := []string{}
	for _, cg := range cgroups {
		spath := strings.Split(cg.Path, ".")
		if len(spath) > 0 {
			childrenPaths = append(childrenPaths, cg.Path)
		}
	}

	query = `UPDATE groups
				SET path = text2ltree(replace(ltree2text(path), $1 || '.', ''))
				WHERE path <@ ANY($2::ltree[]);`

	if _, err := tx.Exec(query, pGroup.Path, childrenPaths); err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	return nil
}

func (repo groupRepository) UnassignAllChildrenGroups(ctx context.Context, id string) error {
	query := `
			UPDATE groups AS g SET
				parent_id = NULL
			WHERE g.parent_id = :parent_id ;
	`

	result, err := repo.db.NamedExecContext(ctx, query, dbGroup{ParentID: &id})
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo groupRepository) Delete(ctx context.Context, groupID string) error {
	q := "DELETE FROM groups AS g WHERE g.id = $1;"

	result, err := repo.db.ExecContext(ctx, q, groupID)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (repo groupRepository) RetrieveAllParentGroups(ctx context.Context, domainID, userID, groupID string, pm groups.PageMeta) (groups.Page, error) {
	cGroup, err := repo.RetrieveByID(ctx, groupID)
	if err != nil {
		return groups.Page{}, err
	}

	query := buildQuery(pm)

	levelCondition := fmt.Sprintf("g.path @> '%s' ", cGroup.Path)

	switch {
	case query == "":
		query = " WHERE " + levelCondition
	default:
		query = query + " AND " + levelCondition
	}

	return repo.retrieveGroups(ctx, domainID, userID, query, pm)
}

func (repo groupRepository) RetrieveChildrenGroups(ctx context.Context, domainID, userID, groupID string, startLevel, endLevel int64, pm groups.PageMeta) (groups.Page, error) {
	pGroup, err := repo.RetrieveByID(ctx, groupID)
	if err != nil {
		return groups.Page{}, err
	}

	query := buildQuery(pm)

	levelCondition := ""
	switch {
	// Retrieve all children groups from parent group level
	case startLevel == 0 && endLevel < 0:
		levelCondition = fmt.Sprintf(" path ~ '%s.*'::::lquery ", pGroup.Path)

	// Retrieve specific level of children groups from parent group level
	case (startLevel > 0) && (startLevel == endLevel || endLevel == 0):
		levelCondition = fmt.Sprintf(" path ~ '%s.*{%d}'::::lquery ", pGroup.Path, startLevel)

	// Retrieve all children groups from specific level from parent group level
	case startLevel > 0 && endLevel < 0:
		levelCondition = fmt.Sprintf(" path ~ '%s.*{%d,}'::::lquery ", pGroup.Path, startLevel)

	// Retrieve children groups between specific level from parent group level
	case startLevel > 0 && endLevel > 0 && startLevel < endLevel:
		levelCondition = fmt.Sprintf(" path ~ '%s.*{%d,%d}'::::lquery ", pGroup.Path, startLevel, endLevel)
	default:
		return groups.Page{}, errors.Wrap(repoerr.ErrViewEntity, fmt.Errorf("invalid level range: start level: %d end level: %d", startLevel, endLevel))
	}

	switch {
	case query == "":
		query = " WHERE " + levelCondition
	default:
		query = query + " AND " + levelCondition
	}

	return repo.retrieveGroups(ctx, domainID, userID, query, pm)
}

func (repo groupRepository) RetrieveUserGroups(ctx context.Context, domainID, userID string, pm groups.PageMeta) (groups.Page, error) {
	query := buildQuery(pm)
	if pm.RootGroup {
		query += (` AND
			NOT EXISTS (
			SELECT 1
			FROM groups anc
			JOIN final_groups fg
				ON fg.id = anc.id
			WHERE anc.domain_id = g.domain_id
				AND anc.path @> g.path
				AND anc.id <> g.id
			)`)
	}

	return repo.retrieveGroups(ctx, domainID, userID, query, pm)
}

func (repo groupRepository) retrieveGroups(ctx context.Context, domainID, userID, query string, pm groups.PageMeta) (groups.Page, error) {
	baseQuery := repo.userGroupsBaseQuery(domainID, userID)

	orderClause := ""
	var orderBy string
	switch pm.Order {
	case "name":
		orderBy = "g.name"
	case "created_at":
		orderBy = "g.created_at"
	case "updated_at", "":
		orderBy = "COALESCE(g.updated_at, g.created_at)"
	}

	if orderBy != "" {
		dir := pm.Dir
		if dir != api.AscDir && dir != api.DescDir {
			dir = api.DescDir
		}
		orderClause = fmt.Sprintf("ORDER BY %s %s, g.id %s", orderBy, dir, dir)
	}

	q := fmt.Sprintf(`%s
        SELECT
            g.id,
            g.name,
            g.domain_id,
            COALESCE(g.parent_id, '') AS parent_id,
            g.description,
            g.tags,
            g.metadata,
            g.created_at,
            g.updated_at,
            g.updated_by,
            g.status,
            g.path as path,
            g.role_id,
            g.role_name,
            g.actions,
            g.access_type,
            g.access_provider_id,
            g.access_provider_role_id,
            g.access_provider_role_name,
            g.access_provider_role_actions
        FROM final_groups g
        %s
        %s
        LIMIT :limit OFFSET :offset;`,
		baseQuery, query, orderClause)

	dbPageMeta, err := toDBGroupPageMeta(pm)
	if err != nil {
		return groups.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	var items []groups.Group
	if !pm.OnlyTotal {
		rows, err := repo.db.NamedQueryContext(ctx, q, dbPageMeta)
		if err != nil {
			return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
		defer rows.Close()

		items, err = repo.processRows(rows)
		if err != nil {
			return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
	}

	cq := fmt.Sprintf(`%s
        SELECT COUNT(*) AS total_count
        FROM (
            SELECT g.id
            FROM final_groups g
            %s
        ) AS subquery;`,
		baseQuery, query)

	total, err := postgres.Total(ctx, repo.db, cq, dbPageMeta)
	if err != nil {
		return groups.Page{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	page := groups.Page{PageMeta: pm}
	page.Total = total
	page.Groups = items
	return page, nil
}

func (repo groupRepository) userGroupsBaseQuery(domainID, userID string) string {
	return fmt.Sprintf(`
WITH direct_groups AS (
SELECT
	g.*,
	gr.entity_id AS entity_id,
	grm.member_id AS member_id,
	gr.id AS role_id,
	gr."name" AS role_name,
	array_agg(gra."action") AS actions
FROM
	groups_role_members grm
JOIN
	groups_role_actions gra ON gra.role_id = grm.role_id
JOIN
	groups_roles gr ON gr.id = grm.role_id
JOIN
	"groups" g ON g.id = gr.entity_id
WHERE
	grm.member_id = '%s'
	AND g.domain_id = '%s'
GROUP BY
	gr.entity_id, grm.member_id, gr.id, gr."name", g."path", g.id
),
direct_groups_with_subgroup AS (
	SELECT
		g.*,
		gr.entity_id AS entity_id,
		grm.member_id AS member_id,
		gr.id AS role_id,
		gr."name" AS role_name,
		array_agg(DISTINCT gra."action") AS actions
	FROM
		groups_role_members grm
	JOIN
		groups_role_actions gra ON gra.role_id = grm.role_id
	JOIN
		groups_roles gr ON gr.id = grm.role_id
	JOIN
		"groups" g ON g.id = gr.entity_id
	WHERE
		grm.member_id = '%s'
		AND g.domain_id = '%s'
	GROUP BY
		gr.entity_id, grm.member_id, gr.id, gr."name", g."path", g.id
	HAVING
		bool_or(gra."action" LIKE 'subgroup_%%')
),
direct_leaf_groups_with_subgroup  AS (
	SELECT dgws.*
	FROM direct_groups_with_subgroup dgws
	WHERE NOT EXISTS (
		SELECT 1
		FROM direct_groups_with_subgroup dgws2
		WHERE
			dgws2.path @> dgws.path
			AND dgws2.id != dgws.id
	)
),
indirect_child_groups AS (
	SELECT
		DISTINCT indirect_child_groups.id as child_id,
		indirect_child_groups.*,
		dlgws.id as access_provider_id,
		dlgws.role_id as access_provider_role_id,
		dlgws.role_name as access_provider_role_name,
		dlgws.actions as access_provider_role_actions
	FROM
		direct_leaf_groups_with_subgroup dlgws
	JOIN
		groups indirect_child_groups ON indirect_child_groups.path <@ dlgws.path  -- Finds all children of entity_id based on ltree path
	WHERE
		indirect_child_groups.domain_id = '%s'
		AND
		NOT EXISTS (  -- Ensures that the indirect_child_groups.id is not already in the direct_groups_with_subgroup table
			SELECT 1
			FROM direct_groups_with_subgroup dgws
			WHERE dgws.id = indirect_child_groups.id
		)
),
direct_indirect_groups as (
	SELECT
		id,
		parent_id,
		domain_id,
		"name",
		tags,
		description,
		metadata,
		created_at,
		updated_at,
		updated_by,
		status,
		"path",
		role_id,
		role_name,
		actions,
		'direct' AS access_type,
		'' AS access_provider_id,
		'' AS access_provider_role_id,
		'' AS access_provider_role_name,
		array[]::::text[] AS access_provider_role_actions
	FROM
		direct_groups
	UNION
	SELECT
		id,
		parent_id,
		domain_id,
		"name",
		tags,
		description,
		metadata,
		created_at,
		updated_at,
		updated_by,
		status,
		"path",
		'' AS role_id,
		'' AS role_name,
		array[]::::text[] AS actions,
		'indirect' AS access_type,
		access_provider_id,
		access_provider_role_id,
		access_provider_role_name,
		access_provider_role_actions
	FROM
		indirect_child_groups
),
final_groups AS (
	SELECT
		dig.id,
		dig.parent_id,
		dig.domain_id,
		dig."name",
		dig.tags,
		dig.description,
		dig.metadata,
		dig.created_at,
		dig.updated_at,
		dig.updated_by,
		dig.status,
		dig."path",
		dig.role_id,
		dig.role_name,
		dig.actions,
		dig.access_type,
		dig.access_provider_id,
		dig.access_provider_role_id,
		dig.access_provider_role_name,
		dig.access_provider_role_actions
	FROM
		direct_indirect_groups as dig
	UNION
	SELECT
		dg.id,
		dg.parent_id,
		dg.domain_id,
		dg."name",
		dg.tags,
		dg.description,
		dg.metadata,
		dg.created_at,
		dg.updated_at,
		dg.updated_by,
		dg.status,
		dg."path",
		'' AS role_id,
		'' AS role_name,
		array[]::::text[] AS actions,
		'domain' AS access_type,
		d.id AS access_provider_id,
		dr.id AS access_provider_role_id,
		dr."name" AS access_provider_role_name,
		array_agg(dra."action") as actions
	FROM
		domains_role_members drm
	JOIN
		domains_role_actions dra ON dra.role_id = drm.role_id
	JOIN
		domains_roles dr ON dr.id = drm.role_id
	JOIN
		domains d ON d.id = dr.entity_id
	JOIN
		"groups" dg ON dg.domain_id = d.id
	WHERE
		drm.member_id = '%s' -- user_id
	 	AND d.id = '%s' -- domain_id
	 	AND dra."action" LIKE 'group_%%'
	 	AND NOT EXISTS (  -- Ensures that the direct and indirect groups are not in included.
			SELECT 1 FROM direct_indirect_groups dig
			WHERE dig.id = dg.id
		)
	GROUP BY
		dg.id, d.id, dr.id
)
		`, userID, domainID, userID, domainID, domainID, userID, domainID)
}

func buildQuery(gm groups.PageMeta, ids ...string) string {
	queries := []string{}

	if len(ids) > 0 {
		queries = append(queries, fmt.Sprintf(" id in ('%s') ", strings.Join(ids, "', '")))
	}
	if gm.Name != "" {
		queries = append(queries, "g.name ILIKE '%' || :name || '%'")
	}
	if gm.ID != "" {
		queries = append(queries, "g.id = :id")
	}
	if gm.Status != groups.AllStatus {
		queries = append(queries, "g.status = :status")
	}
	if len(gm.Tags.Elements) > 0 {
		switch gm.Tags.Operator {
		case groups.AndOp:
			queries = append(queries, "tags @> :tags")
		default: // OR
			queries = append(queries, "tags && :tags")
		}
	}
	if gm.DomainID != "" {
		queries = append(queries, "g.domain_id = :domain_id")
	}
	if gm.AccessType != "" {
		queries = append(queries, "g.access_type = :access_type")
	}
	if gm.RoleID != "" {
		queries = append(queries, "g.role_id = :role_id")
	}
	if gm.RoleName != "" {
		queries = append(queries, "g.role_name = :role_name")
	}
	if len(gm.Actions) != 0 {
		queries = append(queries, "g.actions @> :actions")
	}
	if len(gm.Metadata) > 0 {
		queries = append(queries, "g.metadata @> :metadata")
	}
	if !gm.CreatedFrom.IsZero() {
		queries = append(queries, "g.created_at >= :created_from")
	}
	if !gm.CreatedTo.IsZero() {
		queries = append(queries, "g.created_at <= :created_to")
	}
	if len(queries) > 0 {
		return fmt.Sprintf("WHERE %s", strings.Join(queries, " AND "))
	}

	return ""
}

type dbGroup struct {
	ID                        string           `db:"id"`
	ParentID                  *string          `db:"parent_id,omitempty"`
	DomainID                  string           `db:"domain_id,omitempty"`
	Name                      string           `db:"name"`
	Description               sql.NullString   `db:"description,omitempty"`
	Tags                      pgtype.TextArray `db:"tags,omitempty"`
	Level                     int              `db:"level"`
	Path                      string           `db:"path,omitempty"`
	Metadata                  []byte           `db:"metadata,omitempty"`
	CreatedAt                 time.Time        `db:"created_at"`
	UpdatedAt                 sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy                 *string          `db:"updated_by,omitempty"`
	Status                    groups.Status    `db:"status"`
	RoleID                    string           `db:"role_id"`
	RoleName                  string           `db:"role_name"`
	Actions                   pq.StringArray   `db:"actions"`
	AccessType                string           `db:"access_type"`
	AccessProviderId          string           `db:"access_provider_id"`
	AccessProviderRoleId      string           `db:"access_provider_role_id"`
	AccessProviderRoleName    string           `db:"access_provider_role_name"`
	AccessProviderRoleActions pq.StringArray   `db:"access_provider_role_actions"`
	MemberID                  string           `db:"member_id,omitempty"`
	Roles                     json.RawMessage  `db:"roles,omitempty"`
}

func toDBGroup(g groups.Group) (dbGroup, error) {
	data := []byte("{}")
	if len(g.Metadata) > 0 {
		b, err := json.Marshal(g.Metadata)
		if err != nil {
			return dbGroup{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(g.Tags); err != nil {
		return dbGroup{}, err
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
		Description: sql.NullString{String: g.Description.Value, Valid: g.Description.Valid},
		Tags:        tags,
		Metadata:    data,
		Path:        g.Path,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		Status:      g.Status,
	}, nil
}

func toGroup(g dbGroup) (groups.Group, error) {
	var metadata groups.Metadata
	if g.Metadata != nil {
		if err := json.Unmarshal(g.Metadata, &metadata); err != nil {
			return groups.Group{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range g.Tags.Elements {
		tags = append(tags, e.String)
	}
	var parentID string
	if g.ParentID != nil {
		parentID = *g.ParentID
	}
	var updatedAt time.Time
	if g.UpdatedAt.Valid {
		updatedAt = g.UpdatedAt.Time.UTC()
	}
	var updatedBy string
	if g.UpdatedBy != nil {
		updatedBy = *g.UpdatedBy
	}

	var roles []roles.MemberRoleActions
	if g.Roles != nil {
		if err := json.Unmarshal(g.Roles, &roles); err != nil {
			return groups.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	return groups.Group{
		ID:                        g.ID,
		Name:                      g.Name,
		Parent:                    parentID,
		Domain:                    g.DomainID,
		Description:               nullable.Value[string]{Value: g.Description.String, Valid: g.Description.Valid},
		Tags:                      tags,
		Metadata:                  metadata,
		Level:                     g.Level,
		Path:                      g.Path,
		UpdatedAt:                 updatedAt,
		UpdatedBy:                 updatedBy,
		CreatedAt:                 g.CreatedAt.UTC(),
		Status:                    g.Status,
		RoleID:                    g.RoleID,
		RoleName:                  g.RoleName,
		Actions:                   g.Actions,
		AccessType:                g.AccessType,
		AccessProviderId:          g.AccessProviderId,
		AccessProviderRoleId:      g.AccessProviderRoleId,
		AccessProviderRoleName:    g.AccessProviderRoleName,
		AccessProviderRoleActions: g.AccessProviderRoleActions,
		Roles:                     roles,
	}, nil
}

func toDBGroupPageMeta(pm groups.PageMeta) (dbGroupPageMeta, error) {
	data := []byte("{}")
	if len(pm.Metadata) > 0 {
		b, err := json.Marshal(pm.Metadata)
		if err != nil {
			return dbGroupPageMeta{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(pm.Tags.Elements); err != nil {
		return dbGroupPageMeta{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbGroupPageMeta{
		ID:          pm.ID,
		Name:        pm.Name,
		Metadata:    data,
		Tags:        tags,
		Total:       pm.Total,
		Offset:      pm.Offset,
		Limit:       pm.Limit,
		DomainID:    pm.DomainID,
		Status:      pm.Status,
		RoleName:    pm.RoleName,
		RoleID:      pm.RoleID,
		Actions:     pm.Actions,
		AccessType:  pm.AccessType,
		CreatedFrom: pm.CreatedFrom,
		CreatedTo:   pm.CreatedTo,
	}, nil
}

type dbGroupPageMeta struct {
	ID          string           `db:"id"`
	Name        string           `db:"name"`
	ParentID    string           `db:"parent_id"`
	DomainID    string           `db:"domain_id"`
	Metadata    []byte           `db:"metadata"`
	Path        string           `db:"path"`
	Level       uint64           `db:"level"`
	Total       uint64           `db:"total"`
	Limit       uint64           `db:"limit"`
	Offset      uint64           `db:"offset"`
	Subject     string           `db:"subject"`
	RoleName    string           `db:"role_name"`
	RoleID      string           `db:"role_id"`
	Actions     pq.StringArray   `db:"actions"`
	AccessType  string           `db:"access_type"`
	Status      groups.Status    `db:"status"`
	Tags        pgtype.TextArray `db:"tags"`
	CreatedFrom time.Time        `db:"created_from"`
	CreatedTo   time.Time        `db:"created_to"`
}

func (repo groupRepository) processRows(rows *sqlx.Rows) ([]groups.Group, error) {
	var items []groups.Group
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

func (repo groupRepository) getInsertQuery(c context.Context, g groups.Group) (string, error) {
	switch {
	case g.Parent != "":
		parent, err := repo.RetrieveByID(c, g.Parent)
		if err != nil {
			return "", err
		}
		path := parent.Path + "." + g.ID
		if len(strings.Split(path, ".")) > groups.MaxPathLength {
			return "", fmt.Errorf("reached max nested depth")
		}
		return fmt.Sprintf(`INSERT INTO groups (name, description, tags, id, domain_id, parent_id, metadata, created_at, status, path)
		VALUES (:name, :description, :tags, :id, :domain_id, :parent_id, :metadata, :created_at, :status, '%s')
		RETURNING id, name, description, tags, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, status, path, nlevel(path) as level;`, path), nil
	default:
		return `INSERT INTO groups (name, description, tags, id, domain_id, metadata, created_at, status, path)
		VALUES (:name, :description, :tags, :id, :domain_id, :metadata, :created_at, :status, :id)
		RETURNING id, name, description, tags, domain_id, COALESCE(parent_id, '') AS parent_id, metadata, created_at, status, path, nlevel(path) as level;`, nil
	}
}
