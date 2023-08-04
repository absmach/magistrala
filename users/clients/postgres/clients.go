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

	"github.com/jackc/pgtype" // required for SQL access
	"github.com/mainflux/mainflux/internal/postgres"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/groups"
)

var _ mfclients.Repository = (*clientRepo)(nil)

type clientRepo struct {
	db postgres.Database
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) mfclients.Repository {
	return &clientRepo{
		db: db,
	}
}

func (clientRepo) RetrieveBySecret(ctx context.Context, key string) (mfclients.Client, error) {
	return mfclients.Client{}, nil
}

func (repo clientRepo) Save(ctx context.Context, c ...mfclients.Client) ([]mfclients.Client, error) {
	q := `INSERT INTO clients (id, name, tags, owner_id, identity, secret, metadata, created_at, status, role)
        VALUES (:id, :name, :tags, :owner_id, :identity, :secret, :metadata, :created_at, :status, :role)
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at`
	dbc, err := toDBClient(c[0])
	if err != nil {
		return []mfclients.Client{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return []mfclients.Client{}, postgres.HandleError(err, errors.ErrCreateEntity)
	}

	defer row.Close()
	row.Next()
	dbc = dbClient{}
	if err := row.StructScan(&dbc); err != nil {
		return []mfclients.Client{}, err
	}

	client, err := toClient(dbc)
	if err != nil {
		return []mfclients.Client{}, err
	}

	return []mfclients.Client{client}, nil
}

func (repo clientRepo) RetrieveByID(ctx context.Context, id string) (mfclients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status 
        FROM clients WHERE id = :id`

	dbc := dbClient{
		ID: id,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		if err == sql.ErrNoRows {
			return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return mfclients.Client{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	defer row.Close()
	row.Next()
	dbc = dbClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
	}

	return toClient(dbc)
}

func (repo clientRepo) RetrieveByIdentity(ctx context.Context, identity string) (mfclients.Client, error) {
	q := `SELECT id, name, tags, COALESCE(owner_id, '') AS owner_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE identity = :identity AND status = :status`

	dbc := dbClient{
		Identity: identity,
		Status:   mfclients.EnabledStatus,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		if err == sql.ErrNoRows {
			return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
		}
		return mfclients.Client{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	defer row.Close()
	row.Next()
	dbc = dbClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, err)
	}

	return toClient(dbc)
}

func (repo clientRepo) RetrieveAll(ctx context.Context, pm mfclients.Page) (mfclients.ClientsPage, error) {
	query, err := pageQuery(pm)
	if err != nil {
		return mfclients.ClientsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.owner_id, '') AS owner_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := toDBClientsPage(pm)
	if err != nil {
		return mfclients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return mfclients.ClientsPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	var items []mfclients.Client
	for rows.Next() {
		dbc := dbClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return mfclients.ClientsPage{}, errors.Wrap(errors.ErrViewEntity, err)
		}

		c, err := toClient(dbc)
		if err != nil {
			return mfclients.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return mfclients.ClientsPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	page := mfclients.ClientsPage{
		Clients: items,
		Page: mfclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo clientRepo) Members(ctx context.Context, groupID string, pm mfclients.Page) (mfclients.MembersPage, error) {
	emq, err := pageQuery(pm)
	if err != nil {
		return mfclients.MembersPage{}, err
	}

	aq := ""
	// If not admin, the client needs to have a g_list action on the group or they are the owner.
	if pm.Subject != "" {
		aq = `AND (EXISTS (SELECT 1 FROM policies p WHERE p.subject = :subject AND :action=ANY(actions)) 
				OR EXISTS (SELECT 1 FROM groups g WHERE g.owner_id = :subject AND g.id = :group_id))
				AND c.id != :subject`
	}
	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.metadata, c.identity, c.status,
		c.created_at, c.updated_at FROM clients c
		INNER JOIN policies ON c.id=policies.subject %s AND policies.object = :group_id %s
	  	ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, emq, aq)
	dbPage, err := toDBClientsPage(pm)
	if err != nil {
		return mfclients.MembersPage{}, errors.Wrap(postgres.ErrFailedToRetrieveAll, err)
	}
	dbPage.GroupID = groupID
	rows, err := repo.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return mfclients.MembersPage{}, errors.Wrap(postgres.ErrFailedToRetrieveMembers, err)
	}
	defer rows.Close()

	var items []mfclients.Client
	for rows.Next() {
		dbc := dbClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return mfclients.MembersPage{}, errors.Wrap(postgres.ErrFailedToRetrieveMembers, err)
		}

		c, err := toClient(dbc)
		if err != nil {
			return mfclients.MembersPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c INNER JOIN policies ON c.id=policies.subject %s AND policies.object = :group_id`, emq)
	if pm.Subject != "" {
		cq = fmt.Sprintf("%s AND c.id != :subject", cq)
	}

	total, err := postgres.Total(ctx, repo.db, cq, dbPage)
	if err != nil {
		return mfclients.MembersPage{}, errors.Wrap(postgres.ErrFailedToRetrieveMembers, err)
	}

	page := mfclients.MembersPage{
		Members: items,
		Page: mfclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}
	return page, nil
}

