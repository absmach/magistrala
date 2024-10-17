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
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/storage"
	"github.com/jackc/pgtype"
)

type userRepo struct {
	Repository users.UserRepository
	gcStorage  storage.Storage
}

func NewRepository(db postgres.Database, gcp storage.Storage) users.Repository {
	return &userRepo{
		Repository: users.UserRepository{DB: db},
		gcStorage:  gcp,
	}
}

func (repo *userRepo) Save(ctx context.Context, c users.User) (users.User, error) {
	if c.ProfilePicture != "" {
		profilePictureURL, err := repo.gcStorage.UploadProfilePicture(ctx, strings.NewReader(c.ProfilePicture), c.ID)
		if err != nil {
			return users.User{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}

		c.ProfilePicture = profilePictureURL
	}

	q := `INSERT INTO clients (id, tags, identity, secret, metadata, created_at, status, role, first_name, last_name, user_name, profile_picture)
        VALUES (:id, :tags, :identity, :secret, :metadata, :created_at, :status, :role, :first_name, :last_name, :user_name, :profile_picture)
        RETURNING id, tags, identity, metadata, created_at, status, first_name, last_name, user_name, profile_picture`

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
	q := "SELECT 1 FROM clients WHERE id = $1 AND role = $2"
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
	q := `SELECT id, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name, profile_picture
        FROM clients WHERE id = :id`

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

func (repo *userRepo) RetrieveByUserName(ctx context.Context, userName string) (users.User, error) {
	q := `SELECT id, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name, profile_picture
        FROM clients WHERE user_name = :user_name`

	dbc := DBUser{
		UserName: userName,
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

	q := fmt.Sprintf(`SELECT c.id, c.tags, c.identity, c.metadata, c.status, c.role, c.first_name, c.last_name, c.user_name,
    c.created_at, c.updated_at, c.profile_picture, COALESCE(c.updated_by, '') AS updated_by 
    FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

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

func (repo *userRepo) UpdateUserNames(ctx context.Context, user users.User) (users.User, error) {
	if user.FirstName != "" && user.LastName != "" {
		return users.User{}, repoerr.ErrMissingNames
	}

	q := `UPDATE clients SET first_name = :first_name, last_name = :last_name, user_name = :user_name, identity = :identity, updated_at = :updated_at, updated_by = :updated_by,
        WHERE id = :id AND status = :status
		RETURNING id, tags, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name, identity`

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
		UserName:  user.Credentials.UserName,
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
	if user.Credentials.UserName != "" {
		query = append(query, "user_name = :user_name,")
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

	profilePictureURL, err := repo.gcStorage.UploadProfilePicture(ctx, strings.NewReader(user.ProfilePicture), user.ID)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	user.ProfilePicture = profilePictureURL

	if user.ProfilePicture != "" {
		query = append(query, "profile_picture = :profile_picture,")
	}

	if user.Identity != "" {
		query = append(query, "identity = :identity,")
	}

	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, secret, metadata, status, created_at, updated_at, updated_by, last_name, first_name, user_name, profile_picture, identity`, upq)
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
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`
	user.Status = users.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) ChangeStatus(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE clients SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) Delete(ctx context.Context, id string) error {
	var profilePictureURL string

	q := `SELECT profile_picture FROM clients WHERE id = $1`

	err := repo.Repository.DB.QueryRowxContext(ctx, q, id).Scan(&profilePictureURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return repoerr.ErrNotFound
		}
		return postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	if profilePictureURL != "" {
		if err := repo.gcStorage.DeleteProfilePicture(ctx, profilePictureURL); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, fmt.Errorf("failed to delete profile picture: %v", err))
		}
	}

	q = "DELETE FROM clients AS c  WHERE c.id = $1 ;"

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

	q := fmt.Sprintf(`SELECT c.id, c.user_name, c.first_name, c.user_name, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, query)

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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, tq)
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

	q := fmt.Sprintf(`SELECT c.id, c.user_name, c.tags, c.identity, c.metadata, c.status, c.role, c.first_name, c.last_name, c.user_name,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

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

func (repo *userRepo) RetrieveByIdentity(ctx context.Context, identity string) (users.User, error) {
	q := `SELECT id, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name
        FROM clients WHERE identity = :identity AND status = :status`

	dbc := DBUser{
		Identity: identity,
		Status:   users.EnabledStatus,
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
	UserName       string           `db:"user_name, omitempty"`
	FirstName      string           `db:"first_name, omitempty"`
	LastName       string           `db:"last_name, omitempty"`
	ProfilePicture string           `db:"profile_picture, omitempty"`
	Identity       string           `db:"identity,omitempty"`
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
		UserName:       u.Credentials.UserName,
		ProfilePicture: u.ProfilePicture,
		Identity:       u.Identity,
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

	user := users.User{
		ID:        dbu.ID,
		FirstName: dbu.FirstName,
		LastName:  dbu.LastName,
		Credentials: users.Credentials{
			UserName: dbu.UserName,
			Secret:   dbu.Secret,
		},
		Identity:       dbu.Identity,
		Metadata:       metadata,
		CreatedAt:      dbu.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		Status:         dbu.Status,
		Tags:           tags,
		ProfilePicture: dbu.ProfilePicture,
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
	UserName  string       `db:"user_name"`
	Id        string       `db:"id"`
	Identity  string       `db:"identity"`
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
		UserName:  pm.UserName,
		Identity:  pm.Identity,
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
	if pm.UserName != "" {
		query = append(query, "user_name ILIKE '%' || :user_name || '%'")
	}
	if pm.Id != "" {
		query = append(query, "id ILIKE '%' || :id || '%'")
	}
	if pm.Tag != "" {
		query = append(query, "EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || :tag || '%')")
	}
	if pm.Role != users.AllRole {
		query = append(query, "c.role = :role")
	}
	if pm.Identity != "" {
		query = append(query, "identity ILIKE '%' || :identity || '%'")
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
		query = append(query, "c.status = :status")
	}
	if pm.Domain != "" {
		query = append(query, "c.domain_id = :domain_id")
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
	case "user_name", "first_name", "identity", "last_name", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}
