// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

var _ auth.PATSRepository = (*patRepo)(nil)

type patRepo struct {
	db    postgres.Database
	cache auth.Cache
}

func NewPatRepo(db postgres.Database, cache auth.Cache) auth.PATSRepository {
	return &patRepo{
		db:    db,
		cache: cache,
	}
}

func (pr *patRepo) Save(ctx context.Context, pat auth.PAT) error {
	q := `
	INSERT INTO pats (
		id, user_id, name, description, secret, issued_at, expires_at, 
		updated_at, last_used_at, revoked, revoked_at
	) VALUES (
		:id, :user_id, :name, :description, :secret, :issued_at, :expires_at,
		:updated_at, :last_used_at, :revoked, :revoked_at
	)`

	dbPat, err := toDBPats(pat)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	_, err = pr.db.NamedQueryContext(ctx, q, dbPat)
	if err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (pr *patRepo) Retrieve(ctx context.Context, userID, patID string) (auth.PAT, error) {
	pat, err := pr.retrievePATFromDB(ctx, userID, patID)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return pat, nil
}

func (pr *patRepo) RetrieveAll(ctx context.Context, userID string, pm auth.PATSPageMeta) (auth.PATSPage, error) {
	q := `
		SELECT 
		p.id, p.user_id, p.name, p.description, p.issued_at, p.expires_at,
		p.updated_at, p.revoked, p.revoked_at
		FROM pats p WHERE user_id = :user_id
		ORDER BY issued_at DESC
		LIMIT :limit OFFSET :offset`

	dbPage := dbPagemeta{
		Limit:  pm.Limit,
		Offset: pm.Offset,
		User:   userID,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return auth.PATSPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []auth.PAT
	for rows.Next() {
		var pat dbPat
		if err := rows.StructScan(&pat); err != nil {
			return auth.PATSPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		var updatedAt, revokedAt time.Time
		if pat.UpdatedAt.Valid {
			updatedAt = pat.UpdatedAt.Time
		}
		if pat.RevokedAt.Valid {
			revokedAt = pat.RevokedAt.Time
		}

		items = append(items, auth.PAT{
			ID:          pat.ID,
			User:        pat.User,
			Name:        pat.Name,
			Description: pat.Description,
			IssuedAt:    pat.IssuedAt,
			ExpiresAt:   pat.ExpiresAt,
			UpdatedAt:   updatedAt,
			Revoked:     pat.Revoked,
			RevokedAt:   revokedAt,
		})
	}

	cq := `SELECT COUNT(*) FROM pats p WHERE user_id = :user_id`

	total, err := postgres.Total(ctx, pr.db, cq, dbPage)
	if err != nil {
		return auth.PATSPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := auth.PATSPage{
		PATS:   items,
		Total:  total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
	}
	return page, nil
}

func (pr *patRepo) RetrieveSecretAndRevokeStatus(ctx context.Context, userID, patID string) (string, bool, bool, error) {
	q := `
		SELECT p.secret, p.revoked, p.expires_at 
		FROM pats p
		WHERE user_id = $1 AND id = $2`

	rows, err := pr.db.QueryContext(ctx, q, userID, patID)
	if err != nil {
		return "", true, true, postgres.HandleError(repoerr.ErrNotFound, err)
	}
	defer rows.Close()

	var secret string
	var revoked bool
	var expiresAt time.Time

	if !rows.Next() {
		return "", true, true, repoerr.ErrNotFound
	}

	if err := rows.Scan(&secret, &revoked, &expiresAt); err != nil {
		return "", true, true, postgres.HandleError(repoerr.ErrNotFound, err)
	}

	expired := time.Now().After(expiresAt)
	return secret, revoked, expired, nil
}

func (pr *patRepo) UpdateName(ctx context.Context, userID, patID, name string) (auth.PAT, error) {
	q := `
		UPDATE pats p
		SET name = :name, updated_at = :updated_at
		WHERE user_id = :user_id AND id = :id
		RETURNING id, user_id, name, description, secret, issued_at, updated_at, expires_at, revoked, revoked_at, last_used_at`

	upm := dbPagemeta{
		User: userID,
		ID:   patID,
		Name: name,
		UpdatedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	if !rows.Next() {
		return auth.PAT{}, repoerr.ErrNotFound
	}

	var pat dbPat
	if err := rows.StructScan(&pat); err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	res, err := toAuthPat(pat)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return res, nil
}

func (pr *patRepo) UpdateDescription(ctx context.Context, userID, patID, description string) (auth.PAT, error) {
	q := `
		UPDATE pats 
		SET description = :description, updated_at = :updated_at
		WHERE user_id = :user_id AND id = :id
		RETURNING id, user_id, name, description, secret, issued_at, updated_at, expires_at, revoked, revoked_at, last_used_at`

	upm := dbPagemeta{
		User: userID,
		ID:   patID,
		UpdatedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		Description: description,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	if !rows.Next() {
		return auth.PAT{}, repoerr.ErrNotFound
	}

	var pat dbPat
	if err := rows.StructScan(&pat); err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	res, err := toAuthPat(pat)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return res, nil
}

func (pr *patRepo) UpdateTokenHash(ctx context.Context, userID, patID, tokenHash string, expiryAt time.Time) (auth.PAT, error) {
	q := `
		UPDATE pats 
		SET secret = :secret, expires_at = :expires_at, updated_at = :updated_at
		WHERE user_id = :user_id AND id = :id
		RETURNING id, user_id, name, description, secret, issued_at, updated_at, expires_at, revoked, revoked_at, last_used_at`

	upm := dbPagemeta{
		User: userID,
		ID:   patID,
		UpdatedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		ExpiresAt: expiryAt,
		Secret:    tokenHash,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	if !rows.Next() {
		return auth.PAT{}, repoerr.ErrNotFound
	}

	var pat dbPat
	if err := rows.StructScan(&pat); err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	res, err := toAuthPat(pat)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return res, nil
}

func (pr *patRepo) Revoke(ctx context.Context, userID, patID string) error {
	q := `
		UPDATE pats 
		SET revoked = true, revoked_at = :revoked_at
		WHERE user_id = :user_id AND id = :id`

	upm := dbPagemeta{
		User: userID,
		ID:   patID,
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	}

	_, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

func (pr *patRepo) Reactivate(ctx context.Context, userID, patID string) error {
	q := `
		UPDATE pats 
		SET revoked = false, revoked_at = NULL
		WHERE user_id = :user_id AND id = :id`

	upm := dbPagemeta{
		User: userID,
		ID:   patID,
	}

	_, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

func (pr *patRepo) Remove(ctx context.Context, userID, patID string) error {
	q := `DELETE FROM pats WHERE user_id = :user_id AND id = :id`
	upm := dbPagemeta{
		User: userID,
		ID:   patID,
	}

	_, err := pr.db.NamedQueryContext(ctx, q, upm)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pr *patRepo) RemoveAllPAT(ctx context.Context, userID string) error {
	q := `DELETE FROM pats WHERE user_id = :user_id`

	pm := dbPagemeta{
		User: userID,
	}

	_, err := pr.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := pr.cache.RemoveUserAllScope(ctx, userID); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pr *patRepo) AddScope(ctx context.Context, userID string, scopes []auth.Scope) error {
	q := `
		INSERT INTO pat_scopes (id, pat_id, entity_type, optional_domain_id, operation, entity_id)
		VALUES (:id, :pat_id, :entity_type, :optional_domain_id, :operation, :entity_id)`

	var newScopes []auth.Scope

	for _, sc := range scopes {
		processedScope, err := pr.processScope(ctx, sc)
		if err != nil {
			return err
		}
		if processedScope.ID != "" {
			newScopes = append(newScopes, processedScope)
		}
	}

	if len(newScopes) > 0 {
		_, err := pr.db.NamedQueryContext(ctx, q, toDBScope(newScopes))
		if err != nil {
			return postgres.HandleError(repoerr.ErrUpdateEntity, err)
		}
	}

	if err := pr.cache.Save(ctx, userID, scopes); err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

func (pr *patRepo) processScope(ctx context.Context, sc auth.Scope) (auth.Scope, error) {
	q := `
		SELECT COUNT(*) 
		FROM pat_scopes 
		WHERE pat_id = :pat_id 
		  AND entity_type = :entity_type
		  AND optional_domain_id = :optional_domain_id
		  AND operation = :operation
		  AND entity_id = :entity_id
		LIMIT 1`

	params := dbScope{
		PatID:            sc.PatID,
		OptionalDomainID: sc.OptionalDomainID,
		EntityType:       sc.EntityType.String(),
		Operation:        sc.Operation.String(),
		EntityID:         auth.AnyIDs,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return auth.Scope{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return auth.Scope{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
	}

	if count > 0 {
		return auth.Scope{}, repoerr.ErrConflict
	}

	if sc.EntityID == auth.AnyIDs {
		newParams := dbScope{
			PatID:            sc.PatID,
			OptionalDomainID: sc.OptionalDomainID,
			EntityType:       sc.EntityType.String(),
			Operation:        sc.Operation.String(),
		}

		checkEntityQuery := `
			SELECT COUNT(*) 
			FROM pat_scopes 
			WHERE pat_id = :pat_id 
			AND entity_type = :entity_type
			AND optional_domain_id = :optional_domain_id
			AND operation = :operation
			LIMIT 1`

		rows, err := pr.db.NamedQueryContext(ctx, checkEntityQuery, newParams)
		if err != nil {
			return auth.Scope{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
		}
		defer rows.Close()

		var count int
		if rows.Next() {
			if err := rows.Scan(&count); err != nil {
				return auth.Scope{}, postgres.HandleError(repoerr.ErrViewEntity, err)
			}
		}

		if count > 0 {
			updateWithWildcardQuery := `
			UPDATE pat_scopes 
			SET entity_id = :entity_id 
			WHERE pat_id = :pat_id 
			AND entity_type = :entity_type
			AND optional_domain_id = :optional_domain_id
			AND operation = :operation`

			_, err = pr.db.NamedQueryContext(ctx, updateWithWildcardQuery, params)
			if err != nil {
				return auth.Scope{}, postgres.HandleError(repoerr.ErrUpdateEntity, err)
			}
			return auth.Scope{}, nil
		}
	}

	return sc, nil
}

func (pr *patRepo) RemoveScope(ctx context.Context, userID string, scopesIDs ...string) error {
	deleteScopesQuery := fmt.Sprintf(`DELETE FROM pat_scopes WHERE id IN ('%s')`, strings.Join(scopesIDs, ","))

	res, err := pr.db.ExecContext(ctx, deleteScopesQuery)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	if err := pr.cache.Remove(ctx, userID, scopesIDs); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pr *patRepo) CheckScope(ctx context.Context, userID, patID string, entityType auth.EntityType, optionalDomainID string, operation auth.Operation, entityID string) error {
	q := `
        SELECT id, pat_id, entity_type, optional_domain_id, operation, entity_id
        FROM pat_scopes 
        WHERE pat_id = :pat_id 
          AND entity_type = :entity_type
          AND optional_domain_id = :optional_domain_id
          AND operation = :operation
          AND (entity_id = :entity_id OR entity_id = '*')
        LIMIT 1`

	authorized := pr.cache.CheckScope(ctx, userID, patID, optionalDomainID, entityType, operation, entityID)
	if authorized {
		return nil
	}

	scope := dbScope{
		PatID:            patID,
		EntityType:       entityType.String(),
		OptionalDomainID: optionalDomainID,
		Operation:        operation.String(),
		EntityID:         entityID,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, scope)
	if err != nil {
		return errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	if rows.Next() {
		var sc dbScope
		if err := rows.StructScan(&sc); err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}

		entityType, err := auth.ParseEntityType(sc.EntityType)
		if err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}
		operation, err := auth.ParseOperation(sc.Operation)
		if err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}
		authScope := auth.Scope{
			ID:               sc.ID,
			PatID:            sc.PatID,
			OptionalDomainID: sc.OptionalDomainID,
			EntityType:       entityType,
			EntityID:         sc.EntityID,
			Operation:        operation,
		}

		if err := pr.cache.Save(ctx, userID, []auth.Scope{authScope}); err != nil {
			return err
		}

		if authScope.Authorized(entityType, optionalDomainID, operation, entityID) {
			return nil
		}
	}

	return repoerr.ErrNotFound
}

func (pr *patRepo) RemoveAllScope(ctx context.Context, patID string) error {
	pm := dbPagemeta{
		PatID: patID,
	}

	q := `DELETE FROM pat_scopes WHERE pat_id = :pat_id`

	_, err := pr.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}

	if err := pr.cache.RemoveAllScope(ctx, pm.User, patID); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pr *patRepo) RetrieveScope(ctx context.Context, pm auth.ScopesPageMeta) (auth.ScopesPage, error) {
	dbs := dbPagemeta{
		PatID:  pm.PatID,
		Offset: pm.Offset,
		Limit:  pm.Limit,
	}

	scopes, err := pr.retrieveScopeFromDB(ctx, dbs)
	if err != nil {
		return auth.ScopesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	cq := `SELECT COUNT(*) FROM pat_scopes WHERE pat_id = :pat_id`

	total, err := postgres.Total(ctx, pr.db, cq, dbs)
	if err != nil {
		return auth.ScopesPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	return auth.ScopesPage{
		Total:  total,
		Scopes: scopes,
		Offset: pm.Offset,
		Limit:  pm.Limit,
	}, nil
}

func (pr *patRepo) retrieveScopeFromDB(ctx context.Context, pm dbPagemeta) ([]auth.Scope, error) {
	q := `
		SELECT id, pat_id, entity_type, optional_domain_id, operation, entity_id
		FROM pat_scopes WHERE pat_id = :pat_id OFFSET :offset LIMIT :limit`
	scopeRows, err := pr.db.NamedQueryContext(ctx, q, pm)
	if err != nil {
		return []auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer scopeRows.Close()

	var scopes []dbScope
	for scopeRows.Next() {
		var scope dbScope
		if err := scopeRows.StructScan(&scope); err != nil {
			return []auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		scopes = append(scopes, scope)
	}

	sc, err := toAuthScope(scopes)
	if err != nil {
		return []auth.Scope{}, err
	}

	return sc, nil
}

func (pr *patRepo) retrievePATFromDB(ctx context.Context, userID, patID string) (auth.PAT, error) {
	q := `
		SELECT 
		id, user_id, name, description, secret, issued_at, expires_at,
		updated_at, last_used_at, revoked, revoked_at
		FROM pats WHERE user_id = :user_id AND id = :id`

	dbp := dbPagemeta{
		ID:   patID,
		User: userID,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, dbp)
	if err != nil {
		return auth.PAT{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var record dbPat
	if rows.Next() {
		if err := rows.StructScan(&record); err != nil {
			return auth.PAT{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
	}

	pat, err := toAuthPat(record)
	if err != nil {
		return auth.PAT{}, err
	}

	return pat, nil
}
