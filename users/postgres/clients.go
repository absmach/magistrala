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
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
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

	q := `INSERT INTO clients (id, name, tags, identity, secret, metadata, created_at, status, role, first_name, last_name, user_name, profile_picture)
        VALUES (:id, :name, :tags, :identity, :secret, :metadata, :created_at, :status, :role, :first_name, :last_name, :user_name, :profile_picture)
        RETURNING id, name, tags, identity, metadata, status, created_at, first_name, last_name, user_name, profile_picture`

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
	q := `SELECT id, name, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name, profile_picture
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
	q := `SELECT id, name, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name, profile_picture
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

func (repo *userRepo) RetrieveAll(ctx context.Context, pm mgclients.Page) (users.UsersPage, error) {
	query, err := pgclients.PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, c.status, c.role, c.first_name, c.last_name, c.user_name,
    c.created_at, c.updated_at, c.profile_picture, COALESCE(c.updated_by, '') AS updated_by 
    FROM clients c %s ORDER BY c.created_at LIMIT :limit OFFSET :offset;`, query)

	fmt.Println("query alone", query)

	dbPage, err := ToDBUsersPage(pm)
	fmt.Println("dbPage", dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}

	rows, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var usersList []users.User
	for rows.Next() {
		dbc := DBUser{}
		if err := rows.StructScan(&dbc); err != nil {
			return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		u, err := ToUser(dbc)
		if err != nil {
			return users.UsersPage{}, err
		}

		usersList = append(usersList, u)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients u %s;`, query)

	total, err := postgres.Total(ctx, repo.Repository.DB, cq, dbPage)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := users.UsersPage{
		Page: mgclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
		Users: usersList,
	}

	return page, nil
}

func (repo *userRepo) UpdateRole(ctx context.Context, user users.User) (users.User, error) {
	query := `UPDATE clients SET role = :role, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, status, role, first_name, last_name, user_name, created_at, updated_at, updated_by, profile_picture`

	dbc, err := toDBUser(user)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, query, dbc)
	if err != nil {
		return users.User{}, postgres.HandleError(err, repoerr.ErrUpdateEntity)
	}

	defer row.Close()
	if ok := row.Next(); !ok {
		return users.User{}, errors.Wrap(repoerr.ErrNotFound, row.Err())
	}
	dbc = DBUser{}
	if err := row.StructScan(&dbc); err != nil {
		return users.User{}, err
	}

	return ToUser(dbc)
}

