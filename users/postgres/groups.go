// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"
)

var (
	errDeleteGroupDB = errors.New("delete group failed")
	errSelectDb      = errors.New("select group from db error")

	errFK         = "foreign_key_violation"
	errInvalid    = "invalid_text_representation"
	errTruncation = "string_data_right_truncation"
)

var _ users.GroupRepository = (*groupRepository)(nil)

type groupRepository struct {
	db Database
}

// NewGroupRepo instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepo(db Database) users.GroupRepository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, group users.Group) (users.Group, error) {
	var id string
	q := `INSERT INTO groups (name, description, id, owner_id, parent_id, metadata) VALUES (:name, :description, :id, :owner_id, :parent_id, :metadata) RETURNING id`
	if group.ParentID == "" {
		q = `INSERT INTO groups (name, description, id, owner_id, metadata) VALUES (:name, :description, :id, :owner_id, :metadata) RETURNING id`
	}

	dbu, err := toDBGroup(group)
	if err != nil {
		return users.Group{}, err
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return users.Group{}, errors.Wrap(users.ErrMalformedEntity, err)
			case errDuplicate:
				return users.Group{}, errors.Wrap(users.ErrGroupConflict, err)
			}
		}

		return users.Group{}, errors.Wrap(users.ErrCreateGroup, err)
	}

	defer row.Close()
	row.Next()
	if err := row.Scan(&id); err != nil {
		return users.Group{}, err
	}
	group.ID = id
	return group, nil
}

func (gr groupRepository) Update(ctx context.Context, group users.Group) error {
	q := `UPDATE groups SET name = :name, metadata = :metadata, description = :description WHERE id = :id;`
	dbu, err := toDBGroup(group)
	if err != nil {
		return errors.Wrap(users.ErrUpdateGroup, err)
	}

	if _, err := gr.db.NamedExecContext(ctx, q, dbu); err != nil {
		return errors.Wrap(users.ErrUpdateGroup, err)
	}

	return nil
}

func (gr groupRepository) Delete(ctx context.Context, groupID string) error {
	qd := `DELETE FROM groups WHERE id = :id`
	dbg, err := toDBGroup(users.Group{ID: groupID})
	if err != nil {
		return errors.Wrap(errUpdateDB, err)
	}

	res, err := gr.db.NamedExecContext(ctx, qd, dbg)
	if err != nil {
		return errors.Wrap(errDeleteGroupDB, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errDeleteGroupDB, err)
	}

	if cnt != 1 {
		return errors.Wrap(users.ErrDeleteGroupMissing, err)
	}
	return nil
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (users.Group, error) {
	q := `SELECT id, name, owner_id, parent_id, description, metadata FROM groups WHERE id = $1`
	dbu := dbGroup{
		ID: id,
	}

	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return users.Group{}, errors.Wrap(users.ErrNotFound, err)

		}
		return users.Group{}, errors.Wrap(errRetrieveDB, err)
	}

	return toGroup(dbu), nil
}

func (gr groupRepository) RetrieveByName(ctx context.Context, name string) (users.Group, error) {
	q := `SELECT id, name, description, metadata FROM groups WHERE name = $1`

	dbu := dbGroup{
		Name: name,
	}

	if err := gr.db.QueryRowxContext(ctx, q, name).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return users.Group{}, errors.Wrap(users.ErrNotFound, err)

		}
		return users.Group{}, errors.Wrap(errRetrieveDB, err)
	}

	group := toGroup(dbu)
	return group, nil
}

