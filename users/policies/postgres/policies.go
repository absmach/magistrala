// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
)

var _ policies.Repository = (*prepo)(nil)

type prepo struct {
	db postgres.Database
}

// NewRepository instantiates a PostgreSQL implementation of policy repository.
func NewRepository(db postgres.Database) policies.Repository {
	return &prepo{
		db: db,
	}
}

func (pr prepo) Save(ctx context.Context, policy policies.Policy) error {
	q := `INSERT INTO policies (owner_id, subject, object, actions, created_at)
		VALUES (:owner_id, :subject, :object, :actions, :created_at)
		ON CONFLICT (subject, object) DO UPDATE SET actions = :actions,
		updated_at = :updated_at, updated_by = :updated_by`

	dbp, err := toDBPolicy(policy)
	if err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	row, err := pr.db.NamedQueryContext(ctx, q, dbp)
	if err != nil {
		return postgres.HandleError(err, errors.ErrCreateEntity)
	}

	defer row.Close()

	return nil
}

func (pr prepo) CheckAdmin(ctx context.Context, id string) error {
	q := fmt.Sprintf(`SELECT id FROM clients WHERE id = '%s' AND role = '%d';`, id, clients.AdminRole)

	var clientID string
	if err := pr.db.QueryRowxContext(ctx, q).Scan(&clientID); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if clientID == "" {
		return errors.ErrAuthorization
	}

	return nil
}

func (pr prepo) EvaluateUserAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	// Evaluates if two clients are connected to the same group and the subject has the specified action
	// or subject is the owner of the object
	query := fmt.Sprintf(`(SELECT subject, object, actions FROM policies p 
	WHERE p.subject = :subject AND '%s' = ANY(p.actions) AND object IN (SELECT object FROM policies WHERE subject = :object))
	UNION
	(SELECT owner_id as subject, id as object, '{}' as actions FROM clients c WHERE c.owner_id = :subject AND c.id = :object) LIMIT 1;`, ar.Action)

	return pr.evaluate(ctx, query, ar)
}

func (pr prepo) EvaluateGroupAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	// Evaluates if client is a member to that group and has the specified action or is the owner of the group
	query := fmt.Sprintf(`(SELECT subject, object, actions FROM policies p 
	WHERE p.subject = :subject AND p.object = :object AND '%s' = ANY(p.actions))
	UNION
	(SELECT owner_id as subject, id as object, '{}' as actions FROM groups g WHERE g.owner_id = :subject AND g.id = :object)`, ar.Action)

	return pr.evaluate(ctx, query, ar)
}

func (pr prepo) evaluate(ctx context.Context, query string, aReq policies.AccessRequest) (policies.Policy, error) {
	p := policies.Policy{
		Subject: aReq.Subject,
		Object:  aReq.Object,
		Actions: []string{aReq.Action},
	}
	dbp, err := toDBPolicy(p)
	if err != nil {
		return policies.Policy{}, errors.Wrap(errors.ErrAuthorization, err)
	}
	row, err := pr.db.NamedQueryContext(ctx, query, dbp)
	if err != nil {
		return policies.Policy{}, postgres.HandleError(err, errors.ErrAuthorization)
	}

	defer row.Close()

	if ok := row.Next(); !ok {
		return policies.Policy{}, errors.Wrap(errors.ErrAuthorization, row.Err())
	}
	dbp = dbPolicy{}
	if err := row.StructScan(&dbp); err != nil {
		return policies.Policy{}, err
	}
	return toPolicy(dbp)
}

