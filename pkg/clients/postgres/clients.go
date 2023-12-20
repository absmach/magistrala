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

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/jackc/pgtype"
)

type ClientRepository struct {
	DB postgres.Database
}

func (repo ClientRepository) Update(ctx context.Context, client clients.Client) (clients.Client, error) {
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
	client.Status = clients.EnabledStatus
	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, secret,  metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`,
		upq)

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) UpdateTags(ctx context.Context, client clients.Client) (clients.Client, error) {
	client.Status = clients.EnabledStatus
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) UpdateIdentity(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) UpdateSecret(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) UpdateOwner(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET owner_id = :owner_id, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) UpdateRole(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET role = :role, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) ChangeStatus(ctx context.Context, client clients.Client) (clients.Client, error) {
	q := `UPDATE clients SET status = :status WHERE id = :id
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo ClientRepository) RetrieveByID(ctx context.Context, id string) (clients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE id = :id`

	dbc := DBClient{
		ID: id,
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		if err == sql.ErrNoRows {
			return clients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
		return clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	defer row.Close()
	row.Next()
	dbc = DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return clients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}

	return ToClient(dbc)
}

func (repo ClientRepository) RetrieveByIdentity(ctx context.Context, identity string) (clients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE identity = :identity AND status = :status`

	dbc := DBClient{
		Identity: identity,
		Status:   clients.EnabledStatus,
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		if err == sql.ErrNoRows {
			return clients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
		return clients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	defer row.Close()
	row.Next()
	dbc = DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return clients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}

	return ToClient(dbc)
}

func (repo ClientRepository) RetrieveAll(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.owner_id, '') AS owner_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
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

func (repo ClientRepository) RetrieveAllBasicInfo(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	sq, tq := constructSearchQuery(pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, sq)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
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

func (repo ClientRepository) RetrieveAllByIDs(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	if (len(pm.IDs) <= 0) && (pm.Owner == "") {
		return clients.ClientsPage{
			Page: clients.Page{Total: pm.Total, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}
	query, err := PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.owner_id, '') AS owner_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
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

// generic update function.
func (repo ClientRepository) update(ctx context.Context, client clients.Client, query string) (clients.Client, error) {
	dbc, err := ToDBClient(client)
	if err != nil {
		return clients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return clients.Client{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return clients.Client{}, errors.Wrap(errors.ErrNotFound, row.Err())
	}
	dbc = DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return clients.Client{}, err
	}

	return ToClient(dbc)
}

type DBClient struct {
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Identity  string           `db:"identity"`
	Owner     *string          `db:"owner_id,omitempty"` // nullable
	Secret    string           `db:"secret"`
	Metadata  []byte           `db:"metadata,omitempty"`
	CreatedAt time.Time        `db:"created_at,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    clients.Status   `db:"status,omitempty"`
	Role      *clients.Role    `db:"role,omitempty"`
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
	var owner *string
	if c.Owner != "" {
		owner = &c.Owner
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
		ID:        c.ID,
		Name:      c.Name,
		Tags:      tags,
		Owner:     owner,
		Identity:  c.Credentials.Identity,
		Secret:    c.Credentials.Secret,
		Metadata:  data,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
		Role:      &c.Role,
	}, nil
}

func ToClient(c DBClient) (clients.Client, error) {
	var metadata clients.Metadata
	if c.Metadata != nil {
		if err := json.Unmarshal([]byte(c.Metadata), &metadata); err != nil {
			return clients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range c.Tags.Elements {
		tags = append(tags, e.String)
	}
	var owner string
	if c.Owner != nil {
		owner = *c.Owner
	}
	var updatedBy string
	if c.UpdatedBy != nil {
		updatedBy = *c.UpdatedBy
	}
	var updatedAt time.Time
	if c.UpdatedAt.Valid {
		updatedAt = c.UpdatedAt.Time
	}

	cli := clients.Client{
		ID:    c.ID,
		Name:  c.Name,
		Tags:  tags,
		Owner: owner,
		Credentials: clients.Credentials{
			Identity: c.Identity,
			Secret:   c.Secret,
		},
		Metadata:  metadata,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
	}
	if c.Role != nil {
		cli.Role = *c.Role
	}
	return cli, nil
}

func ToDBClientsPage(pm clients.Page) (dbClientsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	var role clients.Role
	if pm.Role != nil {
		role = *pm.Role
	}
	return dbClientsPage{
		Name:     pm.Name,
		Identity: pm.Identity,
		Metadata: data,
		Owner:    pm.Owner,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		Status:   pm.Status,
		Tag:      pm.Tag,
		Role:     uint8(role),
	}, nil
}

type dbClientsPage struct {
	Total    uint64         `db:"total"`
	Limit    uint64         `db:"limit"`
	Offset   uint64         `db:"offset"`
	Name     string         `db:"name"`
	Owner    string         `db:"owner_id"`
	Identity string         `db:"identity"`
	Metadata []byte         `db:"metadata"`
	Tag      string         `db:"tag"`
	Status   clients.Status `db:"status"`
	GroupID  string         `db:"group_id"`
	Role     uint8          `db:"role"`
}

func PageQuery(pm clients.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrViewEntity, err)
	}
	var query []string
	var emq string
	if mq != "" {
		query = append(query, mq)
	}
	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("id IN ('%s')", strings.Join(pm.IDs, "','")))
	}
	if pm.Identity != "" {
		query = append(query, "c.identity = :identity")
	}
	if pm.Name != "" {
		query = append(query, "c.name = :name")
	}
	if pm.Tag != "" {
		query = append(query, ":tag = ANY(c.tags)")
	}
	if pm.Status != clients.AllStatus {
		query = append(query, "c.status = :status")
	}
	// For listing clients that the specified client owns but not sharedby
	if pm.Owner != "" {
		query = append(query, "c.owner_id = :owner_id")
	}

	if pm.Role != nil {
		query = append(query, "c.role = :role")
	}
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}
	return emq, nil
}

func constructSearchQuery(pm clients.Page) (string, string) {
	var query []string
	var emq string
	var tq string

	if pm.Name != "" {
		query = append(query, "name ~ :name")
	}
	if pm.Identity != "" {
		query = append(query, "identity ~ :identity")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	tq = emq

	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}

	return emq, tq
}
