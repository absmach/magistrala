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
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errStringToUUID        = errors.New("error converting string to uuid")
	errGetTotal            = errors.New("failed to get total number of groups")
	errCreateMetadataQuery = errors.New("failed to create query for metadata")
	groupIDFkeyy           = "group_relations_group_id_fkey"
)

var _ auth.GroupRepository = (*groupRepository)(nil)

type groupRepository struct {
	db Database
}

// NewGroupRepo instantiates a PostgreSQL implementation of group
// repository.
func NewGroupRepo(db Database) auth.GroupRepository {
	return &groupRepository{
		db: db,
	}
}

func (gr groupRepository) Save(ctx context.Context, g auth.Group) (auth.Group, error) {
	// For root group path is initialized with id
	q := `INSERT INTO groups (name, description, id, path, owner_id, metadata, created_at, updated_at)
		  VALUES (:name, :description, :id, :id, :owner_id, :metadata, :created_at, :updated_at)
		  RETURNING id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level, created_at, updated_at`
	if g.ParentID != "" {
		// Path is constructed in insert_group_tr - init.go
		q = `INSERT INTO groups (name, description, id, owner_id, parent_id, metadata, created_at, updated_at)
			 VALUES ( :name, :description, :id, :owner_id, :parent_id, :metadata, :created_at, :updated_at)
			 RETURNING id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level, created_at, updated_at`
	}

	dbg, err := toDBGroup(g)
	if err != nil {
		return auth.Group{}, err
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbg)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.ForeignKeyViolation:
				return auth.Group{}, errors.Wrap(errors.ErrCreateEntity, err)
			case pgerrcode.UniqueViolation:
				return auth.Group{}, errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return auth.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return auth.Group{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbg = dbGroup{}
	if err := row.StructScan(&dbg); err != nil {
		return auth.Group{}, err
	}

	return toGroup(dbg)
}

func (gr groupRepository) Update(ctx context.Context, g auth.Group) (auth.Group, error) {
	q := `UPDATE groups SET name = :name, description = :description, metadata = :metadata, updated_at = :updated_at WHERE id = :id
		  RETURNING id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level, created_at, updated_at`

	dbu, err := toDBGroup(g)
	if err != nil {
		return auth.Group{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	row, err := gr.db.NamedQueryContext(ctx, q, dbu)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return auth.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.UniqueViolation:
				return auth.Group{}, errors.Wrap(errors.ErrConflict, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return auth.Group{}, errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}
		return auth.Group{}, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	defer row.Close()
	row.Next()
	dbu = dbGroup{}
	if err := row.StructScan(&dbu); err != nil {
		return g, errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return toGroup(dbu)
}

func (gr groupRepository) Delete(ctx context.Context, groupID string) error {
	qd := `DELETE FROM groups WHERE id = :id`
	group := auth.Group{
		ID: groupID,
	}
	dbg, err := toDBGroup(group)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	res, err := gr.db.NamedExecContext(ctx, qd, dbg)
	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pqErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.ForeignKeyViolation:
				switch pqErr.ConstraintName {
				case groupIDFkeyy:
					return errors.Wrap(auth.ErrGroupNotEmpty, err)
				}
				return errors.Wrap(errors.ErrConflict, err)
			}
		}
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if cnt != 1 {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (gr groupRepository) RetrieveByID(ctx context.Context, id string) (auth.Group, error) {
	dbu := dbGroup{
		ID: id,
	}
	q := `SELECT id, name, owner_id, parent_id, description, metadata, path, nlevel(path) as level, created_at, updated_at FROM groups WHERE id = $1`
	if err := gr.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		if err == sql.ErrNoRows {
			return auth.Group{}, errors.Wrap(errors.ErrNotFound, err)

		}
		return auth.Group{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	return toGroup(dbu)
}

func (gr groupRepository) RetrieveAll(ctx context.Context, pm auth.PageMetadata) (auth.GroupPage, error) {
	_, metaQuery, err := getGroupsMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	var mq string
	if metaQuery != "" {
		mq = fmt.Sprintf(" AND %s", metaQuery)
	}

	q := fmt.Sprintf(`SELECT id, owner_id, parent_id, name, description, metadata, path, nlevel(path) as level, created_at, updated_at FROM groups
					  WHERE nlevel(path) <= :level %s ORDER BY path`, mq)

	dbPage, err := toDBGroupPage("", "", pm)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}
	defer rows.Close()

	items, err := gr.processRows(rows)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	cq := "SELECT COUNT(*) FROM groups"
	if metaQuery != "" {
		cq = fmt.Sprintf(" %s WHERE %s", cq, metaQuery)
	}

	total, err := total(ctx, gr.db, cq, dbPage)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveAll, err)
	}

	page := auth.GroupPage{
		Groups: items,
		PageMetadata: auth.PageMetadata{
			Total: total,
			Size:  uint64(len(items)),
		},
	}

	return page, nil
}

func (gr groupRepository) RetrieveAllParents(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	q := `SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path, nlevel(g.path) as level, g.created_at, g.updated_at
		  FROM groups parent, groups g
	      WHERE parent.id = :id AND g.path @> parent.path AND nlevel(parent.path) - nlevel(g.path) <= :level`
	cq := `SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = :id AND g.path @> parent.path`

	gp, err := gr.retrieve(ctx, groupID, q, cq, pm)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveParents, err)
	}
	return gp, nil
}