func (pr prepo) Update(ctx context.Context, policy policies.Policy) error {
	q := `UPDATE policies SET actions = :actions, updated_at = :updated_at, updated_by = :updated_by
		WHERE subject = :subject AND object = :object`

	dbu, err := toDBPolicy(policy)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if _, err := pr.db.NamedExecContext(ctx, q, dbu); err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (pr prepo) RetrieveAll(ctx context.Context, pm policies.Page) (policies.PolicyPage, error) {
	var query []string
	var emq string

	if pm.OwnerID != "" {
		query = append(query, "owner_id = :owner_id")
	}
	if pm.Subject != "" {
		query = append(query, "subject = :subject")
	}
	if pm.Object != "" {
		query = append(query, "object = :object")
	}
	if pm.Action != "" {
		query = append(query, ":action = ANY (actions)")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT owner_id, subject, object, actions, created_at, updated_at, updated_by
		FROM policies %s ORDER BY updated_at LIMIT :limit OFFSET :offset;`, emq)

	dbPage, err := toDBPoliciesPage(pm)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	rows, err := pr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []policies.Policy
	for rows.Next() {
		dbp := dbPolicy{}
		if err := rows.StructScan(&dbp); err != nil {
			return policies.PolicyPage{}, errors.Wrap(errors.ErrViewEntity, err)
		}

		policy, err := toPolicy(dbp)
		if err != nil {
			return policies.PolicyPage{}, err
		}

		items = append(items, policy)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM policies %s;`, emq)

	total, err := postgres.Total(ctx, pr.db, cq, dbPage)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(errors.ErrViewEntity, err)
	}

	page := policies.PolicyPage{
		Policies: items,
		Page: policies.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (pr prepo) Delete(ctx context.Context, p policies.Policy) error {
	dbp := dbPolicy{
		Subject: p.Subject,
		Object:  p.Object,
	}
	q := `DELETE FROM policies WHERE subject = :subject AND object = :object`
	if _, err := pr.db.NamedExecContext(ctx, q, dbp); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

type dbPolicy struct {
	OwnerID   string           `db:"owner_id"`
	Subject   string           `db:"subject"`
	Object    string           `db:"object"`
	Actions   pgtype.TextArray `db:"actions"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedAt sql.NullTime     `db:"updated_at,omitempty"`
	UpdatedBy *string          `db:"updated_by,omitempty"`
}

func toDBPolicy(p policies.Policy) (dbPolicy, error) {
	var actions pgtype.TextArray
	if err := actions.Set(p.Actions); err != nil {
		return dbPolicy{}, err
	}
	var updatedAt sql.NullTime
	if !p.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{Time: p.UpdatedAt, Valid: true}
	}
	var updatedBy *string
	if p.UpdatedBy != "" {
		updatedBy = &p.UpdatedBy
	}
	return dbPolicy{
		OwnerID:   p.OwnerID,
		Subject:   p.Subject,
		Object:    p.Object,
		Actions:   actions,
		CreatedAt: p.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
	}, nil
}

func toPolicy(dbp dbPolicy) (policies.Policy, error) {
	var actions []string
	for _, e := range dbp.Actions.Elements {
		actions = append(actions, e.String)
	}
	var updatedAt time.Time
	if dbp.UpdatedAt.Valid {
		updatedAt = dbp.UpdatedAt.Time
	}
	var updatedBy string
	if dbp.UpdatedBy != nil {
		updatedBy = *dbp.UpdatedBy
	}
	return policies.Policy{
		OwnerID:   dbp.OwnerID,
		Subject:   dbp.Subject,
		Object:    dbp.Object,
		Actions:   actions,
		CreatedAt: dbp.CreatedAt,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
	}, nil
}

func toDBPoliciesPage(pm policies.Page) (dbPoliciesPage, error) {
	return dbPoliciesPage{
		Total:   pm.Total,
		Offset:  pm.Offset,
		Limit:   pm.Limit,
		OwnerID: pm.OwnerID,
		Subject: pm.Subject,
		Object:  pm.Object,
		Action:  pm.Action,
	}, nil
}

type dbPoliciesPage struct {
	Total   uint64 `db:"total"`
	Limit   uint64 `db:"limit"`
	Offset  uint64 `db:"offset"`
	OwnerID string `db:"owner_id"`
	Subject string `db:"subject"`
	Object  string `db:"object"`
	Action  string `db:"action"`
}
