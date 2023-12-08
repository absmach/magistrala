// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/invitations"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
)

type repository struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) invitations.Repository {
	return &repository{db: db}
}

func (repo *repository) Create(ctx context.Context, invitation invitations.Invitation) (err error) {
	q := `INSERT INTO invitations (invited_by, user_id, domain, token, relation, created_at, updated_at, confirmed_at)
		VALUES (:invited_by, :user_id, :domain, :token, :relation, :created_at, :updated_at, :confirmed_at)`

	if _, err = repo.db.NamedExecContext(ctx, q, invitation); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *repository) Retrieve(ctx context.Context, userID, domainID string) (invitations.Invitation, error) {
	q := `SELECT invited_by, user_id, domain, token, relation, created_at, updated_at, confirmed_at FROM invitations WHERE user_id = :user_id AND domain = :domain;`

	inv := invitations.Invitation{
		UserID: userID,
		Domain: domainID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, inv)
	if err != nil {
		return invitations.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var item invitations.Invitation
	if rows.Next() {
		if err = rows.StructScan(&item); err != nil {
			return invitations.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		return item, nil
	}

	return invitations.Invitation{}, repoerr.ErrNotFound
}

func (repo *repository) RetrieveAll(ctx context.Context, page invitations.Page) (invitations.InvitationPage, error) {
	query := pageQuery(page)

	q := fmt.Sprintf("SELECT invited_by, user_id, domain, relation, created_at, updated_at, confirmed_at FROM invitations %s LIMIT :limit OFFSET :offset;", query)

	rows, err := repo.db.NamedQueryContext(ctx, q, page)
	if err != nil {
		return invitations.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []invitations.Invitation
	for rows.Next() {
		var item invitations.Invitation
		if err = rows.StructScan(&item); err != nil {
			return invitations.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		items = append(items, item)
	}

	tq := fmt.Sprintf(`SELECT COUNT(*) FROM invitations %s`, query)

	total, err := postgres.Total(ctx, repo.db, tq, page)
	if err != nil {
		return invitations.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	invPage := invitations.InvitationPage{
		Total:       total,
		Offset:      page.Offset,
		Limit:       page.Limit,
		Invitations: items,
	}

	return invPage, nil
}

func (repo *repository) UpdateToken(ctx context.Context, invitation invitations.Invitation) (err error) {
	q := `UPDATE invitations SET token = :token, updated_at = :updated_at WHERE user_id = :user_id AND domain = :domain`

	result, err := repo.db.NamedExecContext(ctx, q, invitation)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) UpdateConfirmation(ctx context.Context, invitation invitations.Invitation) (err error) {
	q := `UPDATE invitations SET confirmed_at = :confirmed_at, updated_at = :updated_at WHERE user_id = :user_id AND domain = :domain`

	result, err := repo.db.NamedExecContext(ctx, q, invitation)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) Delete(ctx context.Context, userID, domain string) (err error) {
	q := `DELETE FROM invitations WHERE user_id = $1 AND domain = $2`

	result, err := repo.db.ExecContext(ctx, q, userID, domain)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func pageQuery(pm invitations.Page) string {
	var query []string
	var emq string
	if pm.Domain != "" {
		query = append(query, "domain = :domain")
	}
	if pm.UserID != "" {
		query = append(query, "user_id = :user_id")
	}
	if pm.InvitedBy != "" {
		query = append(query, "invited_by = :invited_by")
	}
	if pm.Relation != "" {
		query = append(query, "relation = :relation")
	}
	if pm.InvitedByOrUserID != "" {
		query = append(query, "(invited_by = :invited_by_or_user_id OR user_id = :invited_by_or_user_id)")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq
}
