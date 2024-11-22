// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/roles"
)

var _ roles.Repository = (*Repository)(nil)

type Repository struct {
	db                 postgres.Database
	tableNamePrefix    string
	entityTableName    string
	entityIDColumnName string
}

// NewRepository instantiates a PostgreSQL
// implementation of Roles repository.
func NewRepository(db postgres.Database, tableNamePrefix, entityTableName, entityIDColumnName string) Repository {
	return Repository{
		db:                 db,
		tableNamePrefix:    tableNamePrefix,
		entityTableName:    entityTableName,
		entityIDColumnName: entityIDColumnName,
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
	MemberID string `db:"member_id"`
}

func toDBRoles(role roles.Role) dbRole {
	var createdBy *string
	if role.CreatedBy != "" {
		createdBy = &role.UpdatedBy
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
func (repo *Repository) AddRoles(ctx context.Context, rps []roles.RoleProvision) ([]roles.Role, error) {

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return []roles.Role{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	var retRoles []roles.Role

	for _, rp := range rps {

		q := fmt.Sprintf(`INSERT INTO %s_roles (id, name, entity_id, created_by, created_at, updated_by, updated_at)
        VALUES (:id, :name, :entity_id, :created_by, :created_at, :updated_by, :updated_at);`, repo.tableNamePrefix)

		if _, err := tx.NamedExec(q, toDBRoles(rp.Role)); err != nil {
			return []roles.Role{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
		}

		retRoles = append(retRoles, rp.Role)

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
				return []roles.Role{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
		}

		if len(rp.OptionalMembers) > 0 {
			mq := fmt.Sprintf(`INSERT INTO %s_role_members (role_id, member_id)
					VALUES (:role_id, :member_id)
					RETURNING role_id, member_id`, repo.tableNamePrefix)

			rMems := []dbRoleMember{}
			for _, m := range rp.OptionalMembers {
				rMems = append(rMems, dbRoleMember{
					RoleID:   rp.ID,
					MemberID: m,
				})
			}
			if _, err := tx.NamedExec(mq, rMems); err != nil {
				return []roles.Role{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return []roles.Role{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return retRoles, nil
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

// Update only role name, don't update ID
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

func (repo *Repository) RetrieveRoleByEntityIDAndName(ctx context.Context, entityID, roleName string) (roles.Role, error) {
	q := fmt.Sprintf(`SELECT id, name, entity_id, created_by, created_at, updated_by, updated_at
        FROM %s_roles WHERE entity_id = :entity_id and name = :name`, repo.tableNamePrefix)

	dbr := dbRole{
		EntityID: entityID,
		Name:     roleName,
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

	return repo.RoleListActions(ctx, role.ID)
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
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s_role_actions WHERE role_id = :role_id AND action IN (:actions)`, repo.tableNamePrefix)

	params := map[string]interface{}{
		"role_id": roleID,
		"actions": actions,
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
	mq := fmt.Sprintf(`INSERT INTO %s_role_members (role_id, member_id)
        VALUES (:role_id, :member_id)
        RETURNING role_id, member_id`, repo.tableNamePrefix)

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
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s_role_members WHERE role_id = :role_id AND action IN (:members)`, repo.tableNamePrefix)

	params := map[string]interface{}{
		"role_id": roleID,
		"members": members,
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

func (repo *Repository) RemoveMemberFromAllRoles(ctx context.Context, memberID string) (err error) {

	return nil
}
