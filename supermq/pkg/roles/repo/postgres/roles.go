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
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
)

var _ roles.Repository = (*Repository)(nil)

type Repository struct {
	db                   postgres.Database
	tableNamePrefix      string
	entityTableName      string
	entityIDColumnName   string
	membersListBaseQuery string
}

// NewRepository instantiates a PostgreSQL
// implementation of Roles repository.
func NewRepository(db postgres.Database, entityType, tableNamePrefix, entityTableName, entityIDColumnName string) Repository {
	var membersListBaseQuery string

	switch entityType {
	case policies.ChannelType:
		membersListBaseQuery = channelMembersListBaseQuery()
	case policies.ClientType:
		membersListBaseQuery = clientMembersListBaseQuery()
	case policies.GroupType:
		membersListBaseQuery = groupMembersListBaseQuery()
	case policies.DomainType:
		membersListBaseQuery = domainMembersListBaseQuery()
	}

	return Repository{
		db:                   db,
		tableNamePrefix:      tableNamePrefix,
		entityTableName:      entityTableName,
		entityIDColumnName:   entityIDColumnName,
		membersListBaseQuery: membersListBaseQuery,
	}
}

type dbPage struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	EntityID string `db:"entity_id"`
	RoleID   string `db:"role_id"`
	Limit    uint64 `db:"limit"`
	Offset   uint64 `db:"offset"`
}
type dbRole struct {
	ID        string       `db:"id"`
	Name      string       `db:"name"`
	EntityID  string       `db:"entity_id"`
	CreatedBy *string      `db:"created_by"`
	CreatedAt sql.NullTime `db:"created_at"`
	UpdatedBy *string      `db:"updated_by"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}

type dbMemberRoles struct {
	MemberID string          `db:"member_id,omitempty"`
	Roles    json.RawMessage `db:"roles,omitempty"`
}

type dbEntityActionRole struct {
	EntityID string `db:"entity_id"`
	Action   string `db:"action"`
	RoleID   string `db:"role_id"`
}
type dbEntityMemberRole struct {
	EntityID string `db:"entity_id"`
	MemberID string `db:"member_id"`
	RoleID   string `db:"role_id"`
}

func dbToEntityActionRole(dbs []dbEntityActionRole) []roles.EntityActionRole {
	var r []roles.EntityActionRole
	for _, d := range dbs {
		r = append(r, roles.EntityActionRole{
			EntityID: d.EntityID,
			Action:   d.Action,
			RoleID:   d.RoleID,
		})
	}
	return r
}

func dbToEntityMemberRole(dbs []dbEntityMemberRole) []roles.EntityMemberRole {
	var r []roles.EntityMemberRole
	for _, d := range dbs {
		r = append(r, roles.EntityMemberRole{
			EntityID: d.EntityID,
			MemberID: d.MemberID,
			RoleID:   d.RoleID,
		})
	}
	return r
}

type dbRoleAction struct {
	RoleID string `db:"role_id"`
	Action string `db:"action"`
}

type dbRoleMember struct {
	RoleID   string `db:"role_id"`
	EntityID string `db:"entity_id"`
	MemberID string `db:"member_id"`
}

func toDBRoles(role roles.Role) dbRole {
	var createdBy *string
	if role.CreatedBy != "" {
		createdBy = &role.CreatedBy
	}
	var createdAt sql.NullTime
	if role.CreatedAt != (time.Time{}) && !role.CreatedAt.IsZero() {
		createdAt = sql.NullTime{Time: role.CreatedAt, Valid: true}
	}

	var updatedBy *string
	if role.UpdatedBy != "" {
		updatedBy = &role.UpdatedBy
	}
	var updatedAt sql.NullTime
	if role.UpdatedAt != (time.Time{}) && !role.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{Time: role.UpdatedAt, Valid: true}
	}

	return dbRole{
		ID:        role.ID,
		Name:      role.Name,
		EntityID:  role.EntityID,
		CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}
}

func toRole(r dbRole) roles.Role {
	var createdBy string
	if r.CreatedBy != nil {
		createdBy = *r.CreatedBy
	}
	var createdAt time.Time
	if r.CreatedAt.Valid {
		createdAt = r.CreatedAt.Time
	}

	var updatedBy string
	if r.UpdatedBy != nil {
		updatedBy = *r.UpdatedBy
	}
	var updatedAt time.Time
	if r.UpdatedAt.Valid {
		updatedAt = r.UpdatedAt.Time
	}

	return roles.Role{
		ID:        r.ID,
		Name:      r.Name,
		EntityID:  r.EntityID,
		CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedBy: updatedBy,
		UpdatedAt: updatedAt,
	}
}

func (repo *Repository) AddRoles(ctx context.Context, rps []roles.RoleProvision) ([]roles.RoleProvision, error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return []roles.RoleProvision{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	for _, rp := range rps {
		q := fmt.Sprintf(`INSERT INTO %s_roles (id, name, entity_id, created_by, created_at, updated_by, updated_at)
        VALUES (:id, :name, :entity_id, :created_by, :created_at, :updated_by, :updated_at);`, repo.tableNamePrefix)

		if _, err := tx.NamedExec(q, toDBRoles(rp.Role)); err != nil {
			return []roles.RoleProvision{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
		}

		if len(rp.OptionalActions) > 0 {
			capq := fmt.Sprintf(`INSERT INTO %s_role_actions (role_id, action)
        				VALUES (:role_id, :action)
        				RETURNING role_id, action`, repo.tableNamePrefix)

			rCaps := []dbRoleAction{}
			for _, cap := range rp.OptionalActions {
				rCaps = append(rCaps, dbRoleAction{
					RoleID: rp.ID,
					Action: string(cap),
				})
			}
			if _, err := tx.NamedExec(capq, rCaps); err != nil {
				return []roles.RoleProvision{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
		}

		if len(rp.OptionalMembers) > 0 {
			mq := fmt.Sprintf(`INSERT INTO %s_role_members (role_id, entity_id, member_id)
					VALUES (:role_id, :entity_id, :member_id)
					RETURNING role_id, entity_id, member_id`, repo.tableNamePrefix)

			rMems := []dbRoleMember{}
			for _, m := range rp.OptionalMembers {
				rMems = append(rMems, dbRoleMember{
					RoleID:   rp.ID,
					MemberID: m,
					EntityID: rp.EntityID,
				})
			}
			if _, err := tx.NamedExec(mq, rMems); err != nil {
				return []roles.RoleProvision{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return []roles.RoleProvision{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return rps, nil
}

func (repo *Repository) RemoveRoles(ctx context.Context, roleIDs []string) error {
	q := fmt.Sprintf("DELETE FROM %s_roles  WHERE id = ANY(:role_ids) ;", repo.tableNamePrefix)

	params := map[string]interface{}{
		"role_ids": roleIDs,
	}
	result, err := repo.db.NamedExecContext(ctx, q, params)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

// Update only role name, don't update ID.
func (repo *Repository) UpdateRole(ctx context.Context, role roles.Role) (roles.Role, error) {
	var query []string
	var upq string
	if role.Name != "" {
		query = append(query, "name = :name,")
	}

	if len(query) > 0 {
		upq = strings.Join(query, " ")
	}

	q := fmt.Sprintf(`UPDATE %s_roles SET %s updated_at = :updated_at, updated_by = :updated_by
        WHERE id = :id
        RETURNING id, name, entity_id, created_by, created_at, updated_by, updated_at`,
		repo.tableNamePrefix, upq)

	row, err := repo.db.NamedQueryContext(ctx, q, toDBRoles(role))
	if err != nil {
		return roles.Role{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer row.Close()

	dbr := dbRole{}
	if row.Next() {
		if err := row.StructScan(&dbr); err != nil {
			return roles.Role{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		return toRole(dbr), nil
	}

	return roles.Role{}, repoerr.ErrNotFound
}

func (repo *Repository) RetrieveRole(ctx context.Context, roleID string) (roles.Role, error) {
	q := fmt.Sprintf(`SELECT id, name, entity_id, created_by, created_at, updated_by, updated_at
        FROM %s_roles WHERE id = :id`, repo.tableNamePrefix)

	dbr := dbRole{
		ID: roleID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return roles.Role{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbr = dbRole{}
	if rows.Next() {
		if err = rows.StructScan(&dbr); err != nil {
			return roles.Role{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		return toRole(dbr), nil
	}

	return roles.Role{}, repoerr.ErrNotFound
}

func (repo *Repository) RetrieveEntityRole(ctx context.Context, entityID, roleID string) (roles.Role, error) {
	q := fmt.Sprintf(`SELECT id, name, entity_id, created_by, created_at, updated_by, updated_at
        FROM %s_roles WHERE entity_id = :entity_id and id = :id`, repo.tableNamePrefix)

	dbr := dbRole{
		EntityID: entityID,
		ID:       roleID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbr)
	if err != nil {
		return roles.Role{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbr = dbRole{}
	if rows.Next() {
		if err = rows.StructScan(&dbr); err != nil {
			return roles.Role{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		return toRole(dbr), nil
	}

	return roles.Role{}, repoerr.ErrNotFound
}

func (repo *Repository) RetrieveAllRoles(ctx context.Context, entityID string, limit, offset uint64) (roles.RolePage, error) {
	q := fmt.Sprintf(`SELECT id, name, entity_id, created_by, created_at, updated_by, updated_at
    	FROM %s_roles WHERE entity_id = :entity_id ORDER BY created_at LIMIT :limit OFFSET :offset;`, repo.tableNamePrefix)

	dbp := dbPage{
		EntityID: entityID,
		Limit:    limit,
		Offset:   offset,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbp)
	if err != nil {
		return roles.RolePage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	items := []roles.Role{}
	for rows.Next() {
		dbr := dbRole{}
		if err := rows.StructScan(&dbr); err != nil {
			return roles.RolePage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		items = append(items, toRole(dbr))
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM %s_roles WHERE entity_id = :entity_id`, repo.tableNamePrefix)

	total, err := postgres.Total(ctx, repo.db, cq, dbp)
	if err != nil {
		return roles.RolePage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := roles.RolePage{
		Roles:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	return page, nil
}

func (repo *Repository) RoleAddActions(ctx context.Context, role roles.Role, actions []string) (caps []string, err error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return []string{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	capq := fmt.Sprintf(`INSERT INTO %s_role_actions (role_id, action)
	VALUES (:role_id, :action)
	RETURNING role_id, action`, repo.tableNamePrefix)

	rCaps := []dbRoleAction{}
	for _, cap := range actions {
		rCaps = append(rCaps, dbRoleAction{
			RoleID: role.ID,
			Action: string(cap),
		})
	}
	if _, err := tx.NamedExecContext(ctx, capq, rCaps); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExecContext(ctx, upq, toDBRoles(role)); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return actions, nil
}

func (repo *Repository) RoleListActions(ctx context.Context, roleID string) ([]string, error) {
	q := fmt.Sprintf(`SELECT role_id, action FROM %s_role_actions WHERE role_id = :role_id ;`, repo.tableNamePrefix)

	dbrcap := dbRoleAction{
		RoleID: roleID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbrcap)
	if err != nil {
		return []string{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		dbrcap = dbRoleAction{}
		if err := rows.StructScan(&dbrcap); err != nil {
			return []string{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		items = append(items, dbrcap.Action)
	}
	return items, nil
}

func (repo *Repository) RoleCheckActionsExists(ctx context.Context, roleID string, actions []string) (bool, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s_role_actions WHERE role_id = :role_id AND action IN ('%s')`, repo.tableNamePrefix, strings.Join(actions, ","))

	params := map[string]interface{}{
		"role_id": roleID,
	}
	var count int
	query, err := repo.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return false, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	defer query.Close()

	if query.Next() {
		if err := query.Scan(&count); err != nil {
			return false, errors.Wrap(repoerr.ErrViewEntity, err)
		}
	}

	// Check if the count matches the number of actions provided
	if count != len(actions) {
		return false, nil
	}

	return true, nil
}

func (repo *Repository) RoleRemoveActions(ctx context.Context, role roles.Role, actions []string) (err error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	q := fmt.Sprintf(`DELETE FROM %s_role_actions WHERE role_id = :role_id AND action = ANY(:actions)`, repo.tableNamePrefix)

	params := map[string]interface{}{
		"role_id": role.ID,
		"actions": actions,
	}

	if _, err := tx.NamedExec(q, params); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExec(upq, toDBRoles(role)); err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (repo *Repository) RoleRemoveAllActions(ctx context.Context, role roles.Role) error {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	q := fmt.Sprintf(`DELETE FROM %s_role_actions WHERE role_id = :role_id `, repo.tableNamePrefix)

	dbrcap := dbRoleAction{RoleID: role.ID}

	if _, err := tx.NamedExec(q, dbrcap); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExec(upq, toDBRoles(role)); err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (repo *Repository) RoleAddMembers(ctx context.Context, role roles.Role, members []string) ([]string, error) {
	mq := fmt.Sprintf(`INSERT INTO %s_role_members (role_id, entity_id, member_id)
        VALUES (:role_id, :entity_id, :member_id)
        RETURNING role_id, :entity_id, member_id`, repo.tableNamePrefix)

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return []string{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	rMems := []dbRoleMember{}
	for _, m := range members {
		rMems = append(rMems, dbRoleMember{
			RoleID:   role.ID,
			EntityID: role.EntityID,
			MemberID: m,
		})
	}
	if _, err := tx.NamedExec(mq, rMems); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExec(upq, toDBRoles(role)); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return []string{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return members, nil
}

func (repo *Repository) RoleListMembers(ctx context.Context, roleID string, limit, offset uint64) (roles.MembersPage, error) {
	q := fmt.Sprintf(`SELECT role_id, member_id FROM %s_role_members WHERE role_id = :role_id LIMIT :limit OFFSET :offset;`, repo.tableNamePrefix)

	dbp := dbPage{
		RoleID: roleID,
		Limit:  limit,
		Offset: offset,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbp)
	if err != nil {
		return roles.MembersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	items := []string{}
	for rows.Next() {
		dbrmems := dbRoleMember{}
		if err := rows.StructScan(&dbrmems); err != nil {
			return roles.MembersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		items = append(items, dbrmems.MemberID)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM %s_role_members WHERE role_id = :role_id`, repo.tableNamePrefix)

	total, err := postgres.Total(ctx, repo.db, cq, dbp)
	if err != nil {
		return roles.MembersPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return roles.MembersPage{
		Members: items,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
	}, nil
}

func (repo *Repository) RoleCheckMembersExists(ctx context.Context, roleID string, members []string) (bool, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s_role_members WHERE role_id = :role_id AND member_id IN ('%s')`, repo.tableNamePrefix, strings.Join(members, ","))

	params := map[string]interface{}{
		"role_id": roleID,
	}
	var count int
	query, err := repo.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return false, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	defer query.Close()

	if query.Next() {
		if err := query.Scan(&count); err != nil {
			return false, errors.Wrap(repoerr.ErrViewEntity, err)
		}
	}

	if count != len(members) {
		return false, nil
	}

	return true, nil
}

func (repo *Repository) RoleRemoveMembers(ctx context.Context, role roles.Role, members []string) (err error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	q := fmt.Sprintf(`DELETE FROM %s_role_members WHERE role_id = :role_id AND member_id = ANY(:member_ids)`, repo.tableNamePrefix)

	params := map[string]interface{}{
		"role_id":    role.ID,
		"member_ids": members,
	}

	if _, err := tx.NamedExec(q, params); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExec(upq, toDBRoles(role)); err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *Repository) RoleRemoveAllMembers(ctx context.Context, role roles.Role) (err error) {
	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()
	q := fmt.Sprintf(`DELETE FROM %s_role_members WHERE role_id = :role_id `, repo.tableNamePrefix)

	dbrcap := dbRoleAction{RoleID: role.ID}

	if _, err := repo.db.NamedExecContext(ctx, q, dbrcap); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	upq := fmt.Sprintf(`UPDATE %s_roles SET updated_at = :updated_at, updated_by = :updated_by WHERE id = :id;`, repo.tableNamePrefix)
	if _, err := tx.NamedExec(upq, toDBRoles(role)); err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *Repository) RetrieveEntitiesRolesActionsMembers(ctx context.Context, entityIDs []string) ([]roles.EntityActionRole, []roles.EntityMemberRole, error) {
	params := map[string]interface{}{
		"entity_ids": entityIDs,
	}

	clientsActionsRolesQuery := fmt.Sprintf(`SELECT e.%s AS entity_id , era."action" AS "action", er.id AS role_id
								FROM %s e
								JOIN %s_roles er ON er.entity_id  = e.%s
								JOIN %s_role_actions era  ON era.role_id  = er.id
								WHERE e.%s = ANY(:entity_ids);
							`, repo.entityIDColumnName, repo.entityTableName, repo.tableNamePrefix, repo.entityIDColumnName, repo.tableNamePrefix, repo.entityIDColumnName)
	rows, err := repo.db.NamedQueryContext(ctx, clientsActionsRolesQuery, params)
	if err != nil {
		return []roles.EntityActionRole{}, []roles.EntityMemberRole{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	defer rows.Close()
	dbears := []dbEntityActionRole{}
	for rows.Next() {
		dbear := dbEntityActionRole{}
		if err = rows.StructScan(&dbear); err != nil {
			return []roles.EntityActionRole{}, []roles.EntityMemberRole{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		dbears = append(dbears, dbear)
	}
	clientsMembersRolesQuery := fmt.Sprintf(`SELECT e.%s AS entity_id , erm.member_id AS member_id, er.id AS role_id
								FROM %s e
								JOIN %s_roles er ON er.entity_id  = e.%s
								JOIN %s_role_members erm ON erm.role_id = er.id
								WHERE e.%s = ANY(:entity_ids);
								`, repo.entityIDColumnName, repo.entityTableName, repo.tableNamePrefix, repo.entityIDColumnName, repo.tableNamePrefix, repo.entityIDColumnName)

	rows, err = repo.db.NamedQueryContext(ctx, clientsMembersRolesQuery, params)
	if err != nil {
		return []roles.EntityActionRole{}, []roles.EntityMemberRole{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	defer rows.Close()
	dbemrs := []dbEntityMemberRole{}
	for rows.Next() {
		dbemr := dbEntityMemberRole{}
		if err = rows.StructScan(&dbemr); err != nil {
			return []roles.EntityActionRole{}, []roles.EntityMemberRole{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		dbemrs = append(dbemrs, dbemr)
	}
	return dbToEntityActionRole(dbears), dbToEntityMemberRole(dbemrs), nil
}

func (repo *Repository) ListEntityMembers(ctx context.Context, entityID string, pageQuery roles.MembersRolePageQuery) (roles.MembersRolePage, error) {
	dbPageQuery, err := toDBMembersRolePageQuery(pageQuery)
	if err != nil {
		return roles.MembersRolePage{}, err
	}
	dbPageQuery.EntityID = entityID

	entityMembersQuery := fmt.Sprintf(`
		%s
		SELECT
			member_id,
			roles
		FROM
			members
	`, repo.membersListBaseQuery)

	entityMembersQuery = applyConditions(entityMembersQuery, pageQuery)
	entityMembersQuery = applyOrdering(entityMembersQuery, pageQuery)
	entityMembersQuery = applyLimitOffset(entityMembersQuery)

	rows, err := repo.db.NamedQueryContext(ctx, entityMembersQuery, dbPageQuery)
	if err != nil {
		return roles.MembersRolePage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	defer rows.Close()
	mems := []roles.MemberRoles{}
	for rows.Next() {
		var dbmr dbMemberRoles
		if err = rows.StructScan(&dbmr); err != nil {
			return roles.MembersRolePage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		var roleActions []roles.MemberRoleActions
		if err := json.Unmarshal(dbmr.Roles, &roleActions); err != nil {
			return roles.MembersRolePage{}, fmt.Errorf("failed to unmarshal roles JSON: %w", err)
		}
		mems = append(mems, roles.MemberRoles{MemberID: dbmr.MemberID, Roles: roleActions})
	}

	entityMembersCountQuery := fmt.Sprintf(`
		%s
		SELECT
			COUNT(*)
		FROM
			members
	`, repo.membersListBaseQuery)

	entityMembersCountQuery = applyConditions(entityMembersCountQuery, pageQuery)

	total, err := postgres.Total(ctx, repo.db, entityMembersCountQuery, dbPageQuery)
	if err != nil {
		return roles.MembersRolePage{}, err
	}

	return roles.MembersRolePage{
		Total:   total,
		Limit:   pageQuery.Limit,
		Offset:  pageQuery.Offset,
		Members: mems,
	}, nil
}

func (repo *Repository) RemoveEntityMembers(ctx context.Context, entityID string, memberIDs []string) error {
	return nil
}

func (repo *Repository) RemoveMemberFromAllRoles(ctx context.Context, memberID string) (err error) {
	return nil
}

func (repo *Repository) SetMemberListBaseQuery(query string) {
	repo.membersListBaseQuery = query
}

func applyConditions(query string, pageQuery roles.MembersRolePageQuery) string {
	var whereClause []string

	if pageQuery.RoleID != "" {
		whereClause = append(whereClause, " roles @>  :role_id ")
	}
	if pageQuery.RoleName != "" {
		whereClause = append(whereClause, " roles @> :role_name ")
	}
	if len(pageQuery.Actions) != 0 {
		whereClause = append(whereClause, " roles @> :actions ")
	}
	if pageQuery.AccessType != "" {
		whereClause = append(whereClause, " roles @> :access_type ")
	}
	if pageQuery.AccessProviderID != "" {
		whereClause = append(whereClause, " roles @> :access_provider_id ")
	}

	var whereCondition string
	if len(whereClause) != 0 {
		whereCondition = "WHERE " + strings.Join(whereClause, " AND ")
	}

	return fmt.Sprintf(`%s
			%s`, query, whereCondition)
}

func applyOrdering(query string, pageQuery roles.MembersRolePageQuery) string {
	switch pageQuery.Order {
	case "access_provider_id", "role_name", "role_id", "access_type":
		query = fmt.Sprintf("%s ORDER BY %s", query, pageQuery.Order)
		if pageQuery.Dir == api.AscDir || pageQuery.Dir == api.DescDir {
			query = fmt.Sprintf("%s %s", query, pageQuery.Dir)
		}
	}
	return query
}

func applyLimitOffset(query string) string {
	return fmt.Sprintf(`%s
			LIMIT :limit OFFSET :offset`, query)
}

type dbMembersRolePageQuery struct {
	Offset           uint64          `db:"offset"`
	Limit            uint64          `db:"limit"`
	OrderBy          string          `db:"order_by"`
	Direction        string          `db:"dir"`
	AccessProviderID json.RawMessage `db:"access_provider_id"`
	RoleId           json.RawMessage `db:"role_id"`
	RoleName         json.RawMessage `db:"role_name"`
	Actions          json.RawMessage `db:"actions"`
	AccessType       json.RawMessage `db:"access_type"`
	EntityID         string          `db:"entity_id"`
}

func toDBMembersRolePageQuery(pageQuery roles.MembersRolePageQuery) (dbMembersRolePageQuery, error) {
	actions := []byte("{}")
	if len(pageQuery.Actions) != 0 {
		var err error
		jactions := []struct {
			Actions []string `json:"actions"`
		}{
			{
				Actions: pageQuery.Actions,
			},
		}
		actions, err = json.Marshal(jactions)
		if err != nil {
			return dbMembersRolePageQuery{}, err
		}
	}

	accessProviderID := []byte("{}")
	if pageQuery.AccessProviderID != "" {
		accessProviderID = []byte(fmt.Sprintf("[{\"access_provider_id\" : \"%s\"}]", pageQuery.AccessProviderID))
	}

	roleID := []byte("{}")
	if pageQuery.RoleID != "" {
		roleID = []byte(fmt.Sprintf("[{\"role_id\" : \"%s\"}]", pageQuery.RoleID))
	}

	roleName := []byte("{}")
	if pageQuery.RoleName != "" {
		roleName = []byte(fmt.Sprintf("[{\"role_name\" : \"%s\"}]", pageQuery.RoleName))
	}

	accessType := []byte("{}")
	if pageQuery.AccessType != "" {
		accessType = []byte(fmt.Sprintf("[{\"access_type\" : \"%s\"}]", pageQuery.AccessType))
	}

	return dbMembersRolePageQuery{
		Offset:           pageQuery.Offset,
		Limit:            pageQuery.Limit,
		OrderBy:          pageQuery.Order,
		Direction:        pageQuery.Dir,
		AccessProviderID: accessProviderID,
		RoleId:           roleID,
		RoleName:         roleName,
		Actions:          actions,
		AccessType:       accessType,
	}, nil
}

func domainMembersListBaseQuery() string {
	return `
WITH ungrouped_members AS (
    SELECT
        dr.id,
        dr.name,
        drm.member_id,
        ARRAY_AGG(DISTINCT all_actions.action) AS actions,
        'direct' AS access_type,
        '' AS access_provider_id
    FROM
        domains_role_members drm
    JOIN domains_roles dr ON
        dr.id = drm.role_id
    JOIN domains_role_actions dra ON
        dra.role_id = dr.id
    JOIN domains_role_actions all_actions ON
        all_actions.role_id = drm.role_id
    WHERE
        dr.entity_id = :entity_id
    GROUP BY
        dr.id,
        drm.member_id
),
members AS (
    SELECT
        um.member_id,
        JSONB_AGG(
            JSON_BUILD_OBJECT(
                'role_id', um.id,
                'role_name', um.name,
                'actions', um.actions,
                'access_type', um.access_type,
                'access_provider_id', um.access_provider_id
            )
        ) AS roles
    FROM
        ungrouped_members um
    GROUP BY
        um.member_id
)
	`
}

func groupMembersListBaseQuery() string {
	return `
WITH ungrouped_members AS (
    SELECT
        gr."name",
        gr.id,
        grm.member_id,
        ARRAY_AGG(DISTINCT agg_gra."action") AS actions,
        CASE
            WHEN g.id = :entity_id THEN 'direct'
            ELSE 'indirect_group'
        END AS access_type,
        CASE
            WHEN g.id = :entity_id THEN ''
            ELSE g.id
        END AS access_provider_id
    FROM
        "groups" g
    JOIN
        groups_roles gr ON
        gr.entity_id = g.id
    JOIN
        groups_role_members grm ON
        grm.role_id = gr.id
    JOIN
        groups_role_actions gra ON
        gra.role_id = gr.id
    JOIN
        groups_role_actions agg_gra ON
        agg_gra.role_id = gr.id
    WHERE
        g.path @> (
            SELECT
                "path"
            FROM
                "groups"
            WHERE
                id = :entity_id
            LIMIT 1
        )
		AND (
        	g.id = :entity_id
        	OR gra."action" LIKE 'subgroup%'
    	) -- --  If g.id = <entity_id>, it allows all actions. If g.id <> <entity_id>, it only allows actions matching 'subgroup%'.
    GROUP BY
        gr.id,
        grm.member_id,
        g.id
UNION
    SELECT
        dr."name",
        dr.id,
        drm.member_id,
        ARRAY_AGG(DISTINCT agg_dra."action") AS actions,
        'domain' AS access_type,
        d.id AS access_provider_id
    FROM
        "groups" g
    JOIN
        domains d ON
        d.id = g.domain_id
    JOIN
        domains_roles dr ON
        dr.entity_id = d.id
    JOIN
        domains_role_members drm ON
        dr.id = drm.role_id
    JOIN
        domains_role_actions dra ON
        dr.id = dra.role_id
    JOIN
        domains_role_actions agg_dra ON
        agg_dra.role_id = dr.id
    WHERE
        g.id = :entity_id
        AND
        dra."action" LIKE 'group%'
    GROUP BY
        dr.id,
        drm.member_id,
        d.id
),
members AS (
    SELECT
        um.member_id,
        JSONB_AGG(
            JSON_BUILD_OBJECT(
                'role_id', um.id,
                'role_name', um.name,
                'actions', um.actions,
                'access_type', um.access_type,
                'access_provider_id', um.access_provider_id
            )
        ) AS roles
    FROM
        ungrouped_members um
    GROUP BY
        um.member_id
)
	`
}

func clientMembersListBaseQuery() string {
	return `
WITH client_group AS (
    SELECT
        c.id,
        c.parent_group_id,
        c.domain_id,
        g."path" AS parent_group_path
    FROM
        clients c
    LEFT JOIN
		"groups" g ON
        g.id = c.parent_group_id
    WHERE
        c.id = :entity_id
    LIMIT 1
),
ungrouped_members AS (
    SELECT
        cr."name",
        cr.id,
        crm.member_id,
        ARRAY_AGG(DISTINCT cra."action") AS actions,
        'direct' AS access_type,
        '' AS access_provider_id,
		''::::LTREE AS access_provider_path
    FROM
        client_group cg
    JOIN
        clients_roles cr ON
        cr.entity_id = cg.id
    JOIN
		clients_role_members crm ON
        crm.role_id = cr.id
    JOIN
		clients_role_actions cra ON
        cra.role_id = cr.id
    GROUP BY
        cr.id,
        crm.member_id
	UNION
    SELECT
        gr."name",
        gr.id,
        grm.member_id,
        ARRAY_AGG(DISTINCT agg_gra."action") AS actions,
        CASE
            WHEN g.id = cg.parent_group_id THEN 'direct_group'
            ELSE 'indirect_group'
        END AS access_type,
        g.id AS access_provider_id,
		g.path AS access_provider_path
    FROM
        client_group cg
    JOIN
        "groups" g ON
        g.PATH @> cg.parent_group_path
    JOIN
        groups_roles gr ON
        g.id = gr.entity_id
    JOIN
		groups_role_members grm ON
        grm.role_id = gr.id
    JOIN
		groups_role_actions gra ON
        gra.role_id = gr.id
    JOIN
        groups_role_actions agg_gra ON
        agg_gra.role_id = gr.id
    WHERE
        (
            gra."action" LIKE 'client%%'
                AND g.id = cg.parent_group_id
        )
        OR
	 	(
            gra."action" LIKE 'subgroup_client%%'
                AND g.id <> cg.parent_group_id
        )
    GROUP BY
        gr.id,
        grm.member_id,
        g.id,
        cg.parent_group_id
	UNION
    SELECT
        dr."name",
        dr.id,
        drm.member_id,
        ARRAY_AGG(DISTINCT agg_dra."action") AS actions,
        'domain' AS access_type,
        d.id AS access_provider_id,
		''::::LTREE AS access_provider_path
    FROM
        client_group cg
    JOIN
        domains d ON
        d.id = cg.domain_id
    JOIN
        domains_roles dr ON
        dr.entity_id = d.id
    JOIN
        domains_role_members drm ON
        dr.id = drm.role_id
    JOIN
        domains_role_actions dra ON
        dr.id = dra.role_id
    JOIN
        domains_role_actions agg_dra ON
        agg_dra.role_id = dr.id
    WHERE
        dra."action" LIKE 'client%'
    GROUP BY
        dr.id,
        drm.member_id,
        d.id
),
members AS (
    SELECT
        um.member_id,
        JSONB_AGG(
            JSON_BUILD_OBJECT(
                'role_id', um.id,
                'role_name', um.name,
                'actions', um.actions,
                'access_type', um.access_type,
                'access_provider_id', um.access_provider_id,
                'access_provider_path', um.access_provider_path
            )
        ) AS roles
    FROM
        ungrouped_members um
    GROUP BY
        um.member_id
)
	`
}

func channelMembersListBaseQuery() string {
	return `
WITH channel_group AS (
    SELECT
        c.id,
        c.parent_group_id,
        c.domain_id,
        g."path" AS parent_group_path
    FROM
        channels c
    LEFT JOIN
		"groups" g ON
        g.id = c.parent_group_id
    WHERE
        c.id = :entity_id
    LIMIT 1
),
ungrouped_members AS (
    SELECT
        cr."name",
        cr.id,
        crm.member_id,
        ARRAY_AGG(DISTINCT cra."action") AS actions,
        'direct' AS access_type,
        '' AS access_provider_id,
		''::::LTREE AS access_provider_path
    FROM
        channel_group cg
    JOIN
        channels_roles cr ON
        cr.entity_id = cg.id
    JOIN
		channels_role_members crm ON
        crm.role_id = cr.id
    JOIN
		channels_role_actions cra ON
        cra.role_id = cr.id
    GROUP BY
        cr.id,
        crm.member_id
	UNION
    SELECT
        gr."name",
        gr.id,
        grm.member_id,
        ARRAY_AGG(DISTINCT agg_gra."action") AS actions,
        CASE
            WHEN g.id = cg.parent_group_id THEN 'direct_group'
            ELSE 'indirect_group'
        END AS access_type,
        g.id AS access_provider_id,
		g.path AS access_provider_path
    FROM
        channel_group cg
    JOIN
        "groups" g ON
        g.PATH @> cg.parent_group_path
    JOIN
        groups_roles gr ON
        g.id = gr.entity_id
    JOIN
		groups_role_members grm ON
        grm.role_id = gr.id
    JOIN
		groups_role_actions gra ON
        gra.role_id = gr.id
    JOIN
        groups_role_actions agg_gra ON
        agg_gra.role_id = gr.id
    WHERE
        (
            gra."action" LIKE 'channel%%'
                AND g.id = cg.parent_group_id
        )
        OR
	 	(
            gra."action" LIKE 'subgroup_channel%%'
                AND g.id <> cg.parent_group_id
        )
    GROUP BY
        gr.id,
        grm.member_id,
        g.id,
        cg.parent_group_id
	UNION
    SELECT
        dr."name",
        dr.id,
        drm.member_id,
        ARRAY_AGG(DISTINCT agg_dra."action") AS actions,
        'domain' AS access_type,
        d.id AS access_provider_id,
		''::::LTREE AS access_provider_path
    FROM
        channel_group cg
    JOIN
        domains d ON
        d.id = cg.domain_id
    JOIN
        domains_roles dr ON
        dr.entity_id = d.id
    JOIN
        domains_role_members drm ON
        dr.id = drm.role_id
    JOIN
        domains_role_actions dra ON
        dr.id = dra.role_id
    JOIN
        domains_role_actions agg_dra ON
        agg_dra.role_id = dr.id
    WHERE
        dra."action" LIKE 'channel%'
    GROUP BY
        dr.id,
        drm.member_id,
        d.id
),
members AS (
    SELECT
        um.member_id,
        JSONB_AGG(
            JSON_BUILD_OBJECT(
                'role_id', um.id,
                'role_name', um.name,
                'actions', um.actions,
                'access_type', um.access_type,
                'access_provider_id', um.access_provider_id,
                'access_provider_path', um.access_provider_path
            )
        ) AS roles
    FROM
        ungrouped_members um
    GROUP BY
        um.member_id
)
	`
}