func (gr groupRepository) RetrieveAllChildren(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	q := `SELECT g.id, g.name, g.owner_id, g.parent_id, g.description, g.metadata, g.path,  nlevel(g.path) as level, g.created_at, g.updated_at
	FROM groups parent, groups g
	WHERE parent.id = :id AND g.path <@ parent.path AND nlevel(g.path) - nlevel(parent.path) < :level`

	cq := `SELECT COUNT(*) FROM groups parent, groups g WHERE parent.id = :id AND g.path <@ parent.path `
	gp, err := gr.retrieve(ctx, groupID, q, cq, pm)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveChildren, err)
	}
	return gp, nil
}

func (gr groupRepository) retrieve(ctx context.Context, groupID, retQuery, cntQuery string, pm auth.PageMetadata) (auth.GroupPage, error) {
	if groupID == "" {
		return auth.GroupPage{}, nil
	}
	_, mq, err := getGroupsMetadataQuery("g", pm.Metadata)
	if err != nil {
		return auth.GroupPage{}, err
	}
	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}

	retQuery = fmt.Sprintf(`%s %s`, retQuery, mq)
	cntQuery = fmt.Sprintf(`%s %s`, cntQuery, mq)

	dbPage, err := toDBGroupPage(groupID, "", pm)
	if err != nil {
		return auth.GroupPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, retQuery, dbPage)
	if err != nil {
		return auth.GroupPage{}, err
	}
	defer rows.Close()

	items, err := gr.processRows(rows)
	if err != nil {
		return auth.GroupPage{}, err
	}

	total, err := total(ctx, gr.db, cntQuery, dbPage)
	if err != nil {
		return auth.GroupPage{}, err
	}

	page := auth.GroupPage{
		Groups: items,
		PageMetadata: auth.PageMetadata{
			Level: pm.Level,
			Total: total,
			Size:  uint64(len(items)),
		},
	}

	return page, nil

}

func (gr groupRepository) Members(ctx context.Context, groupID, groupType string, pm auth.PageMetadata) (auth.MemberPage, error) {
	_, mq, err := getGroupsMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return auth.MemberPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	q := fmt.Sprintf(`SELECT gr.member_id, gr.group_id, gr.type, gr.created_at, gr.updated_at FROM group_relations gr
					  WHERE gr.group_id = :group_id AND gr.type = :type %s`, mq)

	if groupType == "" {
		q = fmt.Sprintf(`SELECT gr.member_id, gr.group_id, gr.type, gr.created_at, gr.updated_at FROM group_relations gr
		                 WHERE gr.group_id = :group_id %s`, mq)
	}

	params, err := toDBMemberPage("", groupID, groupType, pm)
	if err != nil {
		return auth.MemberPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.MemberPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}
	defer rows.Close()

	var items []auth.Member
	for rows.Next() {
		member := dbMember{}
		if err := rows.StructScan(&member); err != nil {
			return auth.MemberPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
		}

		if err != nil {
			return auth.MemberPage{}, err
		}

		items = append(items, auth.Member{ID: member.MemberID, Type: member.Type})
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM groups g, group_relations gr
					   WHERE gr.group_id = :group_id AND gr.group_id = g.id AND gr.type = :type %s;`, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.MemberPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembers, err)
	}

	page := auth.MemberPage{
		Members: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Size:   uint64(len(items)),
		},
	}

	return page, nil
}

func (gr groupRepository) Memberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	_, mq, err := getGroupsMetadataQuery("groups", pm.Metadata)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	if mq != "" {
		mq = fmt.Sprintf("AND %s", mq)
	}
	q := fmt.Sprintf(`SELECT g.id, g.owner_id, g.parent_id, g.name, g.description, g.metadata
					  FROM group_relations gr, groups g
					  WHERE gr.group_id = g.id and gr.member_id = :member_id
		  			  %s ORDER BY id LIMIT :limit OFFSET :offset;`, mq)

	params, err := toDBMemberPage(memberID, "", "", pm)
	if err != nil {
		return auth.GroupPage{}, err
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}
	defer rows.Close()

	var items []auth.Group
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
		}
		gr, err := toGroup(dbg)
		if err != nil {
			return auth.GroupPage{}, err
		}
		items = append(items, gr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM group_relations gr, groups g
					   WHERE gr.group_id = g.id and gr.member_id = :member_id %s `, mq)

	total, err := total(ctx, gr.db, cq, params)
	if err != nil {
		return auth.GroupPage{}, errors.Wrap(auth.ErrFailedToRetrieveMembership, err)
	}

	page := auth.GroupPage{
		Groups: items,
		PageMetadata: auth.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Size:   uint64(len(items)),
		},
	}

	return page, nil
}

