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
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	"github.com/jackc/pgtype"
	"github.com/lib/pq"
)

const (
	entityTableName      = "clients"
	entityIDColumnName   = "id"
	rolesTableNamePrefix = "clients"
)

var _ clients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	DB postgres.Database
	eh errors.Handler
	rolesPostgres.Repository
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) clients.Repository {
	repo := rolesPostgres.NewRepository(db, policies.ClientType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	errHandlerOptions := []errors.HandlerOption{
		postgres.WithDuplicateErrors(NewDuplicateErrors()),
	}
	return &clientRepo{
		DB:         db,
		eh:         postgres.NewErrorHandler(errHandlerOptions...),
		Repository: repo,
	}
}

func (repo *clientRepo) Save(ctx context.Context, cls ...clients.Client) ([]clients.Client, error) {
	var dbClients []DBClient

	for _, client := range cls {
		dbcli, err := ToDBClient(client)
		if err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		dbClients = append(dbClients, dbcli)
	}
	q := `INSERT INTO clients (id, name, tags, domain_id, parent_group_id, identity, secret, metadata, private_metadata, created_at, updated_at, updated_by, status)
	VALUES (:id, :name, :tags, :domain_id, :parent_group_id, :identity, :secret, :metadata, :private_metadata, :created_at, :updated_at, :updated_by, :status)
	RETURNING id, name, tags, identity, secret, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS  parent_group_id, status, created_at, updated_at, updated_by`

	row, err := repo.DB.NamedQueryContext(ctx, q, dbClients)
	if err != nil {
		return []clients.Client{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	var reClients []clients.Client
	for row.Next() {
		dbcli := DBClient{}
		if err := row.StructScan(&dbcli); err != nil {
			return []clients.Client{}, repo.eh.HandleError(repoerr.ErrFailedOpDB, err)
		}

		client, err := ToClient(dbcli)
		if err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}
		reClients = append(reClients, client)
	}
	return reClients, nil
}

func (repo *clientRepo) RetrieveBySecret(ctx context.Context, key, id string, prefix authn.AuthPrefix) (clients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id,  COALESCE(parent_group_id, '') AS parent_group_id, identity, secret, metadata, private_metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = :secret AND status = %d`, clients.EnabledStatus)
	switch prefix {
	case authn.DomainAuth:
		q += " AND domain_id = :domain_id"
	case authn.BasicAuth:
		q += " AND id = :id"
	default:
		return clients.Client{}, repoerr.ErrNotFound
	}

	dbc := DBClient{
		Secret: key,
		Domain: id,
		ID:     id,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = DBClient{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		client, err := ToClient(dbc)
		if err != nil {
			return clients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return client, nil
	}

	return clients.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) Update(ctx context.Context, client clients.Client) (clients.Client, error) {
	var query []string
	var upq string
	if client.Name != "" {
		query = append(query, "name = :name,")
	}
	if client.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if client.PrivateMetadata != nil {
		query = append(query, "private_metadata = :private_metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, secret, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`,
		upq)
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateTags(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateIdentity(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, status, COALESCE(parent_group_id, '') AS parent_group_id, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateSecret(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) ChangeStatus(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, identity, metadata, private_metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo *clientRepo) RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (clients.Client, error) {
	query := `
	WITH selected_client AS (
		SELECT
			c.id,
			c.parent_group_id,
			COALESCE(g."path", ''::::ltree) AS parent_group_path,
			c.domain_id
		FROM
			clients c
		LEFT JOIN
			"groups" g ON c.parent_group_id = g.id
		WHERE
			c.id = :id
		LIMIT 1
	),
	selected_client_roles AS (
		SELECT
			cr.entity_id AS client_id,
			crm.member_id AS member_id,
			cr.id AS role_id,
			cr."name" AS role_name,
			jsonb_agg(DISTINCT cra."action") AS actions,
			'direct' AS access_type,
			''::::ltree AS access_provider_path,
			'' AS access_provider_id
		FROM
			clients_roles cr
		JOIN
			clients_role_members crm ON cr.id = crm.role_id
		JOIN
			clients_role_actions cra ON cr.id = cra.role_id
		JOIN
			selected_client sc ON sc.id = cr.entity_id
			AND crm.member_id = :member_id
		GROUP BY
			cr.entity_id, cr.id, cr.name, crm.member_id
	),
	selected_group_roles AS (
		SELECT
			sc.id AS client_id,
			grm.member_id AS member_id,
			gr.id AS role_id,
			gr."name" AS role_name,
			jsonb_agg(DISTINCT all_actions."action") AS actions,
			gr.entity_id AS access_provider_id,
			g."path" AS access_provider_path,
			CASE
				WHEN gr.entity_id = sc.parent_group_id
				THEN 'direct_group'
				ELSE 'indirect_group'
			END AS access_type
		FROM
			"groups" g
		JOIN
			groups_roles gr ON gr.entity_id = g.id
		JOIN
			groups_role_members grm ON gr.id = grm.role_id
		JOIN
			groups_role_actions gra ON gr.id = gra.role_id
		JOIN
			groups_role_actions all_actions ON gr.id = all_actions.role_id
		JOIN
			selected_client sc ON TRUE
		WHERE
			g."path" @> sc.parent_group_path
			AND grm.member_id = :member_id
			AND (
				(g.id = sc.parent_group_id AND gra."action" LIKE 'client%%')
				OR
				(g.id <> sc.parent_group_id AND gra."action" LIKE 'subgroup_client%%')
			)
		GROUP BY
			sc.id, sc.parent_group_id, gr.entity_id, gr.id, gr."name", g."path", grm.member_id
	),
	selected_domain_roles AS (
		SELECT
			sc.id AS client_id,
			drm.member_id AS member_id,
			dr.entity_id AS group_id,
			dr.id AS role_id,
			dr."name" AS role_name,
			jsonb_agg(DISTINCT all_actions."action") AS actions,
			''::::ltree access_provider_path,
			'domain' AS access_type,
			dr.entity_id AS access_provider_id
		FROM
			domains d
		JOIN
			selected_client sc ON sc.domain_id = d.id
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
			AND dra."action" LIKE 'client%%'
		GROUP BY
			sc.id, dr.entity_id, dr.id, dr."name", drm.member_id
	),
	all_roles AS (
		SELECT
			scr.client_id,
			scr.member_id,
			scr.role_id AS role_id,
			scr.role_name AS role_name,
			scr.actions AS actions,
			scr.access_type AS access_type,
			scr.access_provider_path AS access_provider_path,
			scr.access_provider_id AS access_provider_id
		FROM
			selected_client_roles scr
		UNION
		SELECT
			sgr.client_id,
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
			sdr.client_id,
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
			ar.client_id,
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
			ar.client_id, ar.member_id
	)
	SELECT
		c2.id,
		c2."name",
		c2.tags,
		COALESCE(c2.domain_id, '') AS domain_id,
		COALESCE(c2.parent_group_id, '') AS parent_group_id,
		c2."identity",
		c2.secret,
		c2.metadata,
		c2.created_at,
		c2.updated_at,
		c2.updated_by,
		c2.status,
		fr.member_id,
		fr.roles
	FROM clients c2
		JOIN final_roles fr ON fr.client_id = c2.id
	`
	parameters := map[string]any{
		"id":        id,
		"member_id": memberID,
	}
	row, err := repo.DB.NamedQueryContext(ctx, query, parameters)
	if err != nil {
		return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbc := DBClient{}
	if !row.Next() {
		return clients.Client{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbc); err != nil {
		return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	return ToClient(dbc)
}

func (repo *clientRepo) RetrieveByID(ctx context.Context, id string) (clients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, identity, secret, metadata, private_metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE id = :id`

	dbc := DBClient{
		ID: id,
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbc = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return clients.Client{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		return ToClient(dbc)
	}

	return clients.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) RetrieveAll(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	connJoinQuery := `
				FROM
					clients c
	`

	if pm.Channel != "" {
		connJoinQuery = `
			,conn.connection_types
			FROM
					clients c
			LEFT JOIN (
				SELECT
					conn.client_id,
					conn.channel_id,
					array_agg(conn."type") AS connection_types
				FROM
					connections AS conn
				GROUP BY
					conn.client_id, conn.channel_id
			) conn ON c.id = conn.client_id
		`
	}

	comQuery := fmt.Sprintf(`WITH clients AS (
				SELECT
					c.id,
					c.name,
					c.tags,
					c.identity,
					c.metadata,
					COALESCE(c.domain_id, '') AS domain_id,
					COALESCE(parent_group_id, '') AS parent_group_id,
					COALESCE((SELECT path FROM groups WHERE id = c.parent_group_id), ''::::ltree) AS parent_group_path,
					c.status,
					c.created_at,
					c.updated_at,
					COALESCE(c.updated_by, '') AS updated_by
				FROM
					clients c
			)
			SELECT
				c.*
			%s
			%s
		`, connJoinQuery, pageQuery)

	q := applyOrdering(comQuery, pm)

	q = applyLimitOffset(q)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	var items []clients.Client
	if !pm.OnlyTotal {
		rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
		defer rows.Close()

		for rows.Next() {
			dbc := DBClient{}
			if err := rows.StructScan(&dbc); err != nil {
				return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
			}

			c, err := ToClient(dbc)
			if err != nil {
				return clients.ClientsPage{}, err
			}

			items = append(items, c)
		}
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) AS total_count
			FROM (
				%s
			) AS sub_query;
			`, comQuery)

	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := clients.ClientsPage{
		Clients: items,
		Page: clients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) RetrieveUserClients(ctx context.Context, domainID, userID string, pm clients.Page) (clients.ClientsPage, error) {
	return repo.retrieveClients(ctx, domainID, userID, pm)
}

func (repo *clientRepo) retrieveClients(ctx context.Context, domainID, userID string, pm clients.Page) (clients.ClientsPage, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, err
	}

	bq := repo.userClientBaseQuery(domainID, userID)

	connJoinQuery := `
		FROM
			final_clients c
	`

	if pm.Channel != "" {
		connJoinQuery = `
			,conn.connection_types
			FROM
					final_clients c
			LEFT JOIN (
				SELECT
					conn.client_id,
					conn.channel_id,
					array_agg(conn."type") AS connection_types
				FROM
					connections AS conn
				GROUP BY
					conn.client_id, conn.channel_id
			) conn ON c.id = conn.client_id
		`
	}

	q := fmt.Sprintf(`
				%s
				SELECT
				  	c.id,
					c.name,
					c.domain_id,
					c.parent_group_id,
					c.identity,
					c.secret,
					c.tags,
					c.metadata,
					c.created_at,
					c.updated_at,
					c.updated_by,
					c.status,
					c.parent_group_path,
					c.role_id,
					c.role_name,
					c.actions,
					c.access_type,
					c.access_provider_id,
					c.access_provider_role_id,
					c.access_provider_role_name,
					c.access_provider_role_actions
				%s
				%s
	`, bq, connJoinQuery, pageQuery)

	q = applyOrdering(q, pm)

	q = applyLimitOffset(q)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	var items []clients.Client
	if !pm.OnlyTotal {
		rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}
		defer rows.Close()

		for rows.Next() {
			dbc := DBClient{}
			if err := rows.StructScan(&dbc); err != nil {
				return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
			}

			c, err := ToClient(dbc)
			if err != nil {
				return clients.ClientsPage{}, err
			}

			items = append(items, c)
		}
	}

	cq := fmt.Sprintf(`%s
						SELECT COUNT(*) AS total_count
						FROM (
							SELECT
								c.id,
								c.name,
								c.domain_id,
								c.parent_group_id,
								c.identity,
								c.secret,
								c.tags,
								c.metadata,
								c.created_at,
								c.updated_at,
								c.updated_by,
								c.status,
								c.parent_group_path,
								c.role_id,
								c.role_name,
								c.actions,
								c.access_type,
								c.access_provider_id,
								c.access_provider_role_id,
								c.access_provider_role_name,
								c.access_provider_role_actions
							%s
							%s
						) AS subquery;
			`, bq, connJoinQuery, pageQuery)

	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := clients.ClientsPage{
		Clients: items,
		Page: clients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) userClientBaseQuery(domainID, userID string) string {
	return fmt.Sprintf(`
	WITH direct_clients AS (
		SELECT
			c.id,
			c.name,
			c.domain_id,
			c.parent_group_id,
			c.tags,
			c.metadata,
			c.identity,
			c.secret,
			c.created_at,
			c.updated_at,
			c.updated_by,
			c.status,
			COALESCE((SELECT path FROM groups WHERE id = c.parent_group_id), ''::::ltree) AS parent_group_path,
			cr.id AS role_id,
			cr."name" AS role_name,
			array_agg(cra."action") AS actions,
			'direct' as access_type,
			'' AS access_provider_id,
			'' AS access_provider_role_id,
			'' AS access_provider_role_name,
			array[]::::text[] AS access_provider_role_actions
		FROM
			clients_role_members crm
		JOIN
			clients_role_actions cra ON cra.role_id = crm.role_id
		JOIN
			clients_roles cr ON cr.id = crm.role_id
		JOIN
			clients c ON c.id = cr.entity_id
		WHERE
			crm.member_id = '%s'
			AND c.domain_id = '%s'
		GROUP BY
			cr.entity_id, crm.member_id, cr.id, cr."name", c.id
	),
	direct_groups AS (
		SELECT
			g.*,
			gr.entity_id AS entity_id,
			grm.member_id AS member_id,
			gr.id AS role_id,
			gr."name" AS role_name,
			array_agg(DISTINCT all_actions."action") AS actions
		FROM
			groups_role_members grm
		JOIN
			groups_role_actions gra ON gra.role_id = grm.role_id
		JOIN
			groups_roles gr ON gr.id = grm.role_id
		JOIN
			"groups" g ON g.id = gr.entity_id
		JOIN
			groups_role_actions all_actions ON all_actions.role_id = grm.role_id
		WHERE
			grm.member_id = '%s'
			AND g.domain_id = '%s'
			AND gra."action" LIKE 'client%%'
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
			array_agg(DISTINCT all_actions."action") AS actions
		FROM
			groups_role_members grm
		JOIN
			groups_role_actions gra ON gra.role_id = grm.role_id
		JOIN
			groups_roles gr ON gr.id = grm.role_id
		JOIN
			"groups" g ON g.id = gr.entity_id
		JOIN
			groups_role_actions all_actions ON all_actions.role_id = grm.role_id
		WHERE
			grm.member_id = '%s'
			AND g.domain_id = '%s'
			AND gra."action" LIKE 'subgroup_client%%'
		GROUP BY
			gr.entity_id, grm.member_id, gr.id, gr."name", g."path", g.id
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
			AND NOT EXISTS ( 
			SELECT 1
			FROM direct_groups_with_subgroup dgws
			WHERE dgws.id = indirect_child_groups.id
		)
	),
	final_groups AS (
		SELECT
			id,
			parent_id,
			domain_id,
			"name",
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
			'direct_group' AS access_type,
			id AS access_provider_id,
			role_id AS access_provider_role_id,
			role_name AS access_provider_role_name,
			actions AS access_provider_role_actions
		FROM
			direct_groups
		UNION
		SELECT
			id,
			parent_id,
			domain_id,
			"name",
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
			'indirect_group' AS access_type,
			access_provider_id,
			access_provider_role_id,
			access_provider_role_name,
			access_provider_role_actions
		FROM
			indirect_child_groups
	),
	groups_clients AS (
		SELECT
			c.id,
			c.name,
			c.domain_id,
			c.parent_group_id,
			c.tags,
			c.metadata,
			c.identity,
			c.secret,
			c.created_at,
			c.updated_at,
			c.updated_by,
			c.status,
			g.path AS parent_group_path,
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
		JOIN
			clients c ON c.parent_group_id = g.id
		WHERE
			c.id NOT IN (SELECT id FROM direct_clients)
		UNION
		SELECT	* FROM   direct_clients
	),
	final_clients AS (
		SELECT
			gc.id,
			gc."name",
			gc.domain_id,
			gc.parent_group_id,
			gc.tags,
			gc.metadata,
			gc.identity,
			gc.secret,
			gc.created_at,
			gc.updated_at,
			gc.updated_by,
			gc.status,
			gc.parent_group_path,
			gc.role_id,
			gc.role_name,
			gc.actions,
			gc.access_type,
			gc.access_provider_id,
			gc.access_provider_role_id,
			gc.access_provider_role_name,
			gc.access_provider_role_actions
		FROM
			groups_clients AS  gc
		UNION
		SELECT
			dc.id,
			dc."name",
			dc.domain_id,
			dc.parent_group_id,
			dc.tags,
			dc.metadata,
			dc.identity,
			dc.secret,
			dc.created_at,
			dc.updated_at,
			dc.updated_by,
			dc.status,
			g."path" AS parent_group_path,
			'' AS role_id,
			'' AS role_name,
			array[]::::text[] AS actions,
			'domain' AS access_type,
			d.id AS access_provider_id,
			dr.id AS access_provider_role_id,
			dr."name" AS access_provider_role_name,
			array_agg(dra."action") as access_provider_role_actions
		FROM
			domains_role_members drm
		JOIN
			domains_role_actions dra ON dra.role_id = drm.role_id
		JOIN
			domains_roles dr ON dr.id = drm.role_id
		JOIN
			domains d ON d.id = dr.entity_id
		JOIN
			clients dc ON dc.domain_id = d.id
		LEFT JOIN
			groups g ON dc.parent_group_id = g.id
		WHERE
			drm.member_id = '%s' -- user_id
			 AND d.id = '%s' -- domain_id
			 AND dra."action" LIKE 'client_%%'
			 AND NOT EXISTS (  -- Ensures that the direct and indirect clients are not in included.
				SELECT 1 FROM groups_clients gc
				WHERE gc.id = dc.id
			)
		 GROUP BY
			dc.id, d.id, dr.id, g."path"
	)
	`, userID, domainID, userID, domainID, userID, domainID, domainID, userID, domainID)
}

func (repo *clientRepo) SearchClients(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.metadata, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []clients.Client
	for rows.Next() {
		dbc := DBClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		c, err := ToClient(dbc)
		if err != nil {
			return clients.ClientsPage{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, tq)
	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := clients.ClientsPage{
		Clients: items,
		Page: clients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) update(ctx context.Context, client clients.Client, query string) (clients.Client, error) {
	dbc, err := ToDBClient(client)
	if err != nil {
		return clients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return clients.Client{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbc = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return clients.Client{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
		}

		return ToClient(dbc)
	}

	return clients.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) Delete(ctx context.Context, clientIDs ...string) error {
	q := "DELETE FROM clients AS c  WHERE c.id = ANY(:client_ids) ;"

	params := map[string]any{
		"client_ids": clientIDs,
	}
	result, err := repo.DB.NamedExecContext(ctx, q, params)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

type DBClient struct {
	ID                        string           `db:"id"`
	Name                      string           `db:"name,omitempty"`
	Tags                      pgtype.TextArray `db:"tags,omitempty"`
	Identity                  string           `db:"identity"`
	Domain                    string           `db:"domain_id"`
	ParentGroup               sql.NullString   `db:"parent_group_id,omitempty"`
	Secret                    string           `db:"secret"`
	Metadata                  []byte           `db:"metadata,omitempty"`
	PrivateMetadata           []byte           `db:"private_metadata,omitempty"`
	CreatedAt                 time.Time        `db:"created_at,omitempty"`
	UpdatedAt                 sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy                 *string          `db:"updated_by,omitempty"`
	Status                    clients.Status   `db:"status,omitempty"`
	ParentGroupPath           sql.NullString   `db:"parent_group_path,omitempty"`
	RoleID                    string           `db:"role_id,omitempty"`
	RoleName                  string           `db:"role_name,omitempty"`
	Actions                   pq.StringArray   `db:"actions,omitempty"`
	AccessType                string           `db:"access_type,omitempty"`
	AccessProviderId          string           `db:"access_provider_id,omitempty"`
	AccessProviderRoleId      string           `db:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string           `db:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions pq.StringArray   `db:"access_provider_role_actions,omitempty"`
	ConnectionTypes           pq.Int32Array    `db:"connection_types,omitempty"`
	MemberID                  string           `db:"member_id,omitempty"`
	Roles                     json.RawMessage  `db:"roles,omitempty"`
}

func ToDBClient(c clients.Client) (DBClient, error) {
	privateMetadata := []byte("{}")
	if len(c.PrivateMetadata) > 0 {
		b, err := json.Marshal(c.PrivateMetadata)
		if err != nil {
			return DBClient{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		privateMetadata = b
	}
	metadata := []byte("{}")
	if len(c.Metadata) > 0 {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return DBClient{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		metadata = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(c.Tags); err != nil {
		return DBClient{}, err
	}
	var updatedBy *string
	if c.UpdatedBy != "" {
		updatedBy = &c.UpdatedBy
	}
	var updatedAt sql.NullTime
	if c.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: c.UpdatedAt, Valid: true}
	}

	return DBClient{
		ID:              c.ID,
		Name:            c.Name,
		Tags:            tags,
		Domain:          c.Domain,
		ParentGroup:     toNullString(c.ParentGroup),
		Identity:        c.Credentials.Identity,
		Secret:          c.Credentials.Secret,
		Metadata:        metadata,
		PrivateMetadata: privateMetadata,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       updatedAt,
		UpdatedBy:       updatedBy,
		Status:          c.Status,
	}, nil
}

func ToClient(t DBClient) (clients.Client, error) {
	var privateMetadata, metadata clients.Metadata
	if t.PrivateMetadata != nil {
		if err := json.Unmarshal([]byte(t.PrivateMetadata), &privateMetadata); err != nil {
			return clients.Client{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}
	if t.Metadata != nil {
		if err := json.Unmarshal([]byte(t.Metadata), &metadata); err != nil {
			return clients.Client{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	var tags []string
	for _, e := range t.Tags.Elements {
		tags = append(tags, e.String)
	}

	var updatedBy string
	if t.UpdatedBy != nil {
		updatedBy = *t.UpdatedBy
	}

	var updatedAt time.Time
	if t.UpdatedAt.Valid {
		updatedAt = t.UpdatedAt.Time.UTC()
	}

	var connTypes []connections.ConnType
	for _, ct := range t.ConnectionTypes {
		connType, err := connections.NewType(uint(ct))
		if err != nil {
			return clients.Client{}, err
		}
		connTypes = append(connTypes, connType)
	}

	var roles []roles.MemberRoleActions
	if t.Roles != nil {
		if err := json.Unmarshal(t.Roles, &roles); err != nil {
			return clients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	cli := clients.Client{
		ID:          t.ID,
		Name:        t.Name,
		Tags:        tags,
		Domain:      t.Domain,
		ParentGroup: toString(t.ParentGroup),
		Credentials: clients.Credentials{
			Identity: t.Identity,
			Secret:   t.Secret,
		},
		Metadata:                  metadata,
		PrivateMetadata:           privateMetadata,
		CreatedAt:                 t.CreatedAt.UTC(),
		UpdatedAt:                 updatedAt,
		UpdatedBy:                 updatedBy,
		Status:                    t.Status,
		ParentGroupPath:           toString(t.ParentGroupPath),
		RoleID:                    t.RoleID,
		RoleName:                  t.RoleName,
		Actions:                   t.Actions,
		AccessType:                t.AccessType,
		AccessProviderId:          t.AccessProviderId,
		AccessProviderRoleId:      t.AccessProviderRoleId,
		AccessProviderRoleName:    t.AccessProviderRoleName,
		AccessProviderRoleActions: t.AccessProviderRoleActions,
		ConnectionTypes:           connTypes,
		Roles:                     roles,
	}
	return cli, nil
}

func ToDBClientsPage(pm clients.Page) (dbClientsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	var tags pgtype.TextArray
	if err := tags.Set(pm.Tags.Elements); err != nil {
		return dbClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbClientsPage{
		Offset:      pm.Offset,
		Limit:       pm.Limit,
		Name:        pm.Name,
		Identity:    pm.Identity,
		Id:          pm.ID,
		Metadata:    data,
		Domain:      pm.Domain,
		Status:      pm.Status,
		Tags:        tags,
		GroupID:     pm.Group,
		ChannelID:   pm.Channel,
		RoleName:    pm.RoleName,
		ConnType:    pm.ConnectionType,
		RoleID:      pm.RoleID,
		Actions:     pm.Actions,
		AccessType:  pm.AccessType,
		CreatedFrom: pm.CreatedFrom,
		CreatedTo:   pm.CreatedTo,
	}, nil
}

type dbClientsPage struct {
	Limit       uint64           `db:"limit"`
	Offset      uint64           `db:"offset"`
	Name        string           `db:"name"`
	Id          string           `db:"id"`
	Domain      string           `db:"domain_id"`
	Identity    string           `db:"identity"`
	Metadata    []byte           `db:"metadata"`
	Tags        pgtype.TextArray `db:"tags"`
	Status      clients.Status   `db:"status"`
	GroupID     *string          `db:"group_id"`
	ChannelID   string           `db:"channel_id"`
	ConnType    string           `db:"type"`
	RoleName    string           `db:"role_name"`
	RoleID      string           `db:"role_id"`
	Actions     pq.StringArray   `db:"actions"`
	AccessType  string           `db:"access_type"`
	CreatedFrom time.Time        `db:"created_from"`
	CreatedTo   time.Time        `db:"created_to"`
}

func PageQuery(pm clients.Page) (string, error) {
	var query []string
	if pm.Name != "" {
		query = append(query, "c.name ILIKE '%' || :name || '%'")
	}
	if pm.Identity != "" {
		query = append(query, "c.identity ILIKE '%' || :identity || '%'")
	}
	if pm.ID != "" {
		query = append(query, "c.id = :id")
	}
	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case clients.AndOp:
			query = append(query, "tags @> :tags")
		default: // OR
			query = append(query, "tags && :tags")
		}
	}
	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("c.id IN ('%s')", strings.Join(pm.IDs, "','")))
	}

	if pm.Status != clients.AllStatus {
		query = append(query, "c.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "c.domain_id = :domain_id")
	}

	if pm.Group != nil {
		switch *pm.Group {
		case "":
			query = append(query, "c.parent_group_id = '' ")
		default:
			query = append(query, "c.parent_group_path <@ (SELECT path from groups where id = :group_id) ")
		}
	}

	if pm.Channel != "" {
		query = append(query, "conn.channel_id = :channel_id ")
		if pm.ConnectionType != "" {
			query = append(query, "conn.type = :conn_type ")
		}
	}
	if pm.AccessType != "" {
		query = append(query, "c.access_type = :access_type")
	}
	if pm.RoleID != "" {
		query = append(query, "c.role_id = :role_id")
	}
	if pm.RoleName != "" {
		query = append(query, "c.role_name = :role_name")
	}
	if len(pm.Actions) != 0 {
		query = append(query, "c.actions @> :actions")
	}
	if len(pm.Metadata) > 0 {
		query = append(query, "c.metadata @> :metadata")
	}

	if !pm.CreatedFrom.IsZero() {
		query = append(query, "c.created_at >= :created_from")
	}
	if !pm.CreatedTo.IsZero() {
		query = append(query, "c.created_at <= :created_to")
	}

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}
	return emq, nil
}

func applyOrdering(emq string, pm clients.Page) string {
	var orderBy string
	switch pm.Order {
	case "name":
		orderBy = "name"
	case "identity":
		orderBy = "identity"
	case "created_at":
		orderBy = "created_at"
	case "updated_at":
		orderBy = "COALESCE(updated_at, created_at)"
	default:
		return emq
	}

	if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
		return fmt.Sprintf("%s ORDER BY %s %s, id %s", emq, orderBy, pm.Dir, pm.Dir)
	}
	return fmt.Sprintf("%s ORDER BY %s", emq, orderBy)
}

func applyLimitOffset(query string) string {
	return fmt.Sprintf(`%s
			LIMIT :limit OFFSET :offset`, query)
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func toString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

func (repo *clientRepo) RetrieveByIds(ctx context.Context, ids []string) (clients.ClientsPage, error) {
	if len(ids) == 0 {
		return clients.ClientsPage{}, nil
	}

	pm := clients.Page{IDs: ids}
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id,  COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []clients.Client
	for rows.Next() {
		dbc := DBClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToClient(dbc)
		if err != nil {
			return clients.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := clients.ClientsPage{
		Clients: items,
		Page: clients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  total,
		},
	}

	return page, nil
}

func (repo *clientRepo) AddConnections(ctx context.Context, conns []clients.Connection) error {
	dbConns := toDBConnections(conns)
	q := `INSERT INTO connections (channel_id, domain_id, client_id, type)
			VALUES (:channel_id, :domain_id, :client_id, :type);`
	if _, err := repo.DB.NamedExecContext(ctx, q, dbConns); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *clientRepo) RemoveConnections(ctx context.Context, conns []clients.Connection) (retErr error) {
	tx, err := repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if retErr != nil {
			if errRollBack := tx.Rollback(); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollBack))
			}
		}
	}()

	query := `DELETE FROM connections WHERE channel_id = :channel_id AND domain_id = :domain_id AND client_id = :client_id`

	for _, conn := range conns {
		if uint8(conn.Type) > 0 {
			query = query + " AND type = :type "
		}
		dbConn := toDBConnection(conn)
		if _, err := tx.NamedExec(query, dbConn); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, errors.Wrap(fmt.Errorf("failed to delete connection for channel_id: %s, domain_id: %s client_id %s", conn.ChannelID, conn.DomainID, conn.ClientID), err))
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientRepo) SetParentGroup(ctx context.Context, cli clients.Client) error {
	q := "UPDATE clients SET parent_group_id = :parent_group_id, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"

	dbcli, err := ToDBClient(cli)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	result, err := repo.DB.NamedExecContext(ctx, q, dbcli)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (repo *clientRepo) RemoveParentGroup(ctx context.Context, cli clients.Client) error {
	q := "UPDATE clients SET parent_group_id = NULL, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"
	dbcli, err := ToDBClient(cli)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	result, err := repo.DB.NamedExecContext(ctx, q, dbcli)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (repo *clientRepo) ClientConnectionsCount(ctx context.Context, id string) (uint64, error) {
	query := `SELECT COUNT(*) FROM connections WHERE client_id = :client_id`
	dbConn := dbConnection{ClientID: id}

	total, err := postgres.Total(ctx, repo.DB, query, dbConn)
	if err != nil {
		return 0, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	return total, nil
}

func (repo *clientRepo) DoesClientHaveConnections(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM connections WHERE client_id = :client_id`
	dbConn := dbConnection{ClientID: id}

	rows, err := repo.DB.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return false, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (repo *clientRepo) RemoveChannelConnections(ctx context.Context, channelID string) error {
	query := `DELETE FROM connections WHERE channel_id = :channel_id`

	dbConn := dbConnection{ChannelID: channelID}
	if _, err := repo.DB.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientRepo) RemoveClientConnections(ctx context.Context, clientID string) error {
	query := `DELETE FROM connections WHERE client_id = :client_id`

	dbConn := dbConnection{ClientID: clientID}
	if _, err := repo.DB.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientRepo) RetrieveParentGroupClients(ctx context.Context, parentGroupID string) ([]clients.Client, error) {
	query := `SELECT c.id, c.name, c.tags,  c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c WHERE c.parent_group_id = :parent_group_id ;`

	rows, err := repo.DB.NamedQueryContext(ctx, query, DBClient{ParentGroup: toNullString(parentGroupID)})
	if err != nil {
		return []clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var clis []clients.Client
	for rows.Next() {
		dbCli := DBClient{}
		if err := rows.StructScan(&dbCli); err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		cli, err := ToClient(dbCli)
		if err != nil {
			return []clients.Client{}, err
		}

		clis = append(clis, cli)
	}
	return clis, nil
}

func (repo *clientRepo) UnsetParentGroupFromClient(ctx context.Context, parentGroupID string) error {
	query := "UPDATE clients SET parent_group_id = NULL WHERE parent_group_id = :parent_group_id"

	if _, err := repo.DB.NamedExecContext(ctx, query, DBClient{ParentGroup: toNullString(parentGroupID)}); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

type dbConnection struct {
	ClientID  string               `db:"client_id"`
	ChannelID string               `db:"channel_id"`
	DomainID  string               `db:"domain_id"`
	Type      connections.ConnType `db:"type"`
}

func toDBConnections(conns []clients.Connection) []dbConnection {
	var dbconns []dbConnection
	for _, conn := range conns {
		dbconns = append(dbconns, toDBConnection(conn))
	}
	return dbconns
}

func toDBConnection(conn clients.Connection) dbConnection {
	return dbConnection{
		ClientID:  conn.ClientID,
		ChannelID: conn.ChannelID,
		DomainID:  conn.DomainID,
		Type:      conn.Type,
	}
}
