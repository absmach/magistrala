// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"time"

	"github.com/absmach/supermq/auth"
)

type dbPat struct {
	ID          string       `db:"id,omitempty"`
	User        string       `db:"user_id,omitempty"`
	Name        string       `db:"name,omitempty"`
	Description string       `db:"description,omitempty"`
	Secret      string       `db:"secret,omitempty"`
	IssuedAt    time.Time    `db:"issued_at,omitempty"`
	ExpiresAt   time.Time    `db:"expires_at,omitempty"`
	UpdatedAt   sql.NullTime `db:"updated_at,omitempty"`
	LastUsedAt  sql.NullTime `db:"last_used_at,omitempty"`
	Revoked     bool         `db:"revoked,omitempty"`
	RevokedAt   sql.NullTime `db:"revoked_at,omitempty"`
	Status      auth.Status  `db:"status,omitempty"`
	TotalCount  uint64       `db:"total_count"`
}

type dbScope struct {
	ID         string `db:"id,omitempty"`
	PatID      string `db:"pat_id,omitempty"`
	DomainID   string `db:"domain_id,omitempty"`
	EntityType string `db:"entity_type,omitempty"`
	EntityID   string `db:"entity_id,omitempty"`
	Operation  string `db:"operation,omitempty"`
}

type dbPagemeta struct {
	Limit       uint64       `db:"limit"`
	Offset      uint64       `db:"offset"`
	User        string       `db:"user_id"`
	PatID       string       `db:"pat_id"`
	ScopesID    []string     `db:"scopes_id"`
	ID          string       `db:"id"`
	Name        string       `db:"name"`
	UpdatedAt   sql.NullTime `db:"updated_at"`
	ExpiresAt   time.Time    `db:"expires_at"`
	RevokedAt   sql.NullTime `db:"revoked_at"`
	Description string       `db:"description"`
	Secret      string       `db:"secret"`
	Status      auth.Status  `db:"status"`
	Timestamp   time.Time    `db:"timestamp,omitempty"`
}

func toAuthPat(db dbPat) auth.PAT {
	updatedAt := time.Time{}
	lastUsedAt := time.Time{}
	revokedAt := time.Time{}

	if db.UpdatedAt.Valid {
		updatedAt = db.UpdatedAt.Time
	}

	if db.LastUsedAt.Valid {
		lastUsedAt = db.LastUsedAt.Time
	}

	if db.RevokedAt.Valid {
		revokedAt = db.RevokedAt.Time
	}

	return auth.PAT{
		ID:          db.ID,
		User:        db.User,
		Name:        db.Name,
		Description: db.Description,
		Secret:      db.Secret,
		IssuedAt:    db.IssuedAt,
		ExpiresAt:   db.ExpiresAt,
		UpdatedAt:   updatedAt,
		LastUsedAt:  lastUsedAt,
		Revoked:     db.Revoked,
		RevokedAt:   revokedAt,
		Status:      db.Status,
	}
}

func toAuthScope(dsc []dbScope) ([]auth.Scope, error) {
	scope := []auth.Scope{}

	for _, s := range dsc {
		entityType, err := auth.ParseEntityType(s.EntityType)
		if err != nil {
			return []auth.Scope{}, err
		}
		scope = append(scope, auth.Scope{
			ID:         s.ID,
			PatID:      s.PatID,
			DomainID:   s.DomainID,
			EntityType: entityType,
			EntityID:   s.EntityID,
			Operation:  s.Operation,
		})
	}

	return scope, nil
}

func toDBPats(pat auth.PAT) (dbPat, error) {
	var updatedAt, lastUsedAt, revokedAt sql.NullTime

	if !pat.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{
			Time:  pat.UpdatedAt,
			Valid: true,
		}
	}

	if !pat.LastUsedAt.IsZero() {
		lastUsedAt = sql.NullTime{
			Time:  pat.LastUsedAt,
			Valid: true,
		}
	}

	if !pat.RevokedAt.IsZero() {
		revokedAt = sql.NullTime{
			Time:  pat.RevokedAt,
			Valid: true,
		}
	}

	return dbPat{
		ID:          pat.ID,
		User:        pat.User,
		Name:        pat.Name,
		Description: pat.Description,
		Secret:      pat.Secret,
		IssuedAt:    pat.IssuedAt,
		ExpiresAt:   pat.ExpiresAt,
		Revoked:     pat.Revoked,
		UpdatedAt:   updatedAt,
		LastUsedAt:  lastUsedAt,
		RevokedAt:   revokedAt,
	}, nil
}

func toDBScope(sc []auth.Scope) []dbScope {
	var scopes []dbScope
	for _, s := range sc {
		scopes = append(scopes, dbScope{
			ID:         s.ID,
			PatID:      s.PatID,
			DomainID:   s.DomainID,
			EntityType: s.EntityType.String(),
			EntityID:   s.EntityID,
			Operation:  s.Operation,
		})
	}
	return scopes
}