func (gr groupRepository) Assign(ctx context.Context, groupID, groupType string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToGroup, err)
	}

	qIns := `INSERT INTO group_relations (group_id, member_id, type, created_at, updated_at)
			 VALUES(:group_id, :member_id, :type, :created_at, :updated_at)`

	for _, id := range ids {
		dbg, err := toDBGroupRelation(id, groupID, groupType)
		if err != nil {
			return errors.Wrap(auth.ErrAssignToGroup, err)
		}
		created := time.Now()
		dbg.CreatedAt = created
		dbg.UpdatedAt = created

		if _, err := tx.NamedExecContext(ctx, qIns, dbg); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(auth.ErrMemberAlreadyAssigned, errors.New(pgErr.Detail))
				}
			}

			return errors.Wrap(auth.ErrAssignToGroup, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignToGroup, err)
	}

	return nil
}

func (gr groupRepository) Unassign(ctx context.Context, groupID string, ids ...string) error {
	tx, err := gr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(auth.ErrAssignToGroup, err)
	}

	qDel := `DELETE from group_relations WHERE group_id = :group_id AND member_id = :member_id`

	for _, id := range ids {
		dbg, err := toDBGroupRelation(id, groupID, "")
		if err != nil {
			return errors.Wrap(auth.ErrAssignToGroup, err)
		}

		if _, err := tx.NamedExecContext(ctx, qDel, dbg); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, err)
				}
			}

			return errors.Wrap(auth.ErrAssignToGroup, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(auth.ErrAssignToGroup, err)
	}

	return nil
}

type dbMember struct {
	MemberID  string    `db:"member_id"`
	GroupID   string    `db:"group_id"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
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
	Total    uint64        `db:"total"`
	Limit    uint64        `db:"limit"`
	Offset   uint64        `db:"offset"`
}

type dbMemberPage struct {
	GroupID  string     `db:"group_id"`
	MemberID string     `db:"member_id"`
	Type     string     `db:"type"`
	Metadata dbMetadata `db:"metadata"`
	Limit    uint64     `db:"limit"`
	Offset   uint64     `db:"offset"`
	Size     uint64
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
	return "", errStringToUUID
}

func toDBGroup(g auth.Group) (dbGroup, error) {
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

func toDBGroupPage(id, path string, pm auth.PageMetadata) (dbGroupPage, error) {
	level := auth.MaxLevel
	if pm.Level < auth.MaxLevel {
		level = pm.Level
	}
	return dbGroupPage{
		Metadata: dbMetadata(pm.Metadata),
		ID:       id,
		Path:     path,
		Level:    level,
		Total:    pm.Total,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
	}, nil
}

func toDBMemberPage(memberID, groupID, groupType string, pm auth.PageMetadata) (dbMemberPage, error) {
	return dbMemberPage{
		GroupID:  groupID,
		MemberID: memberID,
		Type:     groupType,
		Metadata: dbMetadata(pm.Metadata),
		Offset:   pm.Offset,
		Limit:    pm.Limit,
	}, nil
}

func toGroup(dbu dbGroup) (auth.Group, error) {
	ownerID, err := toString(dbu.OwnerID)
	if err != nil {
		return auth.Group{}, err
	}

	return auth.Group{
		ID:          dbu.ID,
		Name:        dbu.Name,
		ParentID:    dbu.ParentID.String,
		OwnerID:     ownerID,
		Description: dbu.Description,
		Metadata:    auth.GroupMetadata(dbu.Metadata),
		Level:       dbu.Level,
		Path:        dbu.Path,
		UpdatedAt:   dbu.UpdatedAt,
		CreatedAt:   dbu.CreatedAt,
	}, nil
}

type dbGroupRelation struct {
	GroupID   sql.NullString `db:"group_id"`
	MemberID  sql.NullString `db:"member_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
	Type      string         `db:"type"`
}

func toDBGroupRelation(memberID, groupID, groupType string) (dbGroupRelation, error) {
	var grID sql.NullString
	if groupID != "" {
		grID = sql.NullString{String: groupID, Valid: true}
	}

	var mID sql.NullString
	if memberID != "" {
		mID = sql.NullString{String: memberID, Valid: true}
	}

	return dbGroupRelation{
		GroupID:  grID,
		MemberID: mID,
		Type:     groupType,
	}, nil
}

func getGroupsMetadataQuery(db string, m auth.GroupMetadata) (mb []byte, mq string, err error) {
	if len(m) > 0 {
		mq = `metadata @> :metadata`
		if db != "" {
			mq = db + "." + mq
		}

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", errors.Wrap(err, errCreateMetadataQuery)
		}
		mb = b
	}
	return mb, mq, nil
}

func (gr groupRepository) processRows(rows *sqlx.Rows) ([]auth.Group, error) {
	var items []auth.Group
	for rows.Next() {
		dbg := dbGroup{}
		if err := rows.StructScan(&dbg); err != nil {
			return items, err
		}
		group, err := toGroup(dbg)
		if err != nil {
			return items, err
		}
		items = append(items, group)
	}
	return items, nil
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, errors.Wrap(errGetTotal, err)
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, errors.Wrap(errGetTotal, err)
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
		return errors.ErrScanMetadata
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
