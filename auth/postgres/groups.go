// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mainflux/mainflux/internal/groups"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"
)

const maxLevel = 5

var (
	errDeleteGroupDB          = errors.New("delete group failed")
	errSelectDb               = errors.New("select group from db error")
	errConvertingStringToUUID = errors.New("error converting string")
	errInvalidGroupType       = errors.New("invalid group type")
	errUpdateDB               = errors.New("failed to update db")
	errRetrieveDB             = errors.New("failed retrieving from db")

	errTruncation = "string_data_right_truncation"
	errFK         = "foreign_key_violation"
)

var _ groups.Repository = (*groupRepository)(nil)

type groupRepository struct {
	db    Database
	types map[string]dbGroupType
}

// NewGroupRepo instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepo(db Database) groups.Repository {
	q := `SELECT * FROM group_type`
	rows, err := db.QueryxContext(context.Background(), q)
	if err != nil {
		pqErr, _ := err.(*pq.Error)
		// If there is a problem with group type setup exit.
		panic(pqErr)
	}

	types := map[string]dbGroupType{}
	for rows.Next() {
		dbgrt := dbGroupType{}
		if err := rows.StructScan(&dbgrt); err != nil {
			panic(errors.Wrap(errSelectDb, err))
		}
		if _, ok := types[dbgrt.Name]; ok {
			panic(fmt.Sprintf("duplicated group type: %s", dbgrt.Name))
		}
		types[dbgrt.Name] = dbgrt
	}

	return &groupRepository{
		db:    db,
		types: types,
	}
}

func (gr groupRepository) Save(ctx context.Context, g groups.Group) (groups.Group, error) {
	var id string
	q := `INSERT INTO groups (name, description, id, owner_id, metadata, path, type, created_at, updated_at) 
		  VALUES (:name, :description, :id, :owner_id, :metadata, :name, :type, now(), now()) RETURNING id`
	if g.ParentID != "" {
		// For children groups type is inherited from the parent, this is done in trigger inherit_type_tr - init.go
		q = `INSERT INTO groups (name, description, id, owner_id, parent_id, metadata, path) 
			 SELECT :name, :description, :id, :owner_id, :parent_id, :metadata, text2ltree(ltree2text(tg.path) || '.' || :name) FROM groups tg WHERE id = :parent_id RETURNING id`
	}

	dbu, err := gr.toDBGroup(g)
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
	q := `UPDATE groups SET name = :name, description = :description, metadata = :metadata, updated_at = now()  WHERE id = :id`

	dbu, err := gr.toDBGroup(g)
	if err != nil {
		return groups.Group{}, errors.Wrap(errUpdateDB, err)
	}

	if _, err := gr.db.NamedExecContext(ctx, q, dbu); err != nil {
		return groups.Group{}, errors.Wrap(errUpdateDB, err)
	}

	return g, nil
}