func (gr groupRepository) RetrieveAllWithAncestors(ctx context.Context, groupID string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	_, mq, err := getGroupsMetadataQuery(um)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("WHERE %s", mq)
	}

	cq := fmt.Sprintf("SELECT COUNT(*) FROM groups %s", mq)
	sq := fmt.Sprintf("SELECT id, owner_id, parent_id, name, description, metadata FROM groups %s", mq)
	q := fmt.Sprintf("%s ORDER BY id LIMIT :limit OFFSET :offset", sq)

	if groupID != "" {
		sq = fmt.Sprintf(
			`WITH RECURSIVE subordinates AS (
				SELECT id, owner_id, parent_id, name, description, metadata
				FROM groups
				WHERE id = :id
				UNION
					SELECT groups.id, groups.owner_id, groups.parent_id, groups.name, groups.description, groups.metadata
					FROM groups
					INNER JOIN subordinates s ON s.id = groups.parent_id %s
			)`, mq)
		q = fmt.Sprintf("%s SELECT * FROM subordinates ORDER BY id LIMIT :limit OFFSET :offset", sq)
		cq = fmt.Sprintf("%s SELECT COUNT(*) FROM subordinates", sq)
	}

	dbPage, err := toDBGroupPage("", groupID, offset, limit, um)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	var items []users.Group
	for rows.Next() {
		dbgr := dbGroup{}
		if err := rows.StructScan(&dbgr); err != nil {
			return users.GroupPage{}, errors.Wrap(errSelectDb, err)
		}
		gr := toGroup(dbgr)
		if err != nil {
			return users.GroupPage{}, err
		}
		items = append(items, gr)
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := users.GroupPage{
		Groups: items,
		PageMetadata: users.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveMemberships(ctx context.Context, userID string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	m, mq, err := getGroupsMetadataQuery(um)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.parent_id, g.name, g.description, g.metadata
					  FROM group_relations gr, groups g
					  WHERE gr.group_id = g.id and gr.user_id = :userID
		  			  %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params := map[string]interface{}{
		"userID":   userID,
		"limit":    limit,
		"offset":   offset,
		"metadata": m,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	var items []users.Group
	for rows.Next() {
		dbgr := dbGroup{}
		if err := rows.StructScan(&dbgr); err != nil {
			return users.GroupPage{}, errors.Wrap(errSelectDb, err)
		}
		gr := toGroup(dbgr)
		if err != nil {
			return users.GroupPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*)
					   FROM group_relations gr, groups g
					   WHERE gr.group_id = g.id and gr.user_id = :userID %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return users.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := users.GroupPage{
		Groups: items,
		PageMetadata: users.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (gr groupRepository) Assign(ctx context.Context, userID, groupID string) error {
	dbr, err := toDBGroupRelation(userID, groupID)
	if err != nil {
		return errors.Wrap(users.ErrAssignUserToGroup, err)
	}

	qIns := `INSERT INTO group_relations (group_id, user_id) VALUES (:group_id, :user_id)`
	_, err = gr.db.NamedQueryContext(ctx, qIns, dbr)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return errors.Wrap(users.ErrMalformedEntity, err)
			case errDuplicate:
				return errors.Wrap(users.ErrGroupConflict, err)
			case errFK:
				return errors.Wrap(users.ErrNotFound, err)
			}
		}
		return errors.Wrap(users.ErrAssignUserToGroup, err)
	}

	return nil
}

func (gr groupRepository) Unassign(ctx context.Context, userID, groupID string) error {
	q := `DELETE FROM group_relations WHERE user_id = :user_id AND group_id = :group_id`
	dbr, err := toDBGroupRelation(userID, groupID)
	if err != nil {
		return errors.Wrap(users.ErrNotFound, err)
	}
	if _, err := gr.db.NamedExecContext(ctx, q, dbr); err != nil {
		return errors.Wrap(users.ErrConflict, err)
	}
	return nil
}

type dbGroup struct {
	ID          string        `db:"id"`
	Name        string        `db:"name"`
	OwnerID     uuid.NullUUID `db:"owner_id"`
	ParentID    uuid.NullUUID `db:"parent_id"`
	Description string        `db:"description"`
	Metadata    dbMetadata    `db:"metadata"`
}

type dbGroupPage struct {
	ID       uuid.NullUUID `db:"id"`
	OwnerID  uuid.NullUUID `db:"owner_id"`
	ParentID uuid.NullUUID `db:"parent_id"`
	Metadata dbMetadata    `db:"metadata"`
	Limit    uint64
	Offset   uint64
	Size     uint64
}

func toUUID(id string) (uuid.NullUUID, error) {
	var parentID uuid.NullUUID
	if err := parentID.Scan(id); err != nil {
		if id != "" {
			return parentID, err
		}
		if err := parentID.Scan(nil); err != nil {
			return parentID, err
		}
	}
	return parentID, nil
}

func toDBGroup(g users.Group) (dbGroup, error) {
	parentID := ""
	if g.ParentID != "" {
		parentID = g.ParentID
	}
	parent, err := toUUID(parentID)
	if err != nil {
		return dbGroup{}, err
	}
	owner, err := toUUID(g.OwnerID)
	if err != nil {
		return dbGroup{}, err
	}

	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		ParentID:    parent,
		OwnerID:     owner,
		Description: g.Description,
		Metadata:    g.Metadata,
	}, nil
}

func toDBGroupPage(ownerID, groupID string, offset, limit uint64, um users.Metadata) (dbGroupPage, error) {
	owner, err := toUUID(ownerID)
	if err != nil {
		return dbGroupPage{}, err
	}
	group, err := toUUID(groupID)
	if err != nil {
		return dbGroupPage{}, err
	}
	if err != nil {
		return dbGroupPage{}, err
	}
	return dbGroupPage{
		ID:       group,
		Metadata: dbMetadata(um),
		OwnerID:  owner,
		Offset:   offset,
		Limit:    limit,
	}, nil
}

func toGroup(dbu dbGroup) users.Group {
	return users.Group{
		ID:          dbu.ID,
		Name:        dbu.Name,
		ParentID:    dbu.ParentID.UUID.String(),
		OwnerID:     dbu.OwnerID.UUID.String(),
		Description: dbu.Description,
		Metadata:    dbu.Metadata,
	}
}

type dbGroupRelation struct {
	Group uuid.UUID `db:"group_id"`
	User  uuid.UUID `db:"user_id"`
}

func toDBGroupRelation(userID, groupID string) (dbGroupRelation, error) {
	group, err := uuid.FromString(groupID)
	if err != nil {
		return dbGroupRelation{}, err
	}
	user, err := uuid.FromString(userID)
	if err != nil {
		return dbGroupRelation{}, err
	}
	return dbGroupRelation{
		Group: group,
		User:  user,
	}, nil
}

func getGroupsMetadataQuery(um users.Metadata) ([]byte, string, error) {
	mq := ""
	mb := []byte("{}")
	if len(um) > 0 {
		mq = `groups.metadata @> :metadata`

		b, err := json.Marshal(um)
		if err != nil {
			return nil, "", err
		}
		mb = b
	}
	return mb, mq, nil
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}
