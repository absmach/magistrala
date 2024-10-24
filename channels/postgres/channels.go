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

	"github.com/absmach/magistrala/channels"
	clients "github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	rolesPostgres "github.com/absmach/magistrala/pkg/roles/repo/postgres"
	"github.com/jackc/pgtype"
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
	rolesRepo := rolesPostgres.NewRepository(db, rolesTableNamePrefix, entityTableName, entityIDColumnName)
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
	query, err := PageQuery(pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags,  c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM channels c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM channels c %s;`, query)

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
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM channels c WHERE c.parent_group_id = :parent_group_id ;`

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
	ID          string           `db:"id"`
	Name        string           `db:"name,omitempty"`
	ParentGroup sql.NullString   `db:"parent_group_id,omitempty"`
	Tags        pgtype.TextArray `db:"tags,omitempty"`
	Domain      string           `db:"domain_id"`
	Metadata    []byte           `db:"metadata,omitempty"`
	CreatedAt   time.Time        `db:"created_at,omitempty"`
	UpdatedAt   sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy   *string          `db:"updated_by,omitempty"`
	Status      clients.Status   `db:"status,omitempty"`
	Role        *clients.Role    `db:"role,omitempty"`
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
		ID:          ch.ID,
		Name:        ch.Name,
		ParentGroup: toNullString(ch.ParentGroup),
		Domain:      ch.Domain,
		Tags:        tags,
		Metadata:    data,
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
	var updatedBy string
	if ch.UpdatedBy != nil {
		updatedBy = *ch.UpdatedBy
	}
	var updatedAt time.Time
	if ch.UpdatedAt.Valid {
		updatedAt = ch.UpdatedAt.Time
	}

	newCh := channels.Channel{
		ID:          ch.ID,
		Name:        ch.Name,
		Tags:        tags,
		Domain:      ch.Domain,
		ParentGroup: toString(ch.ParentGroup),
		Metadata:    metadata,
		CreatedAt:   ch.CreatedAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		Status:      ch.Status,
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

	if pm.ClientID != "" {
		query = append(query, "conn.client_id = :client_id")
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
	case "name", "created_at", "updated_at":
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
