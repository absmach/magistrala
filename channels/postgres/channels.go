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
	"github.com/absmach/supermq/channels"
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
	rolesTableNamePrefix = "channels"
	entityTableName      = "channels"
	entityIDColumnName   = "id"
)

var _ channels.Repository = (*channelRepository)(nil)

type channelRepository struct {
	db postgres.Database
	eh errors.Handler
	rolesPostgres.Repository
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewRepository(db postgres.Database) channels.Repository {
	rolesRepo := rolesPostgres.NewRepository(db, policies.ChannelType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	errHandlerOptions := []errors.HandlerOption{
		postgres.WithDuplicateErrors(NewDuplicateErrors()),
	}
	return &channelRepository{
		db:         db,
		eh:         postgres.NewErrorHandler(errHandlerOptions...),
		Repository: rolesRepo,
	}
}

func (cr *channelRepository) Save(ctx context.Context, chs ...channels.Channel) ([]channels.Channel, error) {
	var dbchs []dbChannel
	for _, ch := range chs {
		dbch, err := toDBChannel(ch)
		if err != nil {
			return []channels.Channel{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		dbchs = append(dbchs, dbch)
	}

	q := `INSERT INTO channels (id, name, tags, domain_id, parent_group_id, route,  metadata, created_at, updated_at, updated_by, status)
	VALUES (:id, :name, :tags, :domain_id,  :parent_group_id, :route, :metadata, :created_at, :updated_at, :updated_by, :status)
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route, status, created_at, updated_at, updated_by`

	row, err := cr.db.NamedQueryContext(ctx, q, dbchs)
	if err != nil {
		return []channels.Channel{}, cr.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	var reChs []channels.Channel

	for row.Next() {
		dbch := dbChannel{}
		if err := row.StructScan(&dbch); err != nil {
			return []channels.Channel{}, cr.eh.HandleError(repoerr.ErrFailedOpDB, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			return []channels.Channel{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}
		reChs = append(reChs, ch)
	}
	return reChs, nil
}

func (cr *channelRepository) Update(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	var query []string
	var upq string
	if channel.Name != "" {
		query = append(query, "name = :name,")
	}
	if channel.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}
	q := fmt.Sprintf(`UPDATE channels SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route, status, created_at, updated_at, updated_by`,
		upq)
	channel.Status = channels.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) UpdateTags(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route, status, created_at, updated_at, updated_by`
	channel.Status = channels.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) ChangeStatus(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route, status, created_at, updated_at, updated_by`

	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) RetrieveByID(ctx context.Context, id string) (channels.Channel, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route,  metadata, created_at, updated_at, updated_by, status FROM channels WHERE id = :id`

	dbch := dbChannel{
		ID: id,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbch)
	if err != nil {
		return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbch = dbChannel{}
	if row.Next() {
		if err := row.StructScan(&dbch); err != nil {
			return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
		}
		return toChannel(dbch)
	}

	return channels.Channel{}, repoerr.ErrNotFound
}

func (cr *channelRepository) RetrieveByRoute(ctx context.Context, route, domainID string) (channels.Channel, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, route,  metadata, created_at, updated_at, updated_by, status
		FROM channels WHERE route = :route AND domain_id = :domain_id`

	dbch := dbChannel{
		Route:  toNullString(route),
		Domain: domainID,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbch)
	if err != nil {
		return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbch = dbChannel{}
	if row.Next() {
		if err := row.StructScan(&dbch); err != nil {
			return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
		}
		return toChannel(dbch)
	}

	return channels.Channel{}, repoerr.ErrNotFound
}

func (cr *channelRepository) RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (channels.Channel, error) {
	query := `
	WITH selected_channel AS (
		SELECT
			c.id,
			c.parent_group_id,
			COALESCE(g."path", ''::::ltree) AS parent_group_path,
			c.domain_id
		FROM
			channels c
		LEFT JOIN
			"groups" g ON c.parent_group_id = g.id
		WHERE
			c.id = :id
		LIMIT 1
	),
	selected_channel_roles AS (
		SELECT
			cr.entity_id AS channel_id,
			crm.member_id AS member_id,
			cr.id AS role_id,
			cr."name" AS role_name,
			jsonb_agg(DISTINCT cra."action") AS actions,
			'direct' AS access_type,
			''::::ltree AS access_provider_path,
			'' AS access_provider_id
		FROM
			channels_roles cr
		JOIN
			channels_role_members crm ON cr.id = crm.role_id
		JOIN
			channels_role_actions cra ON cr.id = cra.role_id
		JOIN
			selected_channel sc ON sc.id = cr.entity_id
			AND crm.member_id = :member_id
		GROUP BY
			cr.entity_id, cr.id, cr.name, crm.member_id
	),
	selected_group_roles AS (
		SELECT
			sc.id AS channel_id,
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
			selected_channel sc ON TRUE
		WHERE
			g."path" @> sc.parent_group_path
			AND grm.member_id = :member_id
			AND (
				(g.id = sc.parent_group_id AND gra."action" LIKE 'channel%%')
				OR
				(g.id <> sc.parent_group_id AND gra."action" LIKE 'subgroup_channel%%')
			)
		GROUP BY
			sc.id, sc.parent_group_id, gr.entity_id, gr.id, gr."name", g."path", grm.member_id
	),
	selected_domain_roles AS (
		SELECT
			sc.id AS channel_id,
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
			selected_channel sc ON sc.domain_id = d.id
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
			AND dra."action" LIKE 'channel%%'
		GROUP BY
			sc.id, dr.entity_id, dr.id, dr."name", drm.member_id
	),
	all_roles AS (
		SELECT
			scr.channel_id,
			scr.member_id,
			scr.role_id AS role_id,
			scr.role_name AS role_name,
			scr.actions AS actions,
			scr.access_type AS access_type,
			scr.access_provider_path AS access_provider_path,
			scr.access_provider_id AS access_provider_id
		FROM
			selected_channel_roles scr
		UNION
		SELECT
			sgr.channel_id,
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
			sdr.channel_id,
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
			ar.channel_id,
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
			ar.channel_id, ar.member_id
	)
	SELECT
		c2.id,
		c2."name",
		c2.tags,
		COALESCE(c2.domain_id, '') AS domain_id,
		COALESCE(c2.parent_group_id, '') AS parent_group_id,
		c2.route,
		c2.metadata,
		c2.created_at,
		c2.created_by,
		c2.updated_at,
		c2.updated_by,
		c2.status,
		fr.member_id,
		fr.roles
	FROM channels c2
		JOIN final_roles fr ON fr.channel_id = c2.id
	`
	parameters := map[string]any{
		"id":        id,
		"member_id": memberID,
	}
	row, err := cr.db.NamedQueryContext(ctx, query, parameters)
	if err != nil {
		return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbch := dbChannel{}
	if !row.Next() {
		return channels.Channel{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbch); err != nil {
		return channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	return toChannel(dbch)
}

func (cr *channelRepository) RetrieveAll(ctx context.Context, pm channels.Page) (channels.ChannelsPage, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return channels.ChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	connJoinQuery := `
				FROM
					channels c
	`

	if pm.Client != "" {
		connJoinQuery = `
			,conn.connection_types
			FROM
					channels c
			LEFT JOIN (
				SELECT
					conn.client_id,
					conn.channel_id,
					array_agg(conn."type") AS connection_types
				FROM
					connections AS conn
				GROUP BY
					conn.client_id, conn.channel_id
			) conn ON c.id = conn.channel_id
		`
	}

	comQuery := fmt.Sprintf(`WITH channels AS (
						SELECT
							c.id,
							c.name,
							c.tags,
							c.metadata,
							COALESCE(c.domain_id, '') AS domain_id,
							COALESCE(parent_group_id, '') AS parent_group_id,
							c.route,
							COALESCE((SELECT path FROM groups WHERE id = c.parent_group_id), ''::::ltree) AS parent_group_path,
							c.status,
							c.created_by,
							c.created_at,
							c.updated_at,
							COALESCE(c.updated_by, '') AS updated_by
					    FROM
        					channels c
					)
					SELECT
						c.*
					%s
					%s
					`, connJoinQuery, pageQuery)

	q := applyOrdering(comQuery, pm)

	q = applyLimitOffset(q)

	dbPage, err := toDBChannelsPage(pm)
	if err != nil {
		return channels.ChannelsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	var items []channels.Channel
	if !pm.OnlyTotal {
		rows, err := cr.db.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return channels.ChannelsPage{}, cr.eh.HandleError(repoerr.ErrFailedToRetrieveAllGroups, err)
		}
		defer rows.Close()

		for rows.Next() {
			dbch := dbChannel{}
			if err := rows.StructScan(&dbch); err != nil {
				return channels.ChannelsPage{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
			}

			ch, err := toChannel(dbch)
			if err != nil {
				return channels.ChannelsPage{}, err
			}

			items = append(items, ch)
		}
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) AS total_count
			FROM (
				%s
			) AS sub_query;
			`, comQuery)

	total, err := postgres.Total(ctx, cr.db, cq, dbPage)
	if err != nil {
		return channels.ChannelsPage{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := channels.ChannelsPage{
		Channels: items,
		Page: channels.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}
	return page, nil
}

func (repo *channelRepository) RetrieveUserChannels(ctx context.Context, domainID, userID string, pm channels.Page) (channels.ChannelsPage, error) {
	return repo.retrieveChannels(ctx, domainID, userID, pm)
}

func (repo *channelRepository) retrieveChannels(ctx context.Context, domainID, userID string, pm channels.Page) (channels.ChannelsPage, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return channels.ChannelsPage{}, err
	}

	bq := repo.userChannelsBaseQuery(domainID, userID)

	connJoinQuery := `
		FROM
			final_channels c
	`

	if pm.Client != "" {
		connJoinQuery = `
			,conn.connection_types
			FROM
					final_channels c
			LEFT JOIN (
				SELECT
					conn.client_id,
					conn.channel_id,
					array_agg(conn."type") AS connection_types
				FROM
					connections AS conn
				GROUP BY
					conn.client_id, conn.channel_id
			) conn ON c.id = conn.channel_id
		`
	}

	q := fmt.Sprintf(`
				%s
				SELECT
				  	c.id,
					c.name,
					c.domain_id,
					c.parent_group_id,
					c.route,
					c.tags,
					c.metadata,
					c.created_by,
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

	dbPage, err := toDBChannelsPage(pm)
	if err != nil {
		return channels.ChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	var items []channels.Channel
	if !pm.OnlyTotal {
		rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return channels.ChannelsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}
		defer rows.Close()

		for rows.Next() {
			dbc := dbChannel{}
			if err := rows.StructScan(&dbc); err != nil {
				return channels.ChannelsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
			}

			c, err := toChannel(dbc)
			if err != nil {
				return channels.ChannelsPage{}, err
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
								c.route,
								c.tags,
								c.metadata,
								c.created_by,
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

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return channels.ChannelsPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := channels.ChannelsPage{
		Channels: items,
		Page: channels.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *channelRepository) userChannelsBaseQuery(domainID, userID string) string {
	return fmt.Sprintf(`
WITH direct_channels AS (
	select
		c.id,
		c.name,
		c.domain_id,
		c.parent_group_id,
		c.route,
		c.tags,
		c.metadata,
		c.created_by,
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
		channels_role_members crm
	JOIN
		channels_role_actions cra ON cra.role_id = crm.role_id
	JOIN
		channels_roles cr ON cr.id = crm.role_id
	JOIN
		channels c ON c.id = cr.entity_id
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
		AND gra."action" LIKE 'channel%%'
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
		AND gra."action" LIKE 'subgroup_channel%%'
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
		groups indirect_child_groups ON indirect_child_groups.path <@ dlgws.path
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
groups_channels AS (
	SELECT
		c.id,
		c.name,
		c.domain_id,
		c.parent_group_id,
		c.route,
		c.tags,
		c.metadata,
		c.created_by,
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
		channels c ON c.parent_group_id = g.id
	WHERE
		c.id NOT IN (SELECT id FROM direct_channels)
	UNION
	SELECT	* FROM   direct_channels
),
final_channels AS (
	SELECT
		gc.id,
		gc."name",
		gc.domain_id,
		gc.parent_group_id,
		gc.route,
		gc.tags,
		gc.metadata,
		gc.created_by,
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
		groups_channels AS  gc
	UNION
	SELECT
		dc.id,
		dc."name",
		dc.domain_id,
		dc.parent_group_id,
		dc.route,
		dc.tags,
		dc.metadata,
		dc.created_by,
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
		channels dc ON dc.domain_id = d.id
	LEFT JOIN
		groups g ON dc.parent_group_id = g.id
	WHERE
		drm.member_id = '%s' -- user_id
	 	AND d.id = '%s' -- domain_id
	 	AND dra."action" LIKE 'channel_%%'
	 	AND NOT EXISTS (  -- Ensures that the direct and indirect channels are not in included.
			SELECT 1 FROM groups_channels gc
			WHERE gc.id = dc.id
		)
	 GROUP BY
		dc.id, d.id, dr.id, g."path"
)
	`, userID, domainID, userID, domainID, userID, domainID, domainID, userID, domainID)
}

func (cr *channelRepository) Remove(ctx context.Context, ids ...string) error {
	q := "DELETE FROM channels AS c  WHERE c.id = ANY(:channel_ids) ;"
	params := map[string]any{
		"channel_ids": ids,
	}
	result, err := cr.db.NamedExecContext(ctx, q, params)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) SetParentGroup(ctx context.Context, ch channels.Channel) error {
	q := "UPDATE channels SET parent_group_id = :parent_group_id, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"
	dbCh, err := toDBChannel(ch)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	result, err := cr.db.NamedExecContext(ctx, q, dbCh)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) RemoveParentGroup(ctx context.Context, ch channels.Channel) error {
	q := "UPDATE channels SET parent_group_id = NULL, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"
	dbCh, err := toDBChannel(ch)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	result, err := cr.db.NamedExecContext(ctx, q, dbCh)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) AddConnections(ctx context.Context, conns []channels.Connection) error {
	dbConns := toDBConnections(conns)
	q := `INSERT INTO connections (channel_id, domain_id, client_id, type)
			VALUES (:channel_id, :domain_id, :client_id, :type );`

	if _, err := cr.db.NamedExecContext(ctx, q, dbConns); err != nil {
		return cr.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (cr *channelRepository) RemoveConnections(ctx context.Context, conns []channels.Connection) (retErr error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
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
			return cr.eh.HandleError(repoerr.ErrRemoveEntity, errors.Wrap(fmt.Errorf("failed to delete connection for channel_id: %s, domain_id: %s client_id %s", conn.ChannelID, conn.DomainID, conn.ClientID), err))
		}
	}
	if err := tx.Commit(); err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) CheckConnection(ctx context.Context, conn channels.Connection) error {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id AND domain_id = :domain_id AND client_id = :client_id AND type = :type LIMIT 1`
	dbConn := toDBConnection(conn)
	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) ClientAuthorize(ctx context.Context, conn channels.Connection) error {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id AND client_id = :client_id AND domain_id = :domain_id AND type = :type LIMIT 1`
	dbConn := toDBConnection(conn)
	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) ChannelConnectionsCount(ctx context.Context, id string) (uint64, error) {
	query := `SELECT COUNT(*) FROM connections WHERE channel_id = :channel_id`
	dbConn := dbConnection{ChannelID: id}

	total, err := postgres.Total(ctx, cr.db, query, dbConn)
	if err != nil {
		return 0, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	return total, nil
}

func (cr *channelRepository) DoesChannelHaveConnections(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id`
	dbConn := dbConnection{ChannelID: id}

	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return false, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (cr *channelRepository) RemoveClientConnections(ctx context.Context, clientID string) error {
	query := `DELETE FROM connections WHERE client_id = :client_id`

	dbConn := dbConnection{ClientID: clientID}
	if _, err := cr.db.NamedExecContext(ctx, query, dbConn); err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) RemoveChannelConnections(ctx context.Context, channelID string) error {
	query := `DELETE FROM connections WHERE channel_id = :channel_id`

	dbConn := dbConnection{ChannelID: channelID}
	if _, err := cr.db.NamedExecContext(ctx, query, dbConn); err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) RetrieveParentGroupChannels(ctx context.Context, parentGroupID string) ([]channels.Channel, error) {
	query := `SELECT c.id, c.name, c.tags,  c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_by, c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM channels c WHERE c.parent_group_id = :parent_group_id ;`

	rows, err := cr.db.NamedQueryContext(ctx, query, dbChannel{ParentGroup: toNullString(parentGroupID)})
	if err != nil {
		return []channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var chs []channels.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return []channels.Channel{}, cr.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			return []channels.Channel{}, err
		}

		chs = append(chs, ch)
	}
	return chs, nil
}

func (cr *channelRepository) UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) error {
	query := "UPDATE channels SET parent_group_id = NULL WHERE parent_group_id = :parent_group_id"

	if _, err := cr.db.NamedExecContext(ctx, query, dbChannel{ParentGroup: toNullString(parentGroupID)}); err != nil {
		return cr.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) update(ctx context.Context, ch channels.Channel, query string) (channels.Channel, error) {
	dbch, err := toDBChannel(ch)
	if err != nil {
		return channels.Channel{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := cr.db.NamedQueryContext(ctx, query, dbch)
	if err != nil {
		return channels.Channel{}, cr.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbch = dbChannel{}
	if row.Next() {
		if err := row.StructScan(&dbch); err != nil {
			return channels.Channel{}, cr.eh.HandleError(repoerr.ErrUpdateEntity, err)
		}

		return toChannel(dbch)
	}

	return channels.Channel{}, repoerr.ErrNotFound
}

type dbChannel struct {
	ID                        string           `db:"id"`
	Name                      string           `db:"name,omitempty"`
	ParentGroup               sql.NullString   `db:"parent_group_id,omitempty"`
	Tags                      pgtype.TextArray `db:"tags,omitempty"`
	Domain                    string           `db:"domain_id"`
	Route                     sql.NullString   `db:"route,omitempty"`
	Metadata                  []byte           `db:"metadata,omitempty"`
	CreatedBy                 *string          `db:"created_by,omitempty"`
	CreatedAt                 time.Time        `db:"created_at,omitempty"`
	UpdatedAt                 sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy                 *string          `db:"updated_by,omitempty"`
	Status                    channels.Status  `db:"status,omitempty"`
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

func toDBChannel(ch channels.Channel) (dbChannel, error) {
	data := []byte("{}")
	if len(ch.Metadata) > 0 {
		b, err := json.Marshal(ch.Metadata)
		if err != nil {
			return dbChannel{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(ch.Tags); err != nil {
		return dbChannel{}, err
	}
	var createdBy *string
	if ch.CreatedBy != "" {
		createdBy = &ch.CreatedBy
	}
	var updatedBy *string
	if ch.UpdatedBy != "" {
		updatedBy = &ch.UpdatedBy
	}
	var updatedAt sql.NullTime
	if ch.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: ch.UpdatedAt, Valid: true}
	}
	return dbChannel{
		ID:          ch.ID,
		Name:        ch.Name,
		ParentGroup: toNullString(ch.ParentGroup),
		Domain:      ch.Domain,
		Route:       toNullString(ch.Route),
		Tags:        tags,
		Metadata:    data,
		CreatedBy:   createdBy,
		CreatedAt:   ch.CreatedAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		Status:      ch.Status,
	}, nil
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

func toChannel(ch dbChannel) (channels.Channel, error) {
	var metadata channels.Metadata
	if ch.Metadata != nil {
		if err := json.Unmarshal([]byte(ch.Metadata), &metadata); err != nil {
			return channels.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range ch.Tags.Elements {
		tags = append(tags, e.String)
	}
	var createdBy string
	if ch.CreatedBy != nil {
		createdBy = *ch.CreatedBy
	}
	var updatedBy string
	if ch.UpdatedBy != nil {
		updatedBy = *ch.UpdatedBy
	}
	var updatedAt time.Time
	if ch.UpdatedAt.Valid {
		updatedAt = ch.UpdatedAt.Time.UTC()
	}

	connTypes := []connections.ConnType{}
	for _, ct := range ch.ConnectionTypes {
		connType, err := connections.NewType(uint(ct))
		if err != nil {
			return channels.Channel{}, err
		}
		connTypes = append(connTypes, connType)
	}

	var roles []roles.MemberRoleActions
	if ch.Roles != nil {
		if err := json.Unmarshal(ch.Roles, &roles); err != nil {
			return channels.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	newCh := channels.Channel{
		ID:                        ch.ID,
		Name:                      ch.Name,
		Tags:                      tags,
		Domain:                    ch.Domain,
		Route:                     toString(ch.Route),
		ParentGroup:               toString(ch.ParentGroup),
		Metadata:                  metadata,
		CreatedBy:                 createdBy,
		CreatedAt:                 ch.CreatedAt.UTC(),
		UpdatedAt:                 updatedAt,
		UpdatedBy:                 updatedBy,
		Status:                    ch.Status,
		ParentGroupPath:           toString(ch.ParentGroupPath),
		RoleID:                    ch.RoleID,
		RoleName:                  ch.RoleName,
		Actions:                   ch.Actions,
		AccessType:                ch.AccessType,
		AccessProviderId:          ch.AccessProviderId,
		AccessProviderRoleId:      ch.AccessProviderRoleId,
		AccessProviderRoleName:    ch.AccessProviderRoleName,
		AccessProviderRoleActions: ch.AccessProviderRoleActions,
		ConnectionTypes:           connTypes,
		Roles:                     roles,
	}

	return newCh, nil
}

func PageQuery(pm channels.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var query []string
	if pm.Name != "" {
		query = append(query, "c.name ILIKE '%' || :name || '%'")
	}

	if pm.ID != "" {
		query = append(query, "c.id = :id")
	}
	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case channels.AndOp:
			query = append(query, "tags @> :tags")
		default: // OR
			query = append(query, "tags && :tags")
		}
	}

	if mq != "" {
		query = append(query, mq)
	}

	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("id IN ('%s')", strings.Join(pm.IDs, "','")))
	}
	if pm.Status != channels.AllStatus {
		query = append(query, "c.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "c.domain_id = :domain_id")
	}
	if pm.Group.Valid {
		switch {
		case pm.Group.Value != "":
			query = append(query, "c.parent_group_path <@ (SELECT path from groups where id = :group_id) ")
		default:
			query = append(query, "c.parent_group_id = '' ")
		}
	}

	if pm.Client != "" {
		query = append(query, "conn.client_id = :client_id ")
		if pm.ConnectionType != "" {
			query = append(query, ":conn_type = ANY(conn.connection_types) ")
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

func applyOrdering(emq string, pm channels.Page) string {
	col := "COALESCE(c.updated_at, c.created_at)"
	switch pm.Order {
	case "name":
		col = "c.name"
	case "created_at":
		col = "c.created_at"
	case "updated_at", "":
		col = "COALESCE(c.updated_at, c.created_at)"
	}

	dir := pm.Dir
	if dir != api.AscDir && dir != api.DescDir {
		dir = api.DescDir
	}

	return fmt.Sprintf("%s ORDER BY %s %s, c.id %s", emq, col, dir, dir)
}

func applyLimitOffset(query string) string {
	return fmt.Sprintf(`%s
			LIMIT :limit OFFSET :offset`, query)
}

func toDBChannelsPage(pm channels.Page) (dbChannelsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	var tags pgtype.TextArray
	if err := tags.Set(pm.Tags.Elements); err != nil {
		return dbChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	var connType uint8
	if pm.ConnectionType != "" {
		ct, err := connections.ParseConnType(pm.ConnectionType)
		if err != nil {
			return dbChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		connType = uint8(ct)
	}

	return dbChannelsPage{
		Limit:       pm.Limit,
		Offset:      pm.Offset,
		Name:        pm.Name,
		Id:          pm.ID,
		Domain:      pm.Domain,
		Metadata:    data,
		Tags:        tags,
		Status:      pm.Status,
		GroupID:     sql.NullString{Valid: pm.Group.Valid, String: pm.Group.Value},
		ClientID:    pm.Client,
		ConnType:    connType,
		RoleName:    pm.RoleName,
		RoleID:      pm.RoleID,
		Actions:     pm.Actions,
		AccessType:  pm.AccessType,
		CreatedFrom: pm.CreatedFrom,
		CreatedTo:   pm.CreatedTo,
	}, nil
}

type dbChannelsPage struct {
	Limit       uint64           `db:"limit"`
	Offset      uint64           `db:"offset"`
	Name        string           `db:"name"`
	Id          string           `db:"id"`
	Domain      string           `db:"domain_id"`
	Metadata    []byte           `db:"metadata"`
	Tags        pgtype.TextArray `db:"tags"`
	Status      channels.Status  `db:"status"`
	GroupID     sql.NullString   `db:"group_id"`
	ClientID    string           `db:"client_id"`
	ConnType    uint8            `db:"conn_type"`
	RoleName    string           `db:"role_name"`
	RoleID      string           `db:"role_id"`
	Actions     pq.StringArray   `db:"actions"`
	AccessType  string           `db:"access_type"`
	CreatedFrom time.Time        `db:"created_from"`
	CreatedTo   time.Time        `db:"created_to"`
}

type dbConnection struct {
	ChannelID string               `db:"channel_id"`
	DomainID  string               `db:"domain_id"`
	ClientID  string               `db:"client_id"`
	Type      connections.ConnType `db:"type"`
}

func toDBConnections(conns []channels.Connection) []dbConnection {
	var dbconns []dbConnection
	for _, conn := range conns {
		dbconns = append(dbconns, toDBConnection(conn))
	}
	return dbconns
}

func toDBConnection(conn channels.Connection) dbConnection {
	return dbConnection{
		ClientID:  conn.ClientID,
		ChannelID: conn.ChannelID,
		DomainID:  conn.DomainID,
		Type:      conn.Type,
	}
}
