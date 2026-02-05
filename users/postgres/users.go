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
	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/users"
	"github.com/jackc/pgtype"
)

type userRepo struct {
	Repository users.UserRepository
	eh         errors.Handler
}

func NewRepository(db postgres.Database) users.Repository {
	errHandlerOptions := []errors.HandlerOption{
		postgres.WithDuplicateErrors(NewDuplicateErrors()),
	}
	return &userRepo{
		Repository: users.UserRepository{DB: db},
		eh:         postgres.NewErrorHandler(errHandlerOptions...),
	}
}

func (repo *userRepo) Save(ctx context.Context, c users.User) (users.User, error) {
	q := `INSERT INTO users (id, tags, email, secret, metadata, private_metadata, created_at, status, role, first_name, last_name, username, profile_picture, auth_provider)
        VALUES (:id, :tags, :email, :secret, :metadata, :private_metadata, :created_at, :status, :role, :first_name, :last_name, :username, :profile_picture, :auth_provider)
        RETURNING id, tags, email, metadata, private_metadata, created_at, status, role, first_name, last_name, username, profile_picture, verified_at, auth_provider`

	dbu, err := toDBUser(c)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrMarshalBDEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	row.Next()

	dbu = DBUser{}
	if err := row.StructScan(&dbu); err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrFailedOpDB, err)
	}

	user, err := ToUser(dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrUnmarshalBDEntity, err)
	}

	return user, nil
}

func (repo *userRepo) CheckSuperAdmin(ctx context.Context, adminID string) error {
	q := "SELECT 1 FROM users WHERE id = $1 AND role = $2"
	rows, err := repo.Repository.DB.QueryContext(ctx, q, adminID, users.AdminRole)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Err(); err != nil {
			return repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}
		return nil
	}

	return repoerr.ErrNotFound
}

func (repo *userRepo) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	q := `SELECT id, tags, email, secret, metadata, private_metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, username, profile_picture, verified_at, auth_provider
        FROM users WHERE id = :id`

	dbu := DBUser{
		ID: id,
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbu = DBUser{}
	if !rows.Next() {
		return users.User{}, repoerr.ErrNotFound
	}

	if err = rows.StructScan(&dbu); err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	user, err := ToUser(dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrUnmarshalBDEntity, err)
	}

	return user, nil
}

func (repo *userRepo) RetrieveAll(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrParseQueryParams, err)
	}

	squery := applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT u.id, u.tags, u.email, u.metadata, u.status, u.role, u.first_name, u.last_name, u.username,
    u.created_at, u.updated_at, u.profile_picture, COALESCE(u.updated_by, '') AS updated_by, u.verified_at
    FROM users u %s LIMIT :limit OFFSET :offset;`, squery)

	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrMarshalBDEntity, err)
	}

	var items []users.User
	if !pm.OnlyTotal {
		rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
		if err != nil {
			return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrRetrieveAllUsers, err)
		}
		defer rows.Close()

		for rows.Next() {
			dbu := DBUser{}
			if err := rows.StructScan(&dbu); err != nil {
				return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
			}

			c, err := ToUser(dbu)
			if err != nil {
				return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrUnmarshalBDEntity, err)
			}

			items = append(items, c)
		}
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := users.UsersPage{
		Page: users.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
		Users: items,
	}

	return page, nil
}

func (repo *userRepo) UpdateUsername(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET username = :username, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
		RETURNING id, tags, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, email, role, verified_at`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) Update(ctx context.Context, id string, ur users.UserReq) (users.User, error) {
	var query []string
	var upq string
	u := users.User{ID: id}
	if ur.FirstName != nil && *ur.FirstName != "" {
		query = append(query, "first_name = :first_name")
		u.FirstName = *ur.FirstName
	}
	if ur.LastName != nil && *ur.LastName != "" {
		query = append(query, "last_name = :last_name")
		u.LastName = *ur.LastName
	}
	if ur.Metadata != nil {
		query = append(query, "metadata = :metadata")
		u.Metadata = *ur.Metadata
	}
	if ur.PrivateMetadata != nil {
		query = append(query, "private_metadata = :private_metadata")
		u.PrivateMetadata = *ur.PrivateMetadata
	}
	if ur.Tags != nil {
		query = append(query, "tags = :tags")
		u.Tags = *ur.Tags
	}
	if ur.ProfilePicture != nil {
		query = append(query, "profile_picture = :profile_picture")
		u.ProfilePicture = *ur.ProfilePicture
	}
	u.UpdatedAt = time.Now().UTC()
	if ur.UpdatedAt != nil {
		query = append(query, "updated_at = :updated_at")
		u.UpdatedAt = *ur.UpdatedAt
	}
	if ur.UpdatedBy != nil {
		query = append(query, "updated_by = :updated_by")
		u.UpdatedBy = *ur.UpdatedBy
	}

	if len(query) > 0 {
		upq = strings.Join(query, ", ")
	}

	q := fmt.Sprintf(`UPDATE users SET %s
        WHERE id = :id AND status = :status
        RETURNING id, tags, metadata, private_metadata, status, created_at, updated_at, updated_by, last_name, first_name, username, profile_picture, email, role, verified_at`, upq)

	u.Status = users.EnabledStatus
	return repo.update(ctx, u, q)
}