func (repo clientRepo) Update(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
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
	client.Status = mfclients.EnabledStatus
	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`,
		upq)

	return repo.update(ctx, client, q)
}

func (repo clientRepo) UpdateTags(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
	client.Status = mfclients.EnabledStatus
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo clientRepo) UpdateIdentity(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo clientRepo) UpdateSecret(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE identity = :identity AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo clientRepo) UpdateOwner(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
	q := `UPDATE clients SET owner_id = :owner_id, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo clientRepo) ChangeStatus(ctx context.Context, client mfclients.Client) (mfclients.Client, error) {
	q := `UPDATE clients SET status = :status WHERE id = :id
        RETURNING id, name, tags, identity, metadata, COALESCE(owner_id, '') AS owner_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, client, q)
}

func (repo clientRepo) update(ctx context.Context, client mfclients.Client, query string) (mfclients.Client, error) {
	dbc, err := toDBClient(client)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}
	row, err := repo.db.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return mfclients.Client{}, postgres.HandleError(err, errors.ErrUpdateEntity)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return mfclients.Client{}, errors.Wrap(errors.ErrNotFound, row.Err())
	}
	dbc = dbClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mfclients.Client{}, err
	}

	return toClient(dbc)
}

type dbClient struct {
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Identity  string           `db:"identity"`
	Owner     *string          `db:"owner_id,omitempty"` // nullable
	Secret    string           `db:"secret"`
	Metadata  []byte           `db:"metadata,omitempty"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    mfclients.Status `db:"status"`
	Role      mfclients.Role   `db:"role"`
}

func toDBClient(c mfclients.Client) (dbClient, error) {
	data := []byte("{}")
	if len(c.Metadata) > 0 {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return dbClient{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(c.Tags); err != nil {
		return dbClient{}, err
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
	if !c.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{Time: c.UpdatedAt, Valid: true}
	}

	return dbClient{
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
		Role:      c.Role,
	}, nil
}

func toClient(c dbClient) (mfclients.Client, error) {
	var metadata mfclients.Metadata
	if c.Metadata != nil {
		if err := json.Unmarshal([]byte(c.Metadata), &metadata); err != nil {
			return mfclients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
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

	return mfclients.Client{
		ID:    c.ID,
		Name:  c.Name,
		Tags:  tags,
		Owner: owner,
		Credentials: mfclients.Credentials{
			Identity: c.Identity,
			Secret:   c.Secret,
		},
		Metadata:  metadata,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
	}, nil
}

func pageQuery(pm mfclients.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrViewEntity, err)
	}
	var query []string
	var emq string
	if mq != "" {
		query = append(query, mq)
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
	if pm.Status != mfclients.AllStatus {
		query = append(query, "c.status = :status")
	}
	// For listing clients that the specified client owns but not sharedby
	if pm.Owner != "" && pm.SharedBy == "" {
		query = append(query, "c.owner_id = :owner_id")
	}

	// For listing clients that the specified client owns and that are shared with the specified client
	if pm.Owner != "" && pm.SharedBy != "" {
		query = append(query, "(c.owner_id = :owner_id OR (policies.object IN (SELECT object FROM policies WHERE subject = :shared_by AND :action=ANY(actions)))) AND c.id != :shared_by")
	}
	// For listing clients that the specified client is shared with
	if pm.SharedBy != "" && pm.Owner == "" {
		query = append(query, "c.owner_id != :shared_by AND (policies.object IN (SELECT object FROM policies WHERE subject = :shared_by AND :action=ANY(actions))) AND c.id != :shared_by")
	}
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
		if strings.Contains(emq, "policies") {
			emq = fmt.Sprintf("LEFT JOIN policies ON policies.subject = c.id %s", emq)
		}
	}
	return emq, nil

}

func toDBClientsPage(pm mfclients.Page) (dbClientsPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return dbClientsPage{}, errors.Wrap(errors.ErrViewEntity, err)
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
		Subject:  pm.Subject,
		Action:   pm.Action,
		SharedBy: pm.SharedBy,
	}, nil
}

type dbClientsPage struct {
	Total    uint64           `db:"total"`
	Limit    uint64           `db:"limit"`
	Offset   uint64           `db:"offset"`
	Name     string           `db:"name"`
	Owner    string           `db:"owner_id"`
	Identity string           `db:"identity"`
	Metadata []byte           `db:"metadata"`
	Tag      string           `db:"tag"`
	Status   mfclients.Status `db:"status"`
	GroupID  string           `db:"group_id"`
	SharedBy string           `db:"shared_by"`
	Subject  string           `db:"subject"`
	Action   string           `db:"action"`
}
