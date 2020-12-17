// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/internal/groups"
	"github.com/mainflux/mainflux/pkg/errors"
)

const maxLevel = 5

var (
	errDeleteGroupDB          = errors.New("delete group failed")
	errSelectDb               = errors.New("select group from db error")
	errConvertingStringToUUID = errors.New("error converting string")
)

var _ groups.Repository = (*groupRepository)(nil)

type groupRepository struct {
	db Database
}

// NewGroupRepo instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepo(db Database) groups.Repository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, g groups.Group) (groups.Group, error) {
	var id string
	q := `INSERT INTO thing_groups (name, description, id, owner_id, metadata, path, created_at, updated_at) 
		  VALUES (:name, :description, :id, :owner_id, :metadata, CAST(:id AS ltree), now(), now()) RETURNING id`
	if g.ParentID != "" {
		q = `INSERT INTO thing_groups (name, description, id, owner_id, parent_id, metadata, path, created_at, updated_at) 
			 SELECT :name, :description, :id, :owner_id, :parent_id, :metadata, text2ltree(ltree2text(tg.path) || '.' || CAST(:id AS TEXT)), now(), now() FROM thing_groups tg WHERE id = :parent_id RETURNING id`
	}

	dbu, err := toDBGroup(g)
	if err != nil {
		return groups.Group{}, err
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return groups.Group{}, errors.Wrap(groups.ErrMalformedEntity, err)
			case errDuplicate:
				return groups.Group{}, errors.Wrap(groups.ErrGroupConflict, err)
			}
		}

		return groups.Group{}, errors.Wrap(groups.ErrCreateGroup, err)
	}

	defer row.Close()
	row.Next()
	if err := row.Scan(&id); err != nil {
		return groups.Group{}, err
	}
	g.ID = id
	return g, nil
}

func (gr groupRepository) Update(ctx context.Context, g groups.Group) (groups.Group, error) {
	q := `UPDATE thing_groups SET description = :description, name = :name, metadata = :metadata, updated_at = now()  WHERE id = :id`

	dbu, err := toDBGroup(g)
	if err != nil {
		return groups.Group{}, errors.Wrap(errUpdateDB, err)
	}

	if _, err := gr.db.NamedExecContext(ctx, q, dbu); err != nil {
		return groups.Group{}, errors.Wrap(errUpdateDB, err)
	}

	return g, nil
}

