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
	clients "github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/postgres"
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
	rolesPostgres.Repository
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewRepository(db postgres.Database) channels.Repository {
	rolesRepo := rolesPostgres.NewRepository(db, policies.ChannelType, rolesTableNamePrefix, entityTableName, entityIDColumnName)
	return &channelRepository{
		db:         db,
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

	q := `INSERT INTO channels (id, name, tags, domain_id, parent_group_id,  metadata, created_at, updated_at, updated_by, status)
	VALUES (:id, :name, :tags, :domain_id,  :parent_group_id, :metadata, :created_at, :updated_at, :updated_by, :status)
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`

	row, err := cr.db.NamedQueryContext(ctx, q, dbchs)
	if err != nil {
		return []channels.Channel{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	var reChs []channels.Channel

	for row.Next() {
		dbch := dbChannel{}
		if err := row.StructScan(&dbch); err != nil {
			return []channels.Channel{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
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
        RETURNING id, name, tags, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`,
		upq)
	channel.Status = clients.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) UpdateTags(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`
	channel.Status = clients.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) ChangeStatus(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`

	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) RetrieveByID(ctx context.Context, id string) (channels.Channel, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id,  metadata, created_at, updated_at, updated_by, status FROM channels WHERE id = :id`

	dbch := dbChannel{
		ID: id,
	}

	row, err := cr.db.NamedQueryContext(ctx, q, dbch)
	if err != nil {
		return channels.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbch = dbChannel{}
	if row.Next() {
		if err := row.StructScan(&dbch); err != nil {
			return channels.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		return toChannel(dbch)
	}

	return channels.Channel{}, repoerr.ErrNotFound
}

func (cr *channelRepository) RetrieveAll(ctx context.Context, pm channels.PageMetadata) (channels.Page, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
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
						*
					%s
					%s
					`, connJoinQuery, pageQuery)

	q := applyOrdering(comQuery, pm)

	q = applyLimitOffset(q)

	dbPage, err := toDBChannelsPage(pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := cr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []channels.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			return channels.Page{}, err
		}

		items = append(items, ch)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) AS total_count
			FROM (
				%s
			) AS sub_query;
			`, comQuery)

	total, err := postgres.Total(ctx, cr.db, cq, dbPage)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := channels.Page{
		Channels: items,
		PageMetadata: channels.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}
	return page, nil
}

func (repo *channelRepository) RetrieveUserChannels(ctx context.Context, domainID, userID string, pm channels.PageMetadata) (channels.Page, error) {
	return repo.retrieveClients(ctx, domainID, userID, pm)
}

func (repo *channelRepository) retrieveClients(ctx context.Context, domainID, userID string, pm channels.PageMetadata) (channels.Page, error) {
	pageQuery, err := PageQuery(pm)
	if err != nil {
		return channels.Page{}, err
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
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []channels.Channel
	for rows.Next() {
		dbc := dbChannel{}
		if err := rows.StructScan(&dbc); err != nil {
			return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := toChannel(dbc)
		if err != nil {
			return channels.Page{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`%s
						SELECT COUNT(*) AS total_count
						FROM (
							SELECT
								c.id,
								c.name,
								c.domain_id,
								c.parent_group_id,
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
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := channels.Page{
		Channels: items,
		PageMetadata: channels.PageMetadata{
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
indirect_child_groups AS (
	SELECT
		DISTINCT indirect_child_groups.id as child_id,
		indirect_child_groups.*,
		dgws.id as access_provider_id,
		dgws.role_id as access_provider_role_id,
		dgws.role_name as access_provider_role_name,
		dgws.actions as access_provider_role_actions
	FROM
		direct_groups_with_subgroup dgws
	JOIN
		groups indirect_child_groups ON indirect_child_groups.path <@ dgws.path
	WHERE
		indirect_child_groups.domain_id = '%s'
		AND NOT EXISTS (
			SELECT 1
			FROM (
				SELECT id FROM direct_groups_with_subgroup
				UNION ALL
				SELECT id FROM direct_groups
			) excluded
			WHERE excluded.id = indirect_child_groups.id
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
		dc.tags,
		dc.metadata,
		dc.created_by,
		dc.created_at,
		dc.updated_at,
		dc.updated_by,
		dc.status,
		text2ltree('') AS parent_group_path,
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
	WHERE
		drm.member_id = '%s' -- user_id
	 	AND d.id = '%s' -- domain_id
	 	AND dra."action" LIKE 'channel_%%'
	 	AND NOT EXISTS (  -- Ensures that the direct and indirect channels are not in included.
			SELECT 1 FROM groups_channels gc
			WHERE gc.id = dc.id
		)
	 GROUP BY
		dc.id, d.id, dr.id
)
	`, userID, domainID, userID, domainID, userID, domainID, domainID, userID, domainID)
}

func (cr *channelRepository) Remove(ctx context.Context, ids ...string) error {
	q := "DELETE FROM channels AS c  WHERE c.id = ANY(:channel_ids) ;"
	params := map[string]interface{}{
		"channel_ids": ids,
	}
	result, err := cr.db.NamedExecContext(ctx, q, params)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
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
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
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
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
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
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (cr *channelRepository) RemoveConnections(ctx context.Context, conns []channels.Connection) (retErr error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
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

func (cr *channelRepository) CheckConnection(ctx context.Context, conn channels.Connection) error {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id AND domain_id = :domain_id AND client_id = :client_id AND type = :type LIMIT 1`
	dbConn := toDBConnection(conn)
	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return repoerr.ErrNotFound
	}
	return nil
}

func (cr *channelRepository) ClientAuthorize(ctx context.Context, conn channels.Connection) error {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id AND client_id = :client_id AND type = :type LIMIT 1`
	dbConn := toDBConnection(conn)
	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return postgres.HandleError(repoerr.ErrViewEntity, err)
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
		return 0, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	return total, nil
}

func (cr *channelRepository) DoesChannelHaveConnections(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM connections WHERE channel_id = :channel_id`
	dbConn := dbConnection{ChannelID: id}

	rows, err := cr.db.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return false, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (cr *channelRepository) RemoveClientConnections(ctx context.Context, clientID string) error {
	query := `DELETE FROM connections WHERE client_id = :client_id`

	dbConn := dbConnection{ClientID: clientID}
	if _, err := cr.db.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) RemoveChannelConnections(ctx context.Context, channelID string) error {
	query := `DELETE FROM connections WHERE channel_id = :channel_id`

	dbConn := dbConnection{ChannelID: channelID}
	if _, err := cr.db.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (cr *channelRepository) RetrieveParentGroupChannels(ctx context.Context, parentGroupID string) ([]channels.Channel, error) {
	query := `SELECT c.id, c.name, c.tags,  c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_by, c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM channels c WHERE c.parent_group_id = :parent_group_id ;`

	rows, err := cr.db.NamedQueryContext(ctx, query, dbChannel{ParentGroup: toNullString(parentGroupID)})
	if err != nil {
		return []channels.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var chs []channels.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return []channels.Channel{}, errors.Wrap(repoerr.ErrViewEntity, err)
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
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
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
		return channels.Channel{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbch = dbChannel{}
	if row.Next() {
		if err := row.StructScan(&dbch); err != nil {
			return channels.Channel{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
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
	Metadata                  []byte           `db:"metadata,omitempty"`
	CreatedBy                 *string          `db:"created_by,omitempty"`
	CreatedAt                 time.Time        `db:"created_at,omitempty"`
	UpdatedAt                 sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy                 *string          `db:"updated_by,omitempty"`
	Status                    clients.Status   `db:"status,omitempty"`
	ParentGroupPath           string           `db:"parent_group_path,omitempty"`
	RoleID                    string           `db:"role_id,omitempty"`
	RoleName                  string           `db:"role_name,omitempty"`
	Actions                   pq.StringArray   `db:"actions,omitempty"`
	AccessType                string           `db:"access_type,omitempty"`
	AccessProviderId          string           `db:"access_provider_id,omitempty"`
	AccessProviderRoleId      string           `db:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string           `db:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions pq.StringArray   `db:"access_provider_role_actions,omitempty"`
	ConnectionTypes           pq.Int32Array    `db:"connection_types,omitempty"`
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
	var metadata clients.Metadata
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
		updatedAt = ch.UpdatedAt.Time
	}

	connTypes := []connections.ConnType{}
	for _, ct := range ch.ConnectionTypes {
		connType, err := connections.NewType(uint(ct))
		if err != nil {
			return channels.Channel{}, err
		}
		connTypes = append(connTypes, connType)
	}

	newCh := channels.Channel{
		ID:                        ch.ID,
		Name:                      ch.Name,
		Tags:                      tags,
		Domain:                    ch.Domain,
		ParentGroup:               toString(ch.ParentGroup),
		Metadata:                  metadata,
		CreatedBy:                 createdBy,
		CreatedAt:                 ch.CreatedAt,
		UpdatedAt:                 updatedAt,
		UpdatedBy:                 updatedBy,
		Status:                    ch.Status,
		ParentGroupPath:           ch.ParentGroupPath,
		RoleID:                    ch.RoleID,
		RoleName:                  ch.RoleName,
		Actions:                   ch.Actions,
		AccessType:                ch.AccessType,
		AccessProviderId:          ch.AccessProviderId,
		AccessProviderRoleId:      ch.AccessProviderRoleId,
		AccessProviderRoleName:    ch.AccessProviderRoleName,
		AccessProviderRoleActions: ch.AccessProviderRoleActions,
		ConnectionTypes:           connTypes,
	}

	return newCh, nil
}

func PageQuery(pm channels.PageMetadata) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var query []string
	if pm.Name != "" {
		query = append(query, "c.name ILIKE '%' || :name || '%'")
	}

	if pm.Id != "" {
		query = append(query, "c.id ILIKE '%' || :id || '%'")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}

	if mq != "" {
		query = append(query, mq)
	}

	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("id IN ('%s')", strings.Join(pm.IDs, "','")))
	}
	if pm.Status != clients.AllStatus {
		query = append(query, "c.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "c.domain_id = :domain_id")
	}
	if pm.Group != "" {
		query = append(query, "c.parent_group_path <@ (SELECT path from groups where id = :group_id) ")
	}
	if pm.Client != "" {
		query = append(query, "conn.client_id = :client_id ")
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

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}
	return emq, nil
}

func applyOrdering(emq string, pm channels.PageMetadata) string {
	switch pm.Order {
	case "name", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}

func applyLimitOffset(query string) string {
	return fmt.Sprintf(`%s
			LIMIT :limit OFFSET :offset`, query)
}

func toDBChannelsPage(pm channels.PageMetadata) (dbChannelsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbChannelsPage{
		Limit:      pm.Limit,
		Offset:     pm.Offset,
		Name:       pm.Name,
		Id:         pm.Id,
		Domain:     pm.Domain,
		Metadata:   data,
		Tag:        pm.Tag,
		Status:     pm.Status,
		GroupID:    pm.Group,
		ClientID:   pm.Client,
		ConnType:   pm.ConnectionType,
		RoleName:   pm.RoleName,
		RoleID:     pm.RoleID,
		Actions:    pm.Actions,
		AccessType: pm.AccessType,
	}, nil
}

type dbChannelsPage struct {
	Limit      uint64         `db:"limit"`
	Offset     uint64         `db:"offset"`
	Name       string         `db:"name"`
	Id         string         `db:"id"`
	Domain     string         `db:"domain_id"`
	Metadata   []byte         `db:"metadata"`
	Tag        string         `db:"tag"`
	Status     clients.Status `db:"status"`
	GroupID    string         `db:"group_id"`
	ClientID   string         `db:"client_id"`
	ConnType   string         `db:"type"`
	RoleName   string         `db:"role_name"`
	RoleID     string         `db:"role_id"`
	Actions    pq.StringArray `db:"actions"`
	AccessType string         `db:"access_type"`
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
