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
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/things"
	"github.com/jackc/pgtype"
)

type clientRepo struct {
	Repository things.ClientRepository
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) things.Repository {
	return &clientRepo{
		Repository: things.ClientRepository{DB: db},
	}
}

func (repo *clientRepo) Save(ctx context.Context, th ...things.Client) ([]things.Client, error) {
	tx, err := repo.Repository.DB.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	var thingsList []things.Client

	for _, thi := range th {
		q := `INSERT INTO clients (id, name, tags, domain_id, identity, secret, metadata, created_at, updated_at, updated_by, status)
        VALUES (:id, :name, :tags, :domain_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
        RETURNING id, name, tags, identity, secret, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`

		dbthi, err := ToDBClient(thi)
		if err != nil {
			return []things.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbthi)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				return []things.Client{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
			return []things.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		defer row.Close()

		if row.Next() {
			dbthi = DBClient{}
			if err := row.StructScan(&dbthi); err != nil {
				return []things.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}

			thing, err := ToClient(dbthi)
			if err != nil {
				return []things.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
			}
			thingsList = append(thingsList, thing)
		}
	}
	if err = tx.Commit(); err != nil {
		return []things.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return thingsList, nil
}

func (repo *clientRepo) RetrieveBySecret(ctx context.Context, key string) (things.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = :secret AND status = %d`, things.EnabledStatus)

	dbt := DBClient{
		Secret: key,
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbt)
	if err != nil {
		return things.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbt = DBClient{}
	if rows.Next() {
		if err = rows.StructScan(&dbt); err != nil {
			return things.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		thing, err := ToClient(dbt)
		if err != nil {
			return things.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return thing, nil
	}

	return things.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) Update(ctx context.Context, thing things.Client) (things.Client, error) {
	var query []string
	var upq string
	if thing.Name != "" {
		query = append(query, "name = :name,")
	}
	if thing.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, secret,  metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`,
		upq)
	thing.Status = things.EnabledStatus
	return repo.update(ctx, thing, q)
}

func (repo *clientRepo) UpdateTags(ctx context.Context, thing things.Client) (things.Client, error) {
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`
	thing.Status = things.EnabledStatus
	return repo.update(ctx, thing, q)
}

func (repo *clientRepo) UpdateIdentity(ctx context.Context, thing things.Client) (things.Client, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`
	thing.Status = things.EnabledStatus
	return repo.update(ctx, thing, q)
}

func (repo *clientRepo) UpdateSecret(ctx context.Context, thing things.Client) (things.Client, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`
	thing.Status = things.EnabledStatus
	return repo.update(ctx, thing, q)
}

func (repo *clientRepo) ChangeStatus(ctx context.Context, thing things.Client) (things.Client, error) {
	q := `UPDATE clients SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at, updated_at, updated_by`

	return repo.update(ctx, thing, q)
}

func (repo *clientRepo) RetrieveByID(ctx context.Context, id string) (things.Client, error) {
	q := `SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients WHERE id = :id`

	dbt := DBClient{
		ID: id,
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbt)
	if err != nil {
		return things.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbt = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbt); err != nil {
			return things.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		return ToClient(dbt)
	}

	return things.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) RetrieveAll(ctx context.Context, pm things.Page) (things.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []things.Client
	for rows.Next() {
		dbt := DBClient{}
		if err := rows.StructScan(&dbt); err != nil {
			return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToClient(dbt)
		if err != nil {
			return things.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := things.ClientsPage{
		Clients: items,
		Page: things.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) SearchClients(ctx context.Context, pm things.Page) (things.ClientsPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []things.Client
	for rows.Next() {
		dbt := DBClient{}
		if err := rows.StructScan(&dbt); err != nil {
			return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToClient(dbt)
		if err != nil {
			return things.ClientsPage{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, tq)
	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := things.ClientsPage{
		Clients: items,
		Page: things.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) RetrieveAllByIDs(ctx context.Context, pm things.Page) (things.ClientsPage, error) {
	if (len(pm.IDs) == 0) && (pm.Domain == "") {
		return things.ClientsPage{
			Page: things.Page{Total: pm.Total, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}
	query, err := PageQuery(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBClientsPage(pm)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []things.Client
	for rows.Next() {
		dbt := DBClient{}
		if err := rows.StructScan(&dbt); err != nil {
			return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToClient(dbt)
		if err != nil {
			return things.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return things.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := things.ClientsPage{
		Clients: items,
		Page: things.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *clientRepo) update(ctx context.Context, thing things.Client, query string) (things.Client, error) {
	dbc, err := ToDBClient(thing)
	if err != nil {
		return things.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return things.Client{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbc = DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return things.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}

		return ToClient(dbc)
	}

	return things.Client{}, repoerr.ErrNotFound
}

func (repo *clientRepo) Delete(ctx context.Context, id string) error {
	q := "DELETE FROM clients AS c  WHERE c.id = $1 ;"

	result, err := repo.Repository.DB.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

type DBClient struct {
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	Identity  string           `db:"identity"`
	Domain    string           `db:"domain_id"`
	Secret    string           `db:"secret"`
	Metadata  []byte           `db:"metadata,omitempty"`
	CreatedAt time.Time        `db:"created_at,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    things.Status    `db:"status,omitempty"`
}

func ToDBClient(c things.Client) (DBClient, error) {
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
		ID:        c.ID,
		Name:      c.Name,
		Tags:      tags,
		Domain:    c.Domain,
		Identity:  c.Credentials.Identity,
		Secret:    c.Credentials.Secret,
		Metadata:  data,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
	}, nil
}

func ToClient(t DBClient) (things.Client, error) {
	var metadata things.Metadata
	if t.Metadata != nil {
		if err := json.Unmarshal([]byte(t.Metadata), &metadata); err != nil {
			return things.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
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

	thg := things.Client{
		ID:     t.ID,
		Name:   t.Name,
		Tags:   tags,
		Domain: t.Domain,
		Credentials: things.Credentials{
			Identity: t.Identity,
			Secret:   t.Secret,
		},
		Metadata:  metadata,
		CreatedAt: t.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    t.Status,
	}
	return thg, nil
}

func ToDBClientsPage(pm things.Page) (dbClientsPage, error) {
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
	Total    uint64        `db:"total"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
	Name     string        `db:"name"`
	Id       string        `db:"id"`
	Domain   string        `db:"domain_id"`
	Identity string        `db:"identity"`
	Metadata []byte        `db:"metadata"`
	Tag      string        `db:"tag"`
	Status   things.Status `db:"status"`
	GroupID  string        `db:"group_id"`
}

func PageQuery(pm things.Page) (string, error) {
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
	if pm.Status != things.AllStatus {
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

func applyOrdering(emq string, pm things.Page) string {
	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}
