// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
)

var _ bootstrap.BindingStore = (*bindingRepository)(nil)

type bindingRepository struct {
	db  postgres.Database
	log *slog.Logger
}

// NewBindingRepository instantiates a PostgreSQL implementation of BindingStore.
func NewBindingRepository(db postgres.Database, log *slog.Logger) bootstrap.BindingStore {
	return &bindingRepository{db: db, log: log}
}

func (br bindingRepository) Save(ctx context.Context, configID string, bindings []bootstrap.BindingSnapshot) error {
	if len(bindings) == 0 {
		return nil
	}
	q := `INSERT INTO binding_snapshots (config_id, slot, type, resource_id, snapshot, secret_snapshot, updated_at)
		  VALUES (:config_id, :slot, :type, :resource_id, :snapshot, :secret_snapshot, :updated_at)
		  ON CONFLICT (config_id, slot) DO UPDATE SET
		      type            = EXCLUDED.type,
		      resource_id     = EXCLUDED.resource_id,
		      snapshot        = EXCLUDED.snapshot,
		      secret_snapshot = EXCLUDED.secret_snapshot,
		      updated_at      = EXCLUDED.updated_at`

	now := time.Now().UTC()
	dbBindings := make([]dbBindingSnapshot, 0, len(bindings))
	for _, b := range bindings {
		b.ConfigID = configID
		b.UpdatedAt = now
		dbb, err := toDBBindingSnapshot(b)
		if err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		dbBindings = append(dbBindings, dbb)
	}

	if _, err := br.db.NamedExecContext(ctx, q, dbBindings); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return nil
}

func (br bindingRepository) Retrieve(ctx context.Context, configID string) ([]bootstrap.BindingSnapshot, error) {
	q := `SELECT config_id, slot, type, resource_id, snapshot, secret_snapshot, updated_at
		  FROM binding_snapshots WHERE config_id = $1 ORDER BY slot`

	rows, err := br.db.QueryxContext(ctx, q, configID)
	if err != nil {
		return nil, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var snapshots []bootstrap.BindingSnapshot
	for rows.Next() {
		var dbb dbBindingSnapshot
		if err := rows.StructScan(&dbb); err != nil {
			br.log.Error(fmt.Sprintf("failed to scan binding snapshot: %s", err))
			return nil, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		b, err := toBindingSnapshot(dbb)
		if err != nil {
			return nil, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		snapshots = append(snapshots, b)
	}
	return snapshots, nil
}

func (br bindingRepository) Delete(ctx context.Context, configID, slot string) error {
	q := `DELETE FROM binding_snapshots WHERE config_id = $1 AND slot = $2`
	if _, err := br.db.ExecContext(ctx, q, configID, slot); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

// dbBindingSnapshot is the database representation of a BindingSnapshot.
type dbBindingSnapshot struct {
	ConfigID       string    `db:"config_id"`
	Slot           string    `db:"slot"`
	Type           string    `db:"type"`
	ResourceID     string    `db:"resource_id"`
	Snapshot       []byte    `db:"snapshot"`
	SecretSnapshot []byte    `db:"secret_snapshot"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func toDBBindingSnapshot(b bootstrap.BindingSnapshot) (dbBindingSnapshot, error) {
	snap, err := json.Marshal(b.Snapshot)
	if err != nil {
		return dbBindingSnapshot{}, err
	}
	secret, err := json.Marshal(b.SecretSnapshot)
	if err != nil {
		return dbBindingSnapshot{}, err
	}
	return dbBindingSnapshot{
		ConfigID:       b.ConfigID,
		Slot:           b.Slot,
		Type:           b.Type,
		ResourceID:     b.ResourceID,
		Snapshot:       snap,
		SecretSnapshot: secret,
		UpdatedAt:      b.UpdatedAt,
	}, nil
}

func toBindingSnapshot(dbb dbBindingSnapshot) (bootstrap.BindingSnapshot, error) {
	b := bootstrap.BindingSnapshot{
		ConfigID:   dbb.ConfigID,
		Slot:       dbb.Slot,
		Type:       dbb.Type,
		ResourceID: dbb.ResourceID,
		UpdatedAt:  dbb.UpdatedAt,
	}
	if len(dbb.Snapshot) > 0 && string(dbb.Snapshot) != "null" {
		if err := json.Unmarshal(dbb.Snapshot, &b.Snapshot); err != nil {
			return bootstrap.BindingSnapshot{}, err
		}
	}
	if len(dbb.SecretSnapshot) > 0 && string(dbb.SecretSnapshot) != "null" {
		if err := json.Unmarshal(dbb.SecretSnapshot, &b.SecretSnapshot); err != nil {
			return bootstrap.BindingSnapshot{}, err
		}
	}
	return b, nil
}
