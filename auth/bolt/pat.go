// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bolt

import (
	"context"
	"time"

	"github.com/absmach/magistrala/auth"
	"go.etcd.io/bbolt"
)

type patRepo struct {
	db *bbolt.DB
}

// NewPATRepository instantiates a bbolt
// implementation of PAT repository.
func NewDomainRepository(db *bbolt.DB) auth.PATSRepository {
	return &patRepo{
		db: db,
	}
}

func (pr *patRepo) Save(ctx context.Context, pat auth.PAT) (id string, err error) {
	return "", nil
}

func (pr *patRepo) Retrieve(ctx context.Context, userID, patID string) (pat auth.PAT, err error) {
	return auth.PAT{}, nil
}

func (pr *patRepo) UpdateName(ctx context.Context, userID, patID, name string) (auth.PAT, error) {
	return auth.PAT{}, nil
}

func (pr *patRepo) UpdateDescription(ctx context.Context, userID, patID, description string) (auth.PAT, error) {
	return auth.PAT{}, nil
}

func (pr *patRepo) UpdateTokenHash(ctx context.Context, userID, patID, tokenHash string, expiryAt time.Time) (auth.PAT, error) {
	return auth.PAT{}, nil
}

func (pr *patRepo) RetrieveAll(ctx context.Context, userID string) (pats auth.PATSPage, err error) {
	return auth.PATSPage{}, nil
}

func (pr *patRepo) Revoke(ctx context.Context, userID, patID string) error {
	return nil
}

func (pr *patRepo) Remove(ctx context.Context, userID, patID string) error {
	return nil
}

func (pr *patRepo) AddScopeEntry(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	return auth.Scope{}, nil
}

func (pr *patRepo) RemoveScopeEntry(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	return auth.Scope{}, nil
}

func (pr *patRepo) RemoveAllScopeEntry(ctx context.Context, userID, patID string) error {
	return nil
}
