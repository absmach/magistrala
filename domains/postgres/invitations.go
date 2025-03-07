// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/supermq/domains"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

func (repo domainRepo) SaveInvitation(ctx context.Context, invitation domains.Invitation) (err error) {
	q := `INSERT INTO invitations (invited_by, invitee_user_id, domain_id, role_id, created_at)
		VALUES (:invited_by, :invitee_user_id, :domain_id, :role_id, :created_at)`

	dbInv := toDBInvitation(invitation)
	if _, err = repo.db.NamedExecContext(ctx, q, dbInv); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo domainRepo) RetrieveInvitation(ctx context.Context, inviteeUserID, domainID string) (domains.Invitation, error) {
	q := `SELECT invited_by, invitee_user_id, domain_id, role_id, created_at, updated_at, confirmed_at, rejected_at FROM invitations WHERE invitee_user_id = :invitee_user_id AND domain_id = :domain_id;`

	dbinv := dbInvitation{
		InviteeUserID: inviteeUserID,
		DomainID:      domainID,
	}
	rows, err := repo.db.NamedQueryContext(ctx, q, dbinv)
	if err != nil {
		return domains.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbinv = dbInvitation{}
	if rows.Next() {
		if err = rows.StructScan(&dbinv); err != nil {
			return domains.Invitation{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		return toInvitation(dbinv), nil
	}

	return domains.Invitation{}, repoerr.ErrNotFound
}

func (repo domainRepo) RetrieveAllInvitations(ctx context.Context, pm domains.InvitationPageMeta) (domains.InvitationPage, error) {
	query := pageQuery(pm)

	q := fmt.Sprintf(`
		SELECT
			i.invited_by,
			i.invitee_user_id,
			i.domain_id,
			d."name"  AS domain_name,
			i.role_id,
			dr."name" AS role_name,
			i.created_at,
			i.updated_at,
			i.confirmed_at,
			i.rejected_at
		FROM
			invitations i
		LEFT JOIN domains d ON
			i.domain_id = d.id
		LEFT JOIN domains_roles dr ON
			dr.id = i.role_id
 		%s
		LIMIT :limit OFFSET :offset;
		`, query)

	rows, err := repo.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return domains.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []domains.Invitation
	for rows.Next() {
		var dbinv dbInvitation
		if err = rows.StructScan(&dbinv); err != nil {
			return domains.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		items = append(items, toInvitation(dbinv))
	}

	tq := fmt.Sprintf(`
		SELECT
			COUNT(*)
		FROM
			invitations i
		LEFT JOIN domains d ON
			i.domain_id = d.id
		LEFT JOIN domains_roles dr ON
			dr.id = i.role_id   %s
		`, query)

	total, err := postgres.Total(ctx, repo.db, tq, pm)
	if err != nil {
		return domains.InvitationPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	invPage := domains.InvitationPage{
		Total:       total,
		Offset:      pm.Offset,
		Limit:       pm.Limit,
		Invitations: items,
	}

	return invPage, nil
}

func (repo domainRepo) UpdateConfirmation(ctx context.Context, invitation domains.Invitation) (err error) {
	q := `UPDATE invitations SET confirmed_at = :confirmed_at, updated_at = :updated_at WHERE invitee_user_id = :invitee_user_id AND domain_id = :domain_id`

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

func (repo domainRepo) UpdateRejection(ctx context.Context, invitation domains.Invitation) (err error) {
	q := `UPDATE invitations SET rejected_at = :rejected_at, updated_at = :updated_at WHERE invitee_user_id = :invitee_user_id AND domain_id = :domain_id`

	dbInv := toDBInvitation(invitation)
	result, err := repo.db.NamedExecContext(ctx, q, dbInv)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo domainRepo) DeleteInvitation(ctx context.Context, inviteeUserID, domain string) (err error) {
	q := `DELETE FROM invitations WHERE invitee_user_id = $1 AND domain_id = $2`

	result, err := repo.db.ExecContext(ctx, q, inviteeUserID, domain)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func pageQuery(pm domains.InvitationPageMeta) string {
	var query []string
	var emq string
	if pm.DomainID != "" {
		query = append(query, "i.domain_id = :domain_id")
	}
	if pm.InviteeUserID != "" {
		query = append(query, "i.invitee_user_id = :invitee_user_id")
	}
	if pm.InvitedBy != "" {
		query = append(query, "i.invited_by = :invited_by")
	}
	if pm.RoleID != "" {
		query = append(query, "i.role_id = :role_id")
	}
	if pm.InvitedByOrUserID != "" {
		query = append(query, "(i.invited_by = :invited_by_or_user_id OR i.invitee_user_id = :invited_by_or_user_id)")
	}
	if pm.State == domains.Accepted {
		query = append(query, "i.confirmed_at IS NOT NULL")
	}
	if pm.State == domains.Pending {
		query = append(query, "i.confirmed_at IS NULL AND rejected_at IS NULL")
	}
	if pm.State == domains.Rejected {
		query = append(query, "i.rejected_at IS NOT NULL")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq
}

type dbInvitation struct {
	InvitedBy     string         `db:"invited_by"`
	InviteeUserID string         `db:"invitee_user_id"`
	DomainID      string         `db:"domain_id"`
	DomainName    sql.NullString `db:"domain_name,omitempty"`
	RoleID        string         `db:"role_id,omitempty"`
	RoleName      sql.NullString `db:"role_name,omitempty"`
	Relation      string         `db:"relation"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at,omitempty"`
	ConfirmedAt   sql.NullTime   `db:"confirmed_at,omitempty"`
	RejectedAt    sql.NullTime   `db:"rejected_at,omitempty"`
}

func toDBInvitation(inv domains.Invitation) dbInvitation {
	var updatedAt, confirmedAt, rejectedAt sql.NullTime
	if inv.UpdatedAt != (time.Time{}) {
		updatedAt = sql.NullTime{Time: inv.UpdatedAt, Valid: true}
	}
	if inv.ConfirmedAt != (time.Time{}) {
		confirmedAt = sql.NullTime{Time: inv.ConfirmedAt, Valid: true}
	}
	if inv.RejectedAt != (time.Time{}) {
		rejectedAt = sql.NullTime{Time: inv.RejectedAt, Valid: true}
	}

	return dbInvitation{
		InvitedBy:     inv.InvitedBy,
		InviteeUserID: inv.InviteeUserID,
		DomainID:      inv.DomainID,
		RoleID:        inv.RoleID,
		CreatedAt:     inv.CreatedAt,
		UpdatedAt:     updatedAt,
		ConfirmedAt:   confirmedAt,
		RejectedAt:    rejectedAt,
	}
}

func toInvitation(dbinv dbInvitation) domains.Invitation {
	var updatedAt, confirmedAt, rejectedAt time.Time
	if dbinv.UpdatedAt.Valid {
		updatedAt = dbinv.UpdatedAt.Time
	}
	if dbinv.ConfirmedAt.Valid {
		confirmedAt = dbinv.ConfirmedAt.Time
	}
	if dbinv.RejectedAt.Valid {
		rejectedAt = dbinv.RejectedAt.Time
	}

	return domains.Invitation{
		InvitedBy:     dbinv.InvitedBy,
		InviteeUserID: dbinv.InviteeUserID,
		DomainID:      dbinv.DomainID,
		DomainName:    toString(dbinv.DomainName),
		RoleID:        dbinv.RoleID,
		RoleName:      toString(dbinv.RoleName),
		CreatedAt:     dbinv.CreatedAt,
		UpdatedAt:     updatedAt,
		ConfirmedAt:   confirmedAt,
		RejectedAt:    rejectedAt,
	}
}

func toString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
