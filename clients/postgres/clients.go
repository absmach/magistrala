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

	"github.com/absmach/magistrala/clients"
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
	entityTableName      = "clients"
	entityIDColumnName   = "id"
	rolesTableNamePrefix = "clients"
)

var _ clients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	DB postgres.Database
	rolesPostgres.Repository
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) clients.Repository {
	repo := rolesPostgres.NewRepository(db, rolesTableNamePrefix, entityTableName, entityIDColumnName)

	return &clientRepo{
		DB:         db,
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
	q := `INSERT INTO clients (id, name, tags, domain_id, parent_group_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
	VALUES (:id, :name, :tags, :domain_id, :parent_group_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
	RETURNING id, name, tags, identity, secret, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS  parent_group_id, status, created_at, updated_at, updated_by`

	row, err := repo.DB.NamedQueryContext(ctx, q, dbClients)
	if err != nil {
		return []clients.Client{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	var reClients []clients.Client
	for row.Next() {
		dbcli := DBClient{}
		if err := row.StructScan(&dbcli); err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		client, err := ToClient(dbcli)
		if err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}
		reClients = append(reClients, client)
	}
	return reClients, nil
}

func (repo *clientRepo) RetrieveBySecret(ctx context.Context, key string) (clients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id,  COALESCE(parent_group_id, '') AS parent_group_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = :secret AND status = %d`, clients.EnabledStatus)

	dbc := DBClient{
		Secret: key,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return clients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = DBClient{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return clients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
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
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, secret,  metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`,
		upq)
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateTags(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateIdentity(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, COALESCE(parent_group_id, '') AS parent_group_id, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) UpdateSecret(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`
	client.Status = clients.EnabledStatus
	return repo.update(ctx, client, q)
}

func (repo *clientRepo) ChangeStatus(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo *clientRepo) RetrieveByID(ctx context.Context, id string) (clients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE id = :id`

	dbc := DBClient{
		ID: id,
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbc = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		return ToClient(dbc)
	}

	return clients.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) RetrieveAll(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) SearchClients(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, query)

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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, tq)
	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
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

func (repo *clientRepo) RetrieveAllByIDs(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	if (len(pm.IDs) == 0) && (pm.Domain == "") {
		return clients.ClientsPage{
			Page: clients.Page{Total: pm.Total, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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
		return clients.Client{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbc = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return clients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}

		return ToClient(dbc)
	}

	return clients.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) Delete(ctx context.Context, clientIDs ...string) error {
	q := "DELETE FROM clients AS c  WHERE c.id = ANY(:client_ids) ;"

	params := map[string]interface{}{
		"client_ids": clientIDs,
	}
	result, err := repo.DB.NamedExecContext(ctx, q, params)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

type DBClient struct {
	ID          string           `db:"id"`
	Name        string           `db:"name,omitempty"`
	Tags        pgtype.TextArray `db:"tags,omitempty"`
	Identity    string           `db:"identity"`
	Domain      string           `db:"domain_id"`
	ParentGroup sql.NullString   `db:"parent_group_id,omitempty"`
	Secret      string           `db:"secret"`
	Metadata    []byte           `db:"metadata,omitempty"`
	CreatedAt   time.Time        `db:"created_at,omitempty"`
	UpdatedAt   sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy   *string          `db:"updated_by,omitempty"`
	Status      clients.Status   `db:"status,omitempty"`
}

func ToDBClient(c clients.Client) (DBClient, error) {
	data := []byte("{}")
	if len(c.Metadata) > 0 {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return DBClient{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
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
		ID:          c.ID,
		Name:        c.Name,
		Tags:        tags,
		Domain:      c.Domain,
		ParentGroup: toNullString(c.ParentGroup),
		Identity:    c.Credentials.Identity,
		Secret:      c.Credentials.Secret,
		Metadata:    data,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   updatedBy,
		Status:      c.Status,
	}, nil
}

func ToClient(t DBClient) (clients.Client, error) {
	var metadata clients.Metadata
	if t.Metadata != nil {
		if err := json.Unmarshal([]byte(t.Metadata), &metadata); err != nil {
			return clients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
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
		updatedAt = t.UpdatedAt.Time
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
		Metadata:  metadata,
		CreatedAt: t.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    t.Status,
	}
	return cli, nil
}

func ToDBClientsPage(pm clients.Page) (dbClientsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return dbClientsPage{
		Name:     pm.Name,
		Identity: pm.Identity,
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

type dbClientsPage struct {
	Total    uint64         `db:"total"`
	Limit    uint64         `db:"limit"`
	Offset   uint64         `db:"offset"`
	Name     string         `db:"name"`
	Id       string         `db:"id"`
	Domain   string         `db:"domain_id"`
	Identity string         `db:"identity"`
	Metadata []byte         `db:"metadata"`
	Tag      string         `db:"tag"`
	Status   clients.Status `db:"status"`
	GroupID  string         `db:"group_id"`
}

func PageQuery(pm clients.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var query []string
	if pm.Name != "" {
		query = append(query, "name ILIKE '%' || :name || '%'")
	}
	if pm.Identity != "" {
		query = append(query, "identity ILIKE '%' || :identity || '%'")
	}
	if pm.Id != "" {
		query = append(query, "id ILIKE '%' || :id || '%'")
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

func applyOrdering(emq string, pm clients.Page) string {
	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
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