func (gr groupRepository) Delete(ctx context.Context, groupID string) error {
	qd := `DELETE FROM groups WHERE id = :id`
	group := groups.Group{
		ID: groupID,
	}
	dbg, err := gr.toDBGroup(group)
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

	q := `SELECT id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level FROM groups WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return groups.Group{}, errors.Wrap(groups.ErrNotFound, err)

		}
		return groups.Group{}, errors.Wrap(errRetrieveDB, err)
	}

	return toGroup(dbu)
}

func (gr groupRepository) RetrieveAll(ctx context.Context, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	_, mq, err := getGroupsMetadataQuery("groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT id, owner_id, parent_id, name, description, metadata, path, nlevel(path) as level, created_at, updated_at FROM groups 
					  WHERE nlevel(path) <= :level %s ORDER BY path`, mq)
	cq := fmt.Sprintf("SELECT COUNT(*) FROM groups WHERE nlevel(path) <= :level %s", mq)

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

	_, mq, err := getGroupsMetadataQuery("groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path, nlevel(g.path) as level, g.created_at, g.updated_at
					  FROM groups parent, groups g
					  WHERE parent.id = :parent_id AND g.path @> parent.path AND nlevel(parent.path) - nlevel(g.path) <= :level %s`, mq)

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = :parent_id AND g.path @> parent.path %s`, mq)

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
	_, mq, err := getGroupsMetadataQuery("groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	q := fmt.Sprintf(`SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path, nlevel(g.path) as level, g.created_at, g.updated_at 
					  FROM groups parent, groups g
					  WHERE parent.id = :id AND g.path <@ parent.path AND nlevel(g.path) - nlevel(parent.path) <= :level %s`, mq)

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = :id AND g.path <@ parent.path %s`, mq)

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
	m, mq, err := getGroupsMetadataQuery("groups", gm)
	if err != nil {
		return groups.MemberPage{}, errors.Wrap(errRetrieveDB, err)
	}

	q := fmt.Sprintf(`SELECT gr.member_id FROM groups, group_relations gr
					  WHERE gr.group_id = :group AND gr.group_id = g.id
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
		member := dbMember{}
		if err := rows.StructScan(&member); err != nil {
			return groups.MemberPage{}, errors.Wrap(errSelectDb, err)
		}

		if err != nil {
			return groups.MemberPage{}, err
		}

		items = append(items, groups.Member(member.ID))
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM groups, group_relations g
					   WHERE g.group_id = groups.id AND g.group_id = :group  %s;`, mq)

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
	m, mq, err := getGroupsMetadataQuery("groups", gm)
	if err != nil {
		return groups.GroupPage{}, errors.Wrap(errRetrieveDB, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.parent_id, g.name, g.description, g.metadata 
					  FROM group_relations gr, groups g
					  WHERE gr.group_id = g.id and gr.member_id = :userID 
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

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM thing_group_relations gr, groups g
					   WHERE gr.group_id = g.id and gr.member_id = :userID %s;`, mq)

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

func (gr groupRepository) Assign(ctx context.Context, memberID, groupID string) error {
	dbr, err := toDBGroupRelation(memberID, groupID)
	if err != nil {
		return errors.Wrap(groups.ErrAssignToGroup, err)
	}

	qIns := `INSERT INTO group_relations (group_id, member_id) VALUES (:group_id, :member_id)`
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
	q := `DELETE FROM group_relations WHERE member_id = :member_id AND group_id = :group_id`
	dbr, err := toDBGroupRelation(userID, groupID)
	if err != nil {
		return errors.Wrap(groups.ErrNotFound, err)
	}
	if _, err := gr.db.NamedExecContext(ctx, q, dbr); err != nil {
		return errors.Wrap(groups.ErrGroupConflict, err)
	}
	return nil
}

type dbMember struct {
	ID string `db:"member_id"`
}
type dbGroupType struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type dbGroup struct {
	ID          string         `db:"id"`
	ParentID    sql.NullString `db:"parent_id"`
	OwnerID     uuid.NullUUID  `db:"owner_id"`
	Name        string         `db:"name"`
	Description string         `db:"description"`
	Metadata    dbMetadata     `db:"metadata"`
	Type        int            `db:"type"`
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

func (gr groupRepository) toDBGroup(g groups.Group) (dbGroup, error) {

	ownerID, err := toUUID(g.OwnerID)
	if err != nil {
		return dbGroup{}, err
	}

	var parentID sql.NullString
	if g.ParentID != "" {
		parentID = sql.NullString{String: g.ParentID, Valid: true}
	}

	meta := dbMetadata(g.Metadata)
	gType, ok := gr.types[g.Type]
	if !ok {
		return dbGroup{}, errInvalidGroupType
	}

	return dbGroup{
		ID:          g.ID,
		Name:        g.Name,
		ParentID:    parentID,
		OwnerID:     ownerID,
		Description: g.Description,
		Metadata:    meta,
		Type:        gType.ID,
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
	GroupID  uuid.UUID `db:"group_id"`
	MemberID uuid.UUID `db:"member_id"`
}

func toDBGroupRelation(memberID, groupID string) (dbGroupRelation, error) {
	grID, err := uuid.FromString(groupID)
	if err != nil {
		return dbGroupRelation{}, err
	}
	memID, err := uuid.FromString(memberID)
	if err != nil {
		return dbGroupRelation{}, err
	}
	return dbGroupRelation{
		GroupID:  grID,
		MemberID: memID,
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

// dbMetadata type for handling metadata properly in database/sql
type dbMetadata map[string]interface{}

// Scan - Implement the database/sql scanner interface
func (m *dbMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return users.ErrScanMetadata
	}

	if err := json.Unmarshal(b, m); err != nil {
		return err
	}

	return nil
}

// Value Implements valuer
func (m dbMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, err
}