func (repo *userRepo) UpdateUserNames(ctx context.Context, user users.User) (users.User, error) {
	if user.FirstName != "" && user.LastName != "" {
		return users.User{}, repoerr.ErrMissingNames
	}

	user.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)

	q := `UPDATE clients SET first_name = :first_name, last_name = :last_name, user_name = :user_name, name = :name, updated_at = :updated_at
        WHERE id = :id RETURNING id, name, tags, identity, secret, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`

	dbc := DBUser{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UserName:  user.UserName,
		Name:      user.Name,
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
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

func (repo *userRepo) UpdateProfilePicture(ctx context.Context, user users.User) (users.User, error) {
	profilePictureURL, err := repo.gcStorage.UploadProfilePicture(ctx, strings.NewReader(user.ProfilePicture), user.ID)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	user.ProfilePicture = profilePictureURL

	q := `UPDATE clients SET profile_picture = :profile_picture, updated_at = :updated_at
		WHERE id = :id RETURNING id, name, tags, identity, secret, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name, profile_picture`

	dbc, err := toDBUser(user)
	if err != nil {
		return users.User{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	row, err := repo.Repository.DB.NamedQueryContext(ctx, q, dbc)
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

func (repo *userRepo) Update(ctx context.Context, user users.User) (users.User, error) {
	if user.FirstName != "" && user.LastName != "" {
		return users.User{}, repoerr.ErrMissingNames
	}

	user.Name = fmt.Sprintf("%s %s", user.FirstName, user.LastName)

	var query []string
	var upq string
	if user.Name != "" {
		query = append(query, "name = :name,")
	}
	if user.FirstName != "" {
		query = append(query, "first_name = :first_name,")
	}
	if user.LastName != "" {
		query = append(query, "last_name = :last_name,")
	}
	if user.UserName != "" {
		query = append(query, "user_name = :user_name,")
	}
	if user.Metadata != nil {
		query = append(query, "metadata = :metadata,")
	}
	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE clients SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, secret, metadata, status, created_at, updated_at, updated_by, last_name, first_name, user_name`, upq)
	user.Status = mgclients.EnabledStatus
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

func (repo *userRepo) UpdateIdentity(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE clients SET identity = :identity, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`
	user.Status = mgclients.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) UpdateSecret(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE clients SET secret = :secret, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`
	user.Status = mgclients.EnabledStatus
	return repo.update(ctx, user, q)
}

func (repo *userRepo) ChangeStatus(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE clients SET status = :status, updated_at = :updated_at, updated_by = :updated_by
		WHERE id = :id
        RETURNING id, name, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`

	return repo.update(ctx, user, q)
}

func (repo *userRepo) UpdateTags(ctx context.Context, user users.User) (users.User, error) {
	q := `UPDATE clients SET tags = :tags, updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id AND status = :status
        RETURNING id, name, tags, identity, metadata, status, created_at, updated_at, updated_by, first_name, last_name, user_name`
	user.Status = mgclients.EnabledStatus
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

func (repo *userRepo) SearchUsers(ctx context.Context, pm mgclients.Page) (users.UsersPage, error) {
	query, err := pgclients.PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	tq := query
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.created_at, c.updated_at FROM clients c %s LIMIT :limit OFFSET :offset;`, query)

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
		Page: mgclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *userRepo) RetrieveAllByIDs(ctx context.Context, pm mgclients.Page) (users.UsersPage, error) {
	if (len(pm.IDs) == 0) && (pm.Domain == "") {
		return users.UsersPage{
			Page: mgclients.Page{Total: pm.Total, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}
	query, err := pgclients.PageQuery(pm)
	if err != nil {
		return users.UsersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	query = applyOrdering(query, pm)

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, c.status, c.role, c.first_name, c.last_name, c.user_name,
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
		Page: mgclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (repo *userRepo) RetrieveByIdentity(ctx context.Context, identity string) (users.User, error) {
	q := `SELECT id, name, tags, identity, secret, metadata, created_at, updated_at, updated_by, status, role, first_name, last_name, user_name
        FROM clients WHERE identity = :identity AND status = :status`

	dbc := DBUser{
		Identity: identity,
		Status:   mgclients.EnabledStatus,
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
	Name           string           `db:"name,omitempty"`
	Identity       string           `db:"identity"`
	Domain         string           `db:"domain_id"`
	Secret         string           `db:"secret"`
	Metadata       []byte           `db:"metadata,omitempty"`
	Tags           pgtype.TextArray `db:"tags,omitempty"` // Tags
	CreatedAt      time.Time        `db:"created_at,omitempty"`
	UpdatedAt      sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy      *string          `db:"updated_by,omitempty"`
	Groups         []groups.Group   `db:"groups,omitempty"`
	Status         mgclients.Status `db:"status,omitempty"`
	Role           *mgclients.Role  `db:"role,omitempty"`
	UserName       string           `db:"user_name, omitempty"`
	FirstName      string           `db:"first_name, omitempty"`
	LastName       string           `db:"last_name, omitempty"`
	ProfilePicture string           `db:"profile_picture, omitempty"`
}

func toDBUser(c users.User) (DBUser, error) {
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
		ID:             c.ID,
		Name:           c.Name,
		Tags:           tags,
		Identity:       c.Credentials.Identity,
		Secret:         c.Credentials.Secret,
		Metadata:       data,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
		Status:         c.Status,
		Role:           &c.Role,
		LastName:       c.LastName,
		FirstName:      c.FirstName,
		UserName:       c.UserName,
		ProfilePicture: c.ProfilePicture,
	}, nil
}

func ToUser(dbu DBUser) (users.User, error) {
	var metadata users.Metadata
	if dbu.Metadata != nil {
		if err := json.Unmarshal(dbu.Metadata, &metadata); err != nil {
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
		Name:      dbu.Name,
		FirstName: dbu.FirstName,
		LastName:  dbu.LastName,
		UserName:  dbu.UserName,
		Credentials: users.Credentials{
			Identity: dbu.Identity,
			Secret:   dbu.Secret,
		},
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
	Total    uint64           `db:"total"`
	Limit    uint64           `db:"limit"`
	Offset   uint64           `db:"offset"`
	Name     string           `db:"name"`
	Id       string           `db:"id"`
	Identity string           `db:"identity"`
	Metadata []byte           `db:"metadata"`
	Tag      string           `db:"tag"`
	GroupID  string           `db:"group_id"`
	Role     mgclients.Role   `db:"role"`
	Status   mgclients.Status `db:"status"`
}

func ToDBUsersPage(pm mgclients.Page) (DBUsersPage, error) {
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

func applyOrdering(emq string, pm mgclients.Page) string {
	switch pm.Order {
	case "name", "identity", "created_at", "updated_at":
		emq = fmt.Sprintf("%s ORDER BY %s", emq, pm.Order)
		if pm.Dir == api.AscDir || pm.Dir == api.DescDir {
			emq = fmt.Sprintf("%s %s", emq, pm.Dir)
		}
	}
	return emq
}