func (gr groupRepository) Delete(ctx context.Context, groupID string) error {
	qd := `DELETE FROM thing_groups WHERE id = :id`
	group := groups.Group{
		ID: groupID,
	}
	dbg, err := toDBGroup(group)
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
		return errors.Wrap(groups.ErrDeleteGroup, err)
	}
	return nil
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (groups.Group, error) {
	dbu := dbGroup{
		ID: id,
	}

	q := `SELECT id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level FROM thing_groups WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return groups.Group{}, errors.Wrap(groups.ErrNotFound, err)

		}
		return groups.Group{}, errors.Wrap(errRetrieveDB, err)
	}

	return toGroup(dbu)
}

func (gr groupRepository) RetrieveAll(ctx context.Context, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	_, mq, err := getGroupsMetadataQuery("thing_groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT id, owner_id, parent_id, name, description, metadata, path, nlevel(path) as level, created_at, updated_at FROM thing_groups 
					  WHERE nlevel(path) <= :level %s ORDER BY path`, mq)
	cq := fmt.Sprintf("SELECT COUNT(*) FROM thing_groups WHERE nlevel(path) <= :level %s", mq)

	dbPage, err := toDBGroupPage("", "", "", "", level, gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	items, err := processRows(rows)
	if err != nil {
		return groups.GroupPage{}, err
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveAllParents(ctx context.Context, groupID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if groupID == "" {
		return groups.GroupPage{}, nil
	}

	_, mq, err := getGroupsMetadataQuery("thing_groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path, nlevel(g.path) as level, g.created_at, g.updated_at
					  FROM thing_groups parent, thing_groups g
					  WHERE parent.id = :parent_id AND g.path @> parent.path AND nlevel(parent.path) - nlevel(g.path) <= :level %s`, mq)

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM thing_groups parent, thing_groups g WHERE parent.id = :parent_id AND g.path @> parent.path %s`, mq)

	if level > maxLevel {
		level = maxLevel
	}

	dbPage, err := toDBGroupPage("", "", groupID, "", level, gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	items, err := processRows(rows)
	if err != nil {
		return groups.GroupPage{}, err
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveAllChildren(ctx context.Context, groupID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if groupID == "" {
		return groups.GroupPage{}, nil
	}
	_, mq, err := getGroupsMetadataQuery("thing_groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path, nlevel(g.path) as level, g.created_at, g.updated_at 
					  FROM thing_groups parent, thing_groups g
					  WHERE parent.id = :id AND g.path <@ parent.path AND nlevel(g.path) - nlevel(parent.path) <= :level %s`, mq)

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM thing_groups parent, thing_groups g WHERE parent.id = :id AND g.path <@ parent.path %s`, mq)

	if level > maxLevel {
		level = maxLevel
	}

	dbPage, err := toDBGroupPage("", groupID, "", "", level, gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	items, err := processRows(rows)
	if err != nil {
		return groups.GroupPage{}, err
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total: total,
		},
	}

	return page, nil
}

func (gr groupRepository) Members(ctx context.Context, groupID string, offset, limit uint64, gm groups.Metadata) (groups.MemberPage, error) {
	m, mq, err := getGroupsMetadataQuery("things_group", gm)
	if err != nil {
		return groups.MemberPage{}, errors.Wrap(errRetrieveDB, err)
	}

	q := fmt.Sprintf(`SELECT th.id, th.name, th.key, th.metadata FROM things th, thing_group_relations g
					  WHERE th.id = g.thing_id AND g.group_id = :group 
					  %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params := map[string]interface{}{
		"group":    groupID,
		"limit":    limit,
		"offset":   offset,
		"metadata": m,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return groups.MemberPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	var items []groups.Member
	for rows.Next() {
		dbTh := dbThing{}
		if err := rows.StructScan(&dbTh); err != nil {
			return groups.MemberPage{}, errors.Wrap(errSelectDb, err)
		}

		thing, err := toThing(dbTh)
		if err != nil {
			return groups.MemberPage{}, err
		}

		items = append(items, thing)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM things th, thing_group_relations g
					   WHERE th.id = g.thing_id AND g.group_id = :group  %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return groups.MemberPage{}, errors.Wrap(errSelectDb, err)
	}

	page := groups.MemberPage{
		Members: items,
		PageMetadata: groups.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (gr groupRepository) Memberships(ctx context.Context, userID string, offset, limit uint64, gm groups.Metadata) (groups.GroupPage, error) {
	m, mq, err := getGroupsMetadataQuery("thing_groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.parent_id, g.name, g.description, g.metadata 
					  FROM thing_group_relations gr, thing_groups g
					  WHERE gr.group_id = g.id and gr.thing_id = :userID 
		  			  %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params := map[string]interface{}{
		"userID":   userID,
		"limit":    limit,
		"offset":   offset,
		"metadata": m,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}
	defer rows.Close()

	var items []groups.Group
	for rows.Next() {
		dbgr := dbGroup{}
		if err := rows.StructScan(&dbgr); err != nil {
			return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
		}
		gr, err := toGroup(dbgr)
		if err != nil {
			return groups.GroupPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM thing_group_relations gr, thing_groups g
					   WHERE gr.group_id = g.id and gr.thing_id = :userID %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errSelectDb, err)
	}

	page := groups.GroupPage{
		Groups: items,
		PageMetadata: groups.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (gr groupRepository) Assign(ctx context.Context, thingID, groupID string) error {
	dbr, err := toDBGroupRelation(thingID, groupID)
	if err != nil {
		return errors.Wrap(groups.ErrAssignToGroup, err)
	}

	qIns := `INSERT INTO thing_group_relations (group_id, thing_id) VALUES (:group_id, :thing_id)`
	_, err = gr.db.NamedQueryContext(ctx, qIns, dbr)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return errors.Wrap(groups.ErrMalformedEntity, err)
			case errDuplicate:
				return errors.Wrap(groups.ErrGroupConflict, err)
			case errFK:
				return errors.Wrap(groups.ErrNotFound, err)
			}
		}
		return errors.Wrap(groups.ErrAssignToGroup, err)
	}

	return nil
}

func (gr groupRepository) Unassign(ctx context.Context, userID, groupID string) error {
	q := `DELETE FROM thing_group_relations WHERE thing_id = :thing_id AND group_id = :group_id`
	dbr, err := toDBGroupRelation(userID, groupID)
	if err != nil {
		return errors.Wrap(groups.ErrNotFound, err)
	}
	if _, err := gr.db.NamedExecContext(ctx, q, dbr); err != nil {
		return errors.Wrap(groups.ErrGroupConflict, err)
	}
	return nil
}

type dbGroup struct {
	ID          string         `db:"id"`
	ParentID    sql.NullString `db:"parent_id"`
	OwnerID     uuid.NullUUID  `db:"owner_id"`
	Name        string         `db:"name"`
	Description string         `db:"description"`
	Metadata    dbMetadata     `db:"metadata"`
	Level       int            `db:"level"`
	Path        string         `db:"path"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

type dbGroupPage struct {
	ID       string        `db:"id"`
	ParentID string        `db:"parent_id"`
	OwnerID  uuid.NullUUID `db:"owner_id"`
	Metadata dbMetadata    `db:"metadata"`
	Path     string        `db:"path"`
	Level    uint64        `db:"level"`
	Size     uint64        `db:"size"`
}

func toUUID(id string) (uuid.NullUUID, error) {
	var uid uuid.NullUUID
	if id == "" {
		return uuid.NullUUID{UUID: uuid.Nil, Valid: false}, nil
	}
	err := uid.Scan(id)
	return uid, err
}

func toString(id uuid.NullUUID) (string, error) {
	if id.Valid {
		return id.UUID.String(), nil
	}
	if id.UUID == uuid.Nil {
		return "", nil
	}
	return "", errConvertingStringToUUID
}

func toDBGroup(g groups.Group) (dbGroup, error) {
	ownerID, err := toUUID(g.OwnerID)
	if err != nil {
		return dbGroup{}, err
	}

	var parentID sql.NullString
	if g.ParentID != "" {
		parentID = sql.NullString{String: g.ParentID, Valid: true}
	}

	meta := dbMetadata(g.Metadata)

	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		ParentID:    parentID,
		OwnerID:     ownerID,
		Description: g.Description,
		Metadata:    meta,
		Path:        g.Path,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}, nil
}

func toDBGroupPage(ownerID, id, parentID, path string, level uint64, metadata groups.Metadata) (dbGroupPage, error) {
	owner, err := toUUID(ownerID)
	if err != nil {
		return dbGroupPage{}, err
	}

	if err != nil {
		return dbGroupPage{}, err
	}

	return dbGroupPage{
		Metadata: dbMetadata(metadata),
		ID:       id,
		OwnerID:  owner,
		Level:    level,
		Path:     path,
		ParentID: parentID,
	}, nil
}

func toGroup(dbu dbGroup) (groups.Group, error) {
	ownerID, err := toString(dbu.OwnerID)
	if err != nil {
		return groups.Group{}, err
	}

	return groups.Group{
		ID:          dbu.ID,
		Name:        dbu.Name,
		ParentID:    dbu.ParentID.String,
		OwnerID:     ownerID,
		Description: dbu.Description,
		Metadata:    groups.Metadata(dbu.Metadata),
		Level:       dbu.Level,
		Path:        dbu.Path,
		UpdatedAt:   dbu.UpdatedAt,
		CreatedAt:   dbu.CreatedAt,
	}, nil
}

type dbGroupRelation struct {
	GroupID string    `db:"group_id"`
	ThingID uuid.UUID `db:"thing_id"`
}

func toDBGroupRelation(thingID, groupID string) (dbGroupRelation, error) {
	thID, err := uuid.FromString(thingID)
	if err != nil {
		return dbGroupRelation{}, err
	}
	return dbGroupRelation{
		GroupID: groupID,
		ThingID: thID,
	}, nil
}

func getGroupsMetadataQuery(db string, m groups.Metadata) ([]byte, string, error) {
	mq := ""
	mb := []byte("{}")
	if len(m) > 0 {
		mq = db + `.metadata @> :metadata`
		if db == "" {
			mq = `metadata @> :metadata`
		}

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", err
		}
		mb = b
	}
	return mb, mq, nil
}

func processRows(rows *sqlx.Rows) ([]groups.Group, error) {
	var items []groups.Group
	for rows.Next() {
		dbgr := dbGroup{}
		if err := rows.StructScan(&dbgr); err != nil {
			return items, errors.Wrap(errSelectDb, err)
		}
		gr, err := toGroup(dbgr)
		if err != nil {
			continue
		}
		items = append(items, gr)
	}
	return items, nil
}
