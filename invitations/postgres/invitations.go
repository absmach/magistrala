// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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
	q := `INSERT INTO invitations (invited_by, user_id, domain_id, token, relation, created_at)
		VALUES (:invited_by, :user_id, :domain_id, :token, :relation, :created_at)`

	dbInv := toDBInvitation(invitation)
	if _, err = repo.db.NamedExecContext(ctx, q, dbInv); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *repository) Retrieve(ctx context.Context, userID, domainID string) (invitations.Invitation, error) {
	q := `SELECT invited_by, user_id, domain_id, token, relation, created_at, updated_at, confirmed_at FROM invitations WHERE user_id = :user_id AND domain_id = :domain_id;`

	dbinv := dbInvitation{
		UserID:   userID,
		DomainID: domainID,
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbinv)
	if err != nil {
		return invitations.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbinv = dbInvitation{}
	if rows.Next() {
		if err = rows.StructScan(&dbinv); err != nil {
			return invitations.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		return toInvitation(dbinv), nil
	}

	return invitations.Invitation{}, repoerr.ErrNotFound
}

func (repo *repository) RetrieveAll(ctx context.Context, page invitations.Page) (invitations.InvitationPage, error) {
	query := pageQuery(page)

	q := fmt.Sprintf("SELECT invited_by, user_id, domain_id, relation, created_at, updated_at, confirmed_at FROM invitations %s LIMIT :limit OFFSET :offset;", query)

	rows, err := repo.db.NamedQueryContext(ctx, q, page)
	if err != nil {
		return invitations.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []invitations.Invitation
	for rows.Next() {
		var dbinv dbInvitation
		if err = rows.StructScan(&dbinv); err != nil {
			return invitations.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		items = append(items, toInvitation(dbinv))
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
	q := `UPDATE invitations SET token = :token, updated_at = :updated_at WHERE user_id = :user_id AND domain_id = :domain_id`

	dbinv := toDBInvitation(invitation)
	result, err := repo.db.NamedExecContext(ctx, q, dbinv)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) UpdateConfirmation(ctx context.Context, invitation invitations.Invitation) (err error) {
	q := `UPDATE invitations SET confirmed_at = :confirmed_at, updated_at = :updated_at WHERE user_id = :user_id AND domain_id = :domain_id`

	dbinv := toDBInvitation(invitation)
	result, err := repo.db.NamedExecContext(ctx, q, dbinv)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) Delete(ctx context.Context, userID, domain string) (err error) {
	q := `DELETE FROM invitations WHERE user_id = $1 AND domain_id = $2`

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
	if pm.DomainID != "" {
		query = append(query, "domain_id = :domain_id")
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
	if pm.State == invitations.Accepted {
		query = append(query, "confirmed_at IS NOT NULL")
	}
	if pm.State == invitations.Pending {
		query = append(query, "confirmed_at IS NULL")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq
}

type dbInvitation struct {
	InvitedBy   string       `db:"invited_by"`
	UserID      string       `db:"user_id"`
	DomainID    string       `db:"domain_id"`
	Token       string       `db:"token,omitempty"`
	Relation    string       `db:"relation"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   sql.NullTime `db:"updated_at,omitempty"`
	ConfirmedAt sql.NullTime `db:"confirmed_at,omitempty"`
}

func toDBInvitation(inv invitations.Invitation) dbInvitation {
	var updatedAt sql.NullTime
	if inv.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: inv.UpdatedAt, Valid: true}
	}
	var confirmedAt sql.NullTime
	if inv.ConfirmedAt != (time.Time{}) {
		confirmedAt = sql.NullTime{Time: inv.ConfirmedAt, Valid: true}
	}

	return dbInvitation{
		InvitedBy:   inv.InvitedBy,
		UserID:      inv.UserID,
		DomainID:    inv.DomainID,
		Token:       inv.Token,
		Relation:    inv.Relation,
		CreatedAt:   inv.CreatedAt,
		UpdatedAt:   updatedAt,
		ConfirmedAt: confirmedAt,
	}
}

func toInvitation(dbinv dbInvitation) invitations.Invitation {
	var updatedAt time.Time
	if dbinv.UpdatedAt.Valid {
		updatedAt = dbinv.UpdatedAt.Time
	}
	var confirmedAt time.Time
	if dbinv.ConfirmedAt.Valid {
		confirmedAt = dbinv.ConfirmedAt.Time
	}

	return invitations.Invitation{
		InvitedBy:   dbinv.InvitedBy,
		UserID:      dbinv.UserID,
		DomainID:    dbinv.DomainID,
		Token:       dbinv.Token,
		Relation:    dbinv.Relation,
		CreatedAt:   dbinv.CreatedAt,
		UpdatedAt:   updatedAt,
		ConfirmedAt: confirmedAt,
	}
}