func (repo *userRepo) update(ctx context.Context, user users.User, query string) (users.User, error) {
	dbu, err := toDBUser(user)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrMarshalBDEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, query, dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbu = DBUser{}
	if !row.Next() {
		return users.User{}, repoerr.ErrNotFound
	}

	if err := row.StructScan(&dbu); err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrUnmarshalBDEntity, err)
	}

	return ToUser(dbu)
}

func (repo *userRepo) UpdateEmail(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET email = :email, verified_at = NULL, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, email, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, role, verified_at`
	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) UpdateRole(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET role = :role, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, email, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, role, verified_at`
	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) UpdateSecret(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, email, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, role, verified_at`
	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) ChangeStatus(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, tags, email, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, role, verified_at`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) UpdateVerifiedAt(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET verified_at = :verified_at
			WHERE id = :id and email = :email
        RETURNING id, tags, email, metadata, private_metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, role, verified_at`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) Delete(ctx context.Context, id string) error {
	q := "DELETE FROM users AS u  WHERE u.id = $1 ;"

	result, err := repo.Repository.DB.ExecContext(ctx, q, id)
	if err != nil {
		return repo.eh.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *userRepo) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrParseQueryParams, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT u.id, u.username, u.metadata, u.first_name, u.last_name, u.created_at, u.updated_at FROM users u %s LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrMarshalBDEntity, err)
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		dbu := DBUser{}
		if err := rows.StructScan(&dbu); err != nil {
			return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		c, err := ToUser(dbu)
		if err != nil {
			return users.UsersPage{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s;`, tq)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := users.UsersPage{
		Users: items,
		Page: users.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *userRepo) RetrieveAllByIDs(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	if (len(pm.IDs) == 0) && (pm.Domain == "") {
		return users.UsersPage{
			Page: users.Page{Total: pm.Total, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}
	query, err := PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrParseQueryParams, err)
	}
	squery := applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT u.id, u.username, u.tags, u.email, u.metadata, u.status, u.role, u.first_name, u.last_name,
                    u.created_at, u.updated_at, COALESCE(u.updated_by, '') AS updated_by FROM users u %s LIMIT :limit OFFSET :offset;`, squery)
	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrMarshalBDEntity, err)
	}
	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		dbu := DBUser{}
		if err := rows.StructScan(&dbu); err != nil {
			return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		c, err := ToUser(dbu)
		if err != nil {
			return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrUnmarshalBDEntity, err)
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}

	page := users.UsersPage{
		Users: items,
		Page: users.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *userRepo) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	q := `SELECT id, tags, email, secret, metadata, private_metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, username, verified_at, auth_provider
        FROM users WHERE email = :email AND status = :status`

	dbu := DBUser{
		Email:  email,
		Status: users.EnabledStatus,
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbu = DBUser{}
	if row.Next() {
		if err := row.StructScan(&dbu); err != nil {
			return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		return ToUser(dbu)
	}

	return users.User{}, repoerr.ErrNotFound
}

func (repo *userRepo) RetrieveByUsername(ctx context.Context, username string) (users.User, error) {
	q := `SELECT id, tags, email, secret, metadata, private_metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, username, verified_at, auth_provider
		FROM users WHERE username = :username AND status = :status`

	dbu := DBUser{
		Username: sql.NullString{String: username, Valid: username != ""},
		Status:   users.EnabledStatus,
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbu = DBUser{}
	if row.Next() {
		if err := row.StructScan(&dbu); err != nil {
			return users.User{}, repo.eh.HandleError(repoerr.ErrViewEntity, err)
		}

		return ToUser(dbu)
	}

	return users.User{}, repoerr.ErrNotFound
}

type DBUser struct {
	ID              string           `db:"id"`
	Domain          string           `db:"domain_id"`
	Secret          string           `db:"secret"`
	Metadata        []byte           `db:"metadata,omitempty"`
	PrivateMetadata []byte           `db:"private_metadata,omitempty"`
	Tags            pgtype.TextArray `db:"tags,omitempty"` // Tags
	CreatedAt       time.Time        `db:"created_at,omitempty"`
	UpdatedAt       sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy       *string          `db:"updated_by,omitempty"`
	Groups          []groups.Group   `db:"groups,omitempty"`
	Status          users.Status     `db:"status,omitempty"`
	Role            *users.Role      `db:"role,omitempty"`
	Username        sql.NullString   `db:"username, omitempty"`
	FirstName       sql.NullString   `db:"first_name, omitempty"`
	LastName        sql.NullString   `db:"last_name, omitempty"`
	ProfilePicture  sql.NullString   `db:"profile_picture, omitempty"`
	Email           string           `db:"email,omitempty"`
	VerifiedAt      sql.NullTime     `db:"verified_at,omitempty"`
	AuthProvider    sql.NullString   `db:"auth_provider,omitempty"`
}

func toDBUser(u users.User) (DBUser, error) {
	metadata := []byte("{}")
	if len(u.Metadata) > 0 {
		b, err := json.Marshal(u.Metadata)
		if err != nil {
			return DBUser{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		metadata = b
	}
	privateMetadata := []byte("{}")
	if len(u.PrivateMetadata) > 0 {
		b, err := json.Marshal(u.PrivateMetadata)
		if err != nil {
			return DBUser{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		privateMetadata = b
	}
	var tags pgtype.TextArray
	if err := tags.Set(u.Tags); err != nil {
		return DBUser{}, err
	}
	var updatedBy *string
	if u.UpdatedBy != "" {
		updatedBy = &u.UpdatedBy
	}
	var updatedAt sql.NullTime
	if u.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: u.UpdatedAt, Valid: true}
	}
	var verifiedAt sql.NullTime
	if u.VerifiedAt != (time.Time{}) {
		verifiedAt = sql.NullTime{Time: u.VerifiedAt, Valid: true}
	}

	var authProvider sql.NullString
	if u.AuthProvider != "" {
		authProvider = sql.NullString{String: u.AuthProvider, Valid: true}
	}

	return DBUser{
		ID:              u.ID,
		Tags:            tags,
		Secret:          u.Credentials.Secret,
		Metadata:        metadata,
		PrivateMetadata: privateMetadata,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       updatedAt,
		UpdatedBy:       updatedBy,
		Status:          u.Status,
		Role:            &u.Role,
		LastName:        stringToNullString(u.LastName),
		FirstName:       stringToNullString(u.FirstName),
		Username:        stringToNullString(u.Credentials.Username),
		ProfilePicture:  stringToNullString(u.ProfilePicture),
		Email:           u.Email,
		VerifiedAt:      verifiedAt,
		AuthProvider:    authProvider,
	}, nil
}

func ToUser(dbu DBUser) (users.User, error) {
	var metadata, privateMetadata users.Metadata
	if dbu.Metadata != nil {
		if err := json.Unmarshal([]byte(dbu.Metadata), &metadata); err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}
	if dbu.PrivateMetadata != nil {
		if err := json.Unmarshal([]byte(dbu.PrivateMetadata), &privateMetadata); err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
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
		updatedAt = dbu.UpdatedAt.Time.UTC()
	}
	var verifiedAt time.Time
	if dbu.VerifiedAt.Valid {
		verifiedAt = dbu.VerifiedAt.Time.UTC()
	}

	var authProvider string
	if dbu.AuthProvider.Valid {
		authProvider = dbu.AuthProvider.String
	}

	user := users.User{
		ID:        dbu.ID,
		FirstName: nullStringString(dbu.FirstName),
		LastName:  nullStringString(dbu.LastName),
		Credentials: users.Credentials{
			Username: nullStringString(dbu.Username),
			Secret:   dbu.Secret,
		},
		Email:           dbu.Email,
		Metadata:        metadata,
		PrivateMetadata: privateMetadata,
		CreatedAt:       dbu.CreatedAt.UTC(),
		UpdatedAt:       updatedAt,
		UpdatedBy:       updatedBy,
		Status:          dbu.Status,
		Tags:            tags,
		ProfilePicture:  nullStringString(dbu.ProfilePicture),
		VerifiedAt:      verifiedAt,
		AuthProvider:    authProvider,
	}
	if dbu.Role != nil {
		user.Role = *dbu.Role
	}
	return user, nil
}

type DBUsersPage struct {
	Total       uint64           `db:"total"`
	Limit       uint64           `db:"limit"`
	Offset      uint64           `db:"offset"`
	FirstName   string           `db:"first_name"`
	LastName    string           `db:"last_name"`
	Username    string           `db:"username"`
	Id          string           `db:"id"`
	Email       string           `db:"email"`
	Metadata    []byte           `db:"metadata"`
	Tags        pgtype.TextArray `db:"tags"`
	GroupID     string           `db:"group_id"`
	Role        users.Role       `db:"role"`
	Status      users.Status     `db:"status"`
	CreatedFrom time.Time        `db:"created_from"`
	CreatedTo   time.Time        `db:"created_to"`
}

func ToDBUsersPage(pm users.Page) (DBUsersPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return DBUsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	var tags pgtype.TextArray
	if err := tags.Set(pm.Tags.Elements); err != nil {
		return DBUsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return DBUsersPage{
		FirstName:   pm.FirstName,
		LastName:    pm.LastName,
		Username:    pm.Username,
		Email:       pm.Email,
		Id:          pm.Id,
		Metadata:    data,
		Total:       pm.Total,
		Offset:      pm.Offset,
		Limit:       pm.Limit,
		Status:      pm.Status,
		Tags:        tags,
		Role:        pm.Role,
		CreatedFrom: pm.CreatedFrom,
		CreatedTo:   pm.CreatedTo,
	}, nil
}

func PageQuery(pm users.Page) (string, error) {
	var query []string
	if pm.FirstName != "" {
		query = append(query, "first_name ILIKE '%' || :first_name || '%'")
	}
	if pm.LastName != "" {
		query = append(query, "last_name ILIKE '%' || :last_name || '%'")
	}
	if pm.Username != "" {
		query = append(query, "username ILIKE '%' || :username || '%'")
	}
	if pm.Email != "" {
		query = append(query, "email ILIKE '%' || :email || '%'")
	}
	if pm.Id != "" {
		query = append(query, "id ILIKE '%' || :id || '%'")
	}
	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case users.AndOp:
			query = append(query, "tags @> :tags")
		default: // OR
			query = append(query, "tags && :tags")
		}
	}
	if pm.Role != users.AllRole {
		query = append(query, "u.role = :role")
	}
	if len(pm.Metadata) > 0 {
		query = append(query, "metadata @> :metadata")
	}
	if len(pm.IDs) != 0 {
		query = append(query, fmt.Sprintf("id IN ('%s')", strings.Join(pm.IDs, "','")))
	}
	if pm.Status != users.AllStatus {
		query = append(query, "u.status = :status")
	}
	if !pm.CreatedFrom.IsZero() {
		query = append(query, "created_at >= :created_from")
	}
	if !pm.CreatedTo.IsZero() {
		query = append(query, "created_at <= :created_to")
	}

	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq, nil
}

func applyOrdering(emq string, pm users.Page) string {
	col := "COALESCE(u.updated_at, u.created_at)"

	switch pm.Order {
	case "username":
		col = "u.username"
	case "first_name":
		col = "u.first_name"
	case "last_name":
		col = "u.last_name"
	case "email":
		col = "u.email"
	case "created_at":
		col = "u.created_at"
	case "updated_at", "":
		col = "COALESCE(u.updated_at, u.created_at)"
	}

	dir := pm.Dir
	if dir != api.AscDir && dir != api.DescDir {
		dir = api.DescDir
	}

	return fmt.Sprintf("%s ORDER BY %s %s, u.id %s", emq, col, dir, dir)
}

func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func nullStringString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
