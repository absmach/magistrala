// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/absmach/magistrala/pkg/clients"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/users"
	"github.com/jackc/pgtype"
)

// var _ mgclients.Repository = (*clientRepo)(nil)

// type clientRepo struct {
// 	pgclients.Repository
// }

type userRepo struct {
	DB postgres.Database
}

// Repository defines the required dependencies for Client repository.
//
//go:generate mockery --name Repository --output=../mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	mgclients.Repository

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client mgclients.Client) (mgclients.Client, error)

	RetrieveByID(ctx context.Context, id string) (mgclients.Client, error)

	UpdateRole(ctx context.Context, client mgclients.Client) (mgclients.Client, error)

	CheckSuperAdmin(ctx context.Context, adminID string) error
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
// func NewRepository(db postgres.Database) Repository {
// 	return &clientRepo{
// 		Repository: pgclients.Repository{DB: db},
// 	}
// }

func NewRepository(db postgres.Database) Repository {
	return &userRepo{
		DB: db,
	}
}

func (repo *userRepo) Save(ctx context.Context, c users.Users) (users.Users, error) {
	q := `INSERT INTO users (id, name, tags, identity, secret, metadata, created_at, status, role, first_name, last_name, user_name)
        VALUES (:id, :name, :tags, :identity, :secret, :metadata, :created_at, :status, :role, :first_name, :last_name, :user_name)
        RETURNING id, name, tags, identity, metadata, status, created_at, first_name, last_name, user_name`

	dbc, err := toDBUser(c)
	if err != nil {
		return users.Users{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.Users{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbc = DBUser{}
	if err := row.StructScan(&dbc); err != nil {
		return users.Users{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	user, err := ToUser(dbc)
	if err != nil {
		return users.Users{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return user, nil
}

func (repo *userRepo) CheckSuperAdmin(ctx context.Context, adminID string) error {
	q := "SELECT 1 FROM clients WHERE id = $1 AND role = $2"
	rows, err := repo.DB.QueryContext(ctx, q, adminID, mgclients.AdminRole)
	if err != nil {
		return postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Err(); err != nil {
			return postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		return nil
	}

	return repoerr.ErrNotFound
}

func (repo *userRepo) RetrieveByID(ctx context.Context, id string) (users.Users, error) {
	q := `SELECT id, name, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name
        FROM users WHERE id = :id`

	dbc := DBUser{
		ID: id,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.Users{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = DBUser{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return users.Users{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		user, err := ToUser(dbc)
		if err != nil {
			return users.Users{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return user, nil
	}

	return users.Users{}, repoerr.ErrNotFound
}

func (repo *userRepo) RetrieveAll(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	query, err := pgclients.PageQuery(pm)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata,  c.status, c.role,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := pgclients.ToDBClientsPage(pm)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []mgclients.Client
	for rows.Next() {
		dbc := pgclients.DBClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := pgclients.ToClient(dbc)
		if err != nil {
			return mgclients.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := mgclients.ClientsPage{
		Clients: items,
		Page: mgclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *userRepo) UpdateRole(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	query := `UPDATE clients SET role = :role, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, status, role, created_at, updated_at, updated_by`

	dbc, err := pgclients.ToDBClient(client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return mgclients.Client{}, postgres.HandleError(err, repoerr.ErrUpdateEntity)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbc = pgclients.DBClient{}
	if err := row.StructScan(&dbc); err != nil {
		return mgclients.Client{}, err
	}

	return pgclients.ToClient(dbc)
}

type DBUser struct {
	ID        string           `db:"id"`
	Name      string           `db:"name,omitempty"`
	Identity  string           `db:"identity"`
	Domain    string           `db:"domain_id"`
	Secret    string           `db:"secret"`
	Metadata  []byte           `db:"metadata,omitempty"`
	Tags      pgtype.TextArray `db:"tags,omitempty"`
	CreatedAt time.Time        `db:"created_at,omitempty"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
	Groups    []groups.Group   `db:"groups,omitempty"`
	Status    clients.Status   `db:"status,omitempty"`
	Role      *clients.Role    `db:"role,omitempty"`
	UserName  string           `db:"user_name, omitempty"`
	FirstName string           `db:"first_name, omitempty"`
	LastName  string           `db:"last_name, omitempty"`
}

func toDBUser(c users.Users) (DBUser, error) {
	data := []byte("{}")
	if len(c.Metadata) > 0 {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return DBUser{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(c.Tags); err != nil {
		return DBUser{}, err
	}
	var updatedBy *string
	if c.UpdatedBy != "" {
		updatedBy = &c.UpdatedBy
	}
	var updatedAt sql.NullTime
	if c.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: c.UpdatedAt, Valid: true}
	}

	return DBUser{
		ID:        c.ID,
		Name:      c.Name,
		Identity:  c.Credentials.Identity,
		Secret:    c.Credentials.Secret,
		Metadata:  data,
		CreatedAt: c.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    c.Status,
		Role:      &c.Role,
		LastName:  c.LastName,
		FirstName: c.FirstName,
		UserName:  c.UserName,
	}, nil
}

func ToUser(dbu DBUser) (users.Users, error) {
	var metadata users.Metadata
	if dbu.Metadata != nil {
		if err := json.Unmarshal(dbu.Metadata, &metadata); err != nil {
			return users.Users{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}
	var tags []string
	for _, e := range dbu.Tags.Elements {
		tags = append(tags, e.String)
	}
	var updatedBy string
	if dbu.UpdatedBy != nil {
		updatedBy = *dbu.UpdatedBy
	}
	var updatedAt time.Time
	if dbu.UpdatedAt.Valid {
		updatedAt = dbu.UpdatedAt.Time
	}

	user := users.Users{
		ID:        dbu.ID,
		Name:      dbu.Name,
		FirstName: dbu.FirstName,
		LastName:  dbu.LastName,
		UserName:  dbu.UserName,
		Credentials: users.Credentials{
			Identity: dbu.Identity,
			Secret:   dbu.Secret,
		},
		Metadata:  metadata,
		CreatedAt: dbu.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
		Status:    dbu.Status,
		Tags:      tags,
	}
	if dbu.Role != nil {
		user.Role = *dbu.Role
	}
	return user, nil
}

type DBUsersPage struct {
	Total    uint64         `db:"total"`
	Limit    uint64         `db:"limit"`
	Offset   uint64         `db:"offset"`
	Name     string         `db:"name"`
	Id       string         `db:"id"`
	Identity string         `db:"identity"`
	Metadata []byte         `db:"metadata"`
	Tag      string         `db:"tag"`
	GroupID  string         `db:"group_id"`
	Role     clients.Role   `db:"role"`
	Status   clients.Status `db:"status"`
}

func ToDBUsersPage(pm clients.Page) (DBUsersPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return DBUsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	return DBUsersPage{
		Name:     pm.Name,
		Identity: pm.Identity,
		Id:       pm.Id,
		Metadata: data,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		Status:   pm.Status,
		Tag:      pm.Tag,
		Role:     pm.Role,
	}, nil
}
