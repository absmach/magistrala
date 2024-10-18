// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/channels"
	"github.com/absmach/magistrala/pkg/clients"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	clientpg "github.com/absmach/magistrala/pkg/clients/postgres"
	entityRolesRepo "github.com/absmach/magistrala/pkg/entityroles/postrgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgtype"
)

const rolesTableNamePrefix = "channels"

var _ channels.Repository = (*channelRepository)(nil)

type channelRepository struct {
	db postgres.Database
	entityRolesRepo.RolesSvcRepo
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewRepository(db postgres.Database) channels.Repository {

	rolesSvcRepo := entityRolesRepo.NewRolesSvcRepository(db, rolesTableNamePrefix)
	return &channelRepository{
		db:           db,
		RolesSvcRepo: rolesSvcRepo,
	}
}

func (cr *channelRepository) Save(ctx context.Context, chs ...channels.Channel) (retChs []channels.Channel, retErr error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []channels.Channel{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()
	var resChs []channels.Channel
	for _, ch := range chs {
		q := `INSERT INTO channels (id, name, tags, domain_id,  metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :domain_id,  :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`
		dbch, err := toDBChannel(ch)
		if err != nil {
			return []channels.Channel{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		row, err := tx.NamedQuery(q, dbch)
		if err != nil {
			return []channels.Channel{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
		}
		defer row.Close()
		if row.Next() {
			dbch = dbChannel{}
			if err := row.StructScan(&dbch); err != nil {
				return []channels.Channel{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}

			client, err := toChannel(dbch)
			if err != nil {
				return []channels.Channel{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}
			resChs = append(resChs, client)
		}
	}
	if err = tx.Commit(); err != nil {
		return []channels.Channel{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return resChs, nil
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
        RETURNING id, name, tags, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`,
		upq)
	channel.Status = clients.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) UpdateTags(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
	WHERE id = :id AND status = :status
	RETURNING id, name, tags,  metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`
	channel.Status = clients.EnabledStatus
	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) ChangeStatus(ctx context.Context, channel channels.Channel) (channels.Channel, error) {
	q := `UPDATE channels SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`

	return cr.update(ctx, channel, q)
}

func (cr *channelRepository) RetrieveByID(ctx context.Context, id string) (channels.Channel, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id,  metadata, created_at, updated_at, updated_by, status FROM channels WHERE id = :id`

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
	query, err := PageQuery(pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

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

func (cr *channelRepository) RetrieveByThing(ctx context.Context, thID string, pm channels.PageMetadata) (channels.Page, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c JOIN connections conn ON conn.channel_id = c.id  %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c JOIN connections conn ON conn.channel_id = c.id %s;`, query)

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

func (cr *channelRepository) Connect(ctx context.Context, chIDs, thIDs []string) (retErr error) {
	cq := `SELECT id, COALESCE(domain_id, '') AS domain_id, status FROM channels WHERE id IN ANY($1::text[]);`
	rows, err := cr.db.QueryxContext(ctx, cq, chIDs)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer rows.Close()

	var chs []dbChannel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		if dbch.Status != mgclients.EnabledStatus {
			return fmt.Errorf("channel %s is not in enabled status", dbch.ID)
		}
		chs = append(chs, dbch)
	}

	tq := `SELECT id, COALESCE(domain_id, '') AS domain_id, status FROM clients WHERE id IN ANY($1::text[]);`
	rows, err = cr.db.QueryxContext(ctx, tq, chIDs)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer rows.Close()

	var things []clientpg.DBClient
	for rows.Next() {
		dbcli := clientpg.DBClient{}
		if err := rows.StructScan(&dbcli); err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		if dbcli.Status != mgclients.EnabledStatus {
			return fmt.Errorf("thing %s is not in enabled status", dbcli.ID)
		}
		things = append(things, dbcli)
	}

	query := `INSERT INTO connections (channel_id, domain_id, thing_id)
	VALUES (:channel_id, :domain_id, :thing_id)`

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	for _, ch := range chs {
		for _, thing := range things {
			conn := dbConnection{
				ChannelID: ch.ID,
				DomainID:  ch.Domain,
				ThingID:   thing.ID,
			}
			_, err := tx.NamedExec(query, conn)
			if err != nil {
				retErr := errors.Wrap(repoerr.ErrCreateEntity, errors.Wrap(fmt.Errorf("failed to insert connection for channel_id: %s, domain_id: %s thing_id %s", conn.ChannelID, conn.DomainID, conn.ThingID), err))
				if errRollBack := tx.Rollback(); errRollBack != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollBack))
				}
				return retErr
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

func (cr *channelRepository) Disconnect(ctx context.Context, chIDs, thIDs []string) error {
	cq := `SELECT id, COALESCE(domain_id, '') AS domain_id, status FROM channels WHERE id IN ANY($1::text[]);`
	rows, err := cr.db.QueryxContext(ctx, cq, chIDs)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer rows.Close()

	var chs []dbChannel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
		chs = append(chs, dbch)
	}

	query := `DELETE FROM connections WHERE channel_id = :channel_id AND domain_id = :domain_id AND thing_id = :thing_id`

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	for _, ch := range chs {
		for _, thID := range thIDs {
			conn := dbConnection{
				ChannelID: ch.ID,
				DomainID:  ch.Domain,
				ThingID:   thID,
			}
			_, err := tx.NamedExec(query, conn)
			if err != nil {
				retErr := errors.Wrap(repoerr.ErrRemoveEntity, errors.Wrap(fmt.Errorf("failed to delete connection for channel_id: %s, domain_id: %s thing_id %s", conn.ChannelID, conn.DomainID, conn.ThingID), err))
				if errRollBack := tx.Rollback(); errRollBack != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollBack))
				}
				return retErr
			}
		}
	}
	if err := tx.Commit(); err != nil {
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
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Domain    string           `db:"domain_id"`
	Metadata  []byte           `db:"metadata,omitempty"`
	CreatedAt time.Time        `db:"created_at,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    clients.Status   `db:"status,omitempty"`
	Role      *clients.Role    `db:"role,omitempty"`
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
	var updatedBy *string
	if ch.UpdatedBy != "" {
		updatedBy = &ch.UpdatedBy
	}
	var updatedAt sql.NullTime
	if ch.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: ch.UpdatedAt, Valid: true}
	}
	return dbChannel{
		ID:        ch.ID,
		Name:      ch.Name,
		Domain:    ch.Domain,
		Tags:      tags,
		Metadata:  data,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    ch.Status,
	}, nil
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
	var updatedBy string
	if ch.UpdatedBy != nil {
		updatedBy = *ch.UpdatedBy
	}
	var updatedAt time.Time
	if ch.UpdatedAt.Valid {
		updatedAt = ch.UpdatedAt.Time
	}

	newCh := channels.Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Tags:      tags,
		Domain:    ch.Domain,
		Metadata:  metadata,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    ch.Status,
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

	if pm.ThingID != "" {
		query = append(query, "conn.thing_id = :thing_id")
	}
	if pm.Id != "" {
		query = append(query, "c.id ILIKE '%' || :id || '%'")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}

	// If there are search params presents, use search and ignore other options.
	// Always combine role with search params, so len(query) > 1.
	if len(query) > 1 {
		return fmt.Sprintf("WHERE %s", strings.Join(query, " AND ")), nil
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
	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}
	return emq, nil
}

func applyOrdering(emq string, pm channels.PageMetadata) string {
	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}

func toDBChannelsPage(pm channels.PageMetadata) (dbChannelsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbChannelsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbChannelsPage{
		Name:     pm.Name,
		Id:       pm.Id,
		Metadata: data,
		Domain:   pm.Domain,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		Status:   pm.Status,
		Tag:      pm.Tag,
	}, nil
}

type dbChannelsPage struct {
	Total    uint64         `db:"total"`
	Limit    uint64         `db:"limit"`
	Offset   uint64         `db:"offset"`
	Name     string         `db:"name"`
	Id       string         `db:"id"`
	Domain   string         `db:"domain_id"`
	Metadata []byte         `db:"metadata"`
	Tag      string         `db:"tag"`
	Status   clients.Status `db:"status"`
}

type dbConnection struct {
	ChannelID string `db:"channel_id"`
	DomainID  string `db:"domain_id"`
	ThingID   string `db:"thing_id"`
}
