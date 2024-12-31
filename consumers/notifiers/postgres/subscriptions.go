// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/absmach/supermq/consumers/notifiers"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ notifiers.SubscriptionsRepository = (*subscriptionsRepo)(nil)

type subscriptionsRepo struct {
	db Database
}

// New instantiates a PostgreSQL implementation of Subscriptions repository.
func New(db Database) notifiers.SubscriptionsRepository {
	return &subscriptionsRepo{
		db: db,
	}
}

func (repo subscriptionsRepo) Save(ctx context.Context, sub notifiers.Subscription) (string, error) {
	q := `INSERT INTO subscriptions (id, owner_id, contact, topic) VALUES (:id, :owner_id, :contact, :topic) RETURNING id`

	dbSub := dbSubscription{
		ID:      sub.ID,
		OwnerID: sub.OwnerID,
		Contact: sub.Contact,
		Topic:   sub.Topic,
	}

	row, err := repo.db.NamedQueryContext(ctx, q, dbSub)
	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == pgerrcode.UniqueViolation {
			return "", errors.Wrap(repoerr.ErrConflict, err)
		}
		return "", errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	return sub.ID, nil
}

func (repo subscriptionsRepo) Retrieve(ctx context.Context, id string) (notifiers.Subscription, error) {
	q := `SELECT id, owner_id, contact, topic FROM subscriptions WHERE id = $1`
	sub := dbSubscription{}
	if err := repo.db.QueryRowxContext(ctx, q, id).StructScan(&sub); err != nil {
		if err == sql.ErrNoRows {
			return notifiers.Subscription{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
		return notifiers.Subscription{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return fromDBSub(sub), nil
}

func (repo subscriptionsRepo) RetrieveAll(ctx context.Context, pm notifiers.PageMetadata) (notifiers.Page, error) {
	q := `SELECT id, owner_id, contact, topic FROM subscriptions`
	args := make(map[string]interface{})
	if pm.Topic != "" {
		args["topic"] = pm.Topic
	}
	if pm.Contact != "" {
		args["contact"] = pm.Contact
	}
	var condition string
	if len(args) > 0 {
		var cond []string
		for k := range args {
			cond = append(cond, fmt.Sprintf("%s = :%s", k, k))
		}
		condition = fmt.Sprintf(" WHERE %s", strings.Join(cond, " AND "))
		q = fmt.Sprintf("%s%s", q, condition)
	}
	args["offset"] = pm.Offset
	q = fmt.Sprintf("%s OFFSET :offset", q)
	if pm.Limit > 0 {
		q = fmt.Sprintf("%s LIMIT :limit", q)
		args["limit"] = pm.Limit
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return notifiers.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var subs []notifiers.Subscription
	for rows.Next() {
		sub := dbSubscription{}
		if err := rows.StructScan(&sub); err != nil {
			return notifiers.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		subs = append(subs, fromDBSub(sub))
	}

	if len(subs) == 0 {
		return notifiers.Page{}, repoerr.ErrNotFound
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM subscriptions %s`, condition)
	total, err := total(ctx, repo.db, cq, args)
	if err != nil {
		return notifiers.Page{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	ret := notifiers.Page{
		PageMetadata:  pm,
		Total:         total,
		Subscriptions: subs,
	}

	return ret, nil
}

func (repo subscriptionsRepo) Remove(ctx context.Context, id string) error {
	q := `DELETE from subscriptions WHERE id = $1`

	if r := repo.db.QueryRowxContext(ctx, q, id); r.Err() != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, r.Err())
	}
	return nil
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var total uint
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}

type dbSubscription struct {
	ID      string `db:"id"`
	OwnerID string `db:"owner_id"`
	Contact string `db:"contact"`
	Topic   string `db:"topic"`
}

func fromDBSub(sub dbSubscription) notifiers.Subscription {
	return notifiers.Subscription{
		ID:      sub.ID,
		OwnerID: sub.OwnerID,
		Contact: sub.Contact,
		Topic:   sub.Topic,
	}
}
