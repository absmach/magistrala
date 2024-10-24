// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/absmach/magistrala/internal/api"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/users"
	"github.com/jackc/pgtype"
)

type userRepo struct {
	Repository users.UserRepository
}

func NewRepository(db postgres.Database) users.Repository {
	return &userRepo{
		Repository: users.UserRepository{DB: db},
	}
}

func (repo *userRepo) Save(ctx context.Context, c users.User) (users.User, error) {
	q := `INSERT INTO users (id, tags, email, secret, metadata, created_at, status, role, first_name, last_name, username, profile_picture)
        VALUES (:id, :tags, :email, :secret, :metadata, :created_at, :status, :role, :first_name, :last_name, :username, :profile_picture)
        RETURNING id, tags, email, metadata, created_at, status, first_name, last_name, username, profile_picture`

	dbc, err := toDBUser(c)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	row.Next()

	dbc = DBUser{}
	if err := row.StructScan(&dbc); err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	user, err := ToUser(dbc)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
	}

	return user, nil
}

func (repo *userRepo) CheckSuperAdmin(ctx context.Context, adminID string) error {
	q := "SELECT 1 FROM users WHERE id = $1 AND role = $2"
	rows, err := repo.Repository.DB.QueryContext(ctx, q, adminID, mgclients.AdminRole)
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

func (repo *userRepo) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	q := `SELECT id, tags, email, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, username, profile_picture
        FROM users WHERE id = :id`

	dbc := DBUser{
		ID: id,
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = DBUser{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return users.User{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		user, err := ToUser(dbc)
		if err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return user, nil
	}

	return users.User{}, repoerr.ErrNotFound
}

func (repo *userRepo) RetrieveAll(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT u.id, u.tags, u.email, u.metadata, u.status, u.role, u.first_name, u.last_name, u.username,
    u.created_at, u.updated_at, u.profile_picture, COALESCE(u.updated_by, '') AS updated_by 
    FROM users u %s ORDER BY u.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		dbu := DBUser{}
		if err := rows.StructScan(&dbu); err != nil {
			return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToUser(dbu)
		if err != nil {
			return users.UsersPage{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
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
	if user.FirstName != "" && user.LastName != "" {
		return users.User{}, repoerr.ErrMissingNames
	}

	q := `UPDATE users SET first_name = :first_name, last_name = :last_name, username = :username, email = :email, updated_at = :updated_at, updated_by = :updated_by,
        WHERE id = :id AND status = :status
		RETURNING id, tags, metadata, status, created_at, updated_at, updated_by, first_name, last_name, username, email`

	dbc, err := toDBUser(user)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(err, repoerr.ErrUpdateEntity)
	}

	defer row.Close()

	dbc = DBUser{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Username:  user.Credentials.Username,
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	if ok := row.Next(); !ok {
		return users.User{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}

	if err := row.StructScan(&dbc); err != nil {
		return users.User{}, err
	}

	return ToUser(dbc)
}

func (repo *userRepo) Update(ctx context.Context, user users.User) (users.User, error) {
	var query []string
	var upq string
	if user.FirstName != "" {
		query = append(query, "first_name = :first_name,")
	}
	if user.LastName != "" {
		query = append(query, "last_name = :last_name,")
	}
	if user.Credentials.Username != "" {
		query = append(query, "username = :username,")
	}
	if user.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(user.Tags) > 0 {
		query = append(query, "tags = :tags,")
	}
	if user.Role != users.AllRole {
		query = append(query, "role = :role,")
	}

	if user.ProfilePicture.String() != "" {
		query = append(query, "profile_picture = :profile_picture,")
	}

	if user.Email != "" {
		query = append(query, "email = :email,")
	}

	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE users SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, secret, metadata, status, created_at, updated_at, updated_by, last_name, first_name, username, profile_picture, email`, upq)

	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) update(ctx context.Context, user users.User, query string) (users.User, error) {
	dbc, err := toDBUser(user)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbc = DBUser{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}

		return ToUser(dbc)
	}

	return users.User{}, repoerr.ErrNotFound
}

func (repo *userRepo) UpdateSecret(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, email, metadata, status, created_at, updated_at, updated_by, first_name, last_name, username`
	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) ChangeStatus(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE users SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, tags, email, metadata, status, created_at, updated_at, updated_by, first_name, last_name, username`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) Delete(ctx context.Context, id string) error {
	q := "DELETE FROM users AS u  WHERE u.id = $1 ;"

	result, err := repo.Repository.DB.ExecContext(ctx, q, id)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *userRepo) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	query, err := PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT u.id, u.username, u.first_name, u.username, u.created_at, u.updated_at FROM users u %s LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		dbc := DBUser{}
		if err := rows.StructScan(&dbc); err != nil {
			return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToUser(dbc)
		if err != nil {
			return users.UsersPage{}, err
		}

		items = append(items, c)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM users c %s;`, tq)
	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
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
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT u.id, u.username, u.tags, u.email, u.metadata, u.status, u.role, u.first_name, u.last_name, u.username,
					u.created_at, u.updated_at, COALESCE(u.updated_by, '') AS updated_by FROM users u %s ORDER BY u.created_at LIMIT :limit OFFSET :offset;`, query)

	dbPage, err := ToDBUsersPage(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []users.User
	for rows.Next() {
		dbc := DBUser{}
		if err := rows.StructScan(&dbc); err != nil {
			return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := ToUser(dbc)
		if err != nil {
			return users.UsersPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
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
	q := `SELECT id, tags, email, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, username
        FROM users WHERE email = :email AND status = :status`

	dbc := DBUser{
		Email:  email,
		Status: users.EnabledStatus,
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer row.Close()

	dbc = DBUser{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		return ToUser(dbc)
	}

	return users.User{}, repoerr.ErrNotFound
}

type DBUser struct {
	ID             string           `db:"id"`
	Domain         string           `db:"domain_id"`
	Secret         string           `db:"secret"`
	Metadata       []byte           `db:"metadata,omitempty"`
	Tags           pgtype.TextArray `db:"tags,omitempty"` // Tags
	CreatedAt      time.Time        `db:"created_at,omitempty"`
	UpdatedAt      sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy      *string          `db:"updated_by,omitempty"`
	Groups         []groups.Group   `db:"groups,omitempty"`
	Status         users.Status     `db:"status,omitempty"`
	Role           *users.Role      `db:"role,omitempty"`
	Username       string           `db:"username, omitempty"`
	FirstName      string           `db:"first_name, omitempty"`
	LastName       string           `db:"last_name, omitempty"`
	ProfilePicture string           `db:"profile_picture, omitempty"`
	Email          string           `db:"email,omitempty"`
}

func toDBUser(u users.User) (DBUser, error) {
	data := []byte("{}")
	if len(u.Metadata) > 0 {
		b, err := json.Marshal(u.Metadata)
		if err != nil {
			return DBUser{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
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

	return DBUser{
		ID:             u.ID,
		Tags:           tags,
		Secret:         u.Credentials.Secret,
		Metadata:       data,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		Status:         u.Status,
		Role:           &u.Role,
		LastName:       u.LastName,
		FirstName:      u.FirstName,
		Username:       u.Credentials.Username,
		ProfilePicture: u.ProfilePicture.String(),
		Email:          u.Email,
	}, nil
}

func ToUser(dbu DBUser) (users.User, error) {
	var metadata users.Metadata
	if dbu.Metadata != nil {
		if err := json.Unmarshal([]byte(dbu.Metadata), &metadata); err != nil {
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
		updatedAt = dbu.UpdatedAt.Time
	}

	profilePicture, err := url.Parse(dbu.ProfilePicture)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
	}

	user := users.User{
		ID:        dbu.ID,
		FirstName: dbu.FirstName,
		LastName:  dbu.LastName,
		Credentials: users.Credentials{
			Username: dbu.Username,
			Secret:   dbu.Secret,
		},
		Email:          dbu.Email,
		Metadata:       metadata,
		CreatedAt:      dbu.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		Status:         dbu.Status,
		Tags:           tags,
		ProfilePicture: *profilePicture,
	}
	if dbu.Role != nil {
		user.Role = *dbu.Role
	}
	return user, nil
}

type DBUsersPage struct {
	Total     uint64       `db:"total"`
	Limit     uint64       `db:"limit"`
	Offset    uint64       `db:"offset"`
	FirstName string       `db:"first_name"`
	LastName  string       `db:"last_name"`
	Username  string       `db:"username"`
	Id        string       `db:"id"`
	Email     string       `db:"email"`
	Metadata  []byte       `db:"metadata"`
	Tag       string       `db:"tag"`
	GroupID   string       `db:"group_id"`
	Role      users.Role   `db:"role"`
	Status    users.Status `db:"status"`
}

func ToDBUsersPage(pm users.Page) (DBUsersPage, error) {
	_, data, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return DBUsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return DBUsersPage{
		FirstName: pm.FirstName,
		LastName:  pm.LastName,
		Username:  pm.Username,
		Email:     pm.Email,
		Id:        pm.Id,
		Metadata:  data,
		Total:     pm.Total,
		Offset:    pm.Offset,
		Limit:     pm.Limit,
		Status:    pm.Status,
		Tag:       pm.Tag,
		Role:      pm.Role,
	}, nil
}

func PageQuery(pm users.Page) (string, error) {
	mq, _, err := postgres.CreateMetadataQuery("", pm.Metadata)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

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
	if pm.Id != "" {
		query = append(query, "id ILIKE '%' || :id || '%'")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}
	if pm.Role != users.AllRole {
		query = append(query, "u.role = :role")
	}
	if pm.Email != "" {
		query = append(query, "email ILIKE '%' || :email || '%'")
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
	if pm.Status != users.AllStatus {
		query = append(query, "u.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "u.domain_id = :domain_id")
	}
	var emq string
	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	} else if mq != "" {
		emq = fmt.Sprintf("WHERE %s", mq)
	}
	return emq, nil
}

func applyOrdering(emq string, pm users.Page) string {
	switch pm.Order {
	case "username", "first_name", "email", "last_name", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}
