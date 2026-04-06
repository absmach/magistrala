// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/jackc/pgtype"
)

func (repo *repository) SaveClientTelemetry(ctx context.Context, ct journal.ClientTelemetry) error {
	q := `INSERT INTO clients_telemetry (client_id, domain_id, inbound_messages, outbound_messages, first_seen, last_seen)
		VALUES (:client_id, :domain_id, :inbound_messages, :outbound_messages, :first_seen, :last_seen);`

	dbct, err := toDBClientsTelemetry(ct)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	if _, err := repo.db.NamedExecContext(ctx, q, dbct); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *repository) DeleteClientTelemetry(ctx context.Context, clientID, domainID string) error {
	q := `DELETE FROM clients_telemetry AS ct WHERE ct.client_id = :client_id AND ct.domain_id = :domain_id;`

	dbct := dbClientTelemetry{
		ClientID: clientID,
		DomainID: domainID,
	}

	result, err := repo.db.NamedExecContext(ctx, q, dbct)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) RetrieveClientTelemetry(ctx context.Context, clientID, domainID string) (journal.ClientTelemetry, error) {
	q := `SELECT * FROM clients_telemetry WHERE client_id = :client_id AND domain_id = :domain_id;`

	dbct := dbClientTelemetry{
		ClientID: clientID,
		DomainID: domainID,
	}

	rows, err := repo.db.NamedQueryContext(ctx, q, dbct)
	if err != nil {
		return journal.ClientTelemetry{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbct = dbClientTelemetry{}
	if rows.Next() {
		if err = rows.StructScan(&dbct); err != nil {
			return journal.ClientTelemetry{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		ct, err := toClientsTelemetry(dbct)
		if err != nil {
			return journal.ClientTelemetry{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return ct, nil
	}

	return journal.ClientTelemetry{}, repoerr.ErrNotFound
}

func (repo *repository) AddSubscription(ctx context.Context, sub journal.ClientSubscription) error {
	q := `
		INSERT INTO subscriptions (id, subscriber_id, channel_id, subtopic, client_id)
		SELECT :id, :subscriber_id, :channel_id, :subtopic, :client_id
		FROM clients_telemetry
		WHERE client_id = :client_id
		RETURNING id;
	`

	_, err := repo.db.NamedExecContext(ctx, q, sub)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

func (repo *repository) CountSubscriptions(ctx context.Context, clientID string) (uint64, error) {
	q := `SELECT COUNT(*) FROM subscriptions WHERE client_id = :client_id;`

	sb := journal.ClientSubscription{
		ClientID: clientID,
	}

	total, err := postgres.Total(ctx, repo.db, q, sb)
	if err != nil {
		return 0, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	return total, nil
}

func (repo *repository) RemoveSubscription(ctx context.Context, subscriberID string) error {
	q := `DELETE FROM subscriptions WHERE subscriber_id = :subscriber_id;`

	sb := journal.ClientSubscription{
		SubscriberID: subscriberID,
	}

	_, err := repo.db.NamedExecContext(ctx, q, sb)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

func (repo *repository) IncrementInboundMessages(ctx context.Context, ct journal.ClientTelemetry) error {
	q := `INSERT INTO clients_telemetry (client_id,domain_id, inbound_messages,first_seen, last_seen)
		VALUES (:client_id, :domain_id, 1, :first_seen, :last_seen)
		ON CONFLICT (client_id)
		DO UPDATE SET
			inbound_messages = clients_telemetry.inbound_messages + 1,
			last_seen = EXCLUDED.last_seen;
	`

	dbct, err := toDBClientsTelemetry(ct)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	result, err := repo.db.NamedExecContext(ctx, q, dbct)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *repository) IncrementOutboundMessages(ctx context.Context, channelID, subtopic string) error {
	query := `
		SELECT client_id, COUNT(*) AS match_count
		FROM subscriptions
		WHERE channel_id = :channel_id AND subtopic = :subtopic
		GROUP BY client_id
	`
	sb := journal.ClientSubscription{
		ChannelID: channelID,
		Subtopic:  subtopic,
	}

	rows, err := repo.db.NamedQueryContext(ctx, query, sb)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	defer rows.Close()

	tx, err := repo.db.BeginTxx(ctx, nil)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	q := `UPDATE clients_telemetry
		SET outbound_messages = outbound_messages + $1
		WHERE client_id = $2;
		`

	for rows.Next() {
		var clientID string
		var count uint64
		if err = rows.Scan(&clientID, &count); err != nil {
			if err := tx.Rollback(); err != nil {
				return errors.Wrap(errors.ErrRollbackTx, err)
			}
			return postgres.HandleError(repoerr.ErrUpdateEntity, err)
		}

		if _, err = repo.db.ExecContext(ctx, q, count, clientID); err != nil {
			if err := tx.Rollback(); err != nil {
				return errors.Wrap(errors.ErrRollbackTx, err)
			}
			return errors.Wrap(errors.ErrRollbackTx, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}

	return nil
}

type dbClientTelemetry struct {
	ClientID         string       `db:"client_id"`
	DomainID         string       `db:"domain_id"`
	InboundMessages  uint64       `db:"inbound_messages"`
	OutboundMessages uint64       `db:"outbound_messages"`
	FirstSeen        time.Time    `db:"first_seen"`
	LastSeen         sql.NullTime `db:"last_seen"`
}

func toDBClientsTelemetry(ct journal.ClientTelemetry) (dbClientTelemetry, error) {
	var subs pgtype.TextArray
	if err := subs.Set(ct.Subscriptions); err != nil {
		return dbClientTelemetry{}, err
	}

	var lastSeen sql.NullTime
	if ct.LastSeen != (time.Time{}) {
		lastSeen = sql.NullTime{Time: ct.LastSeen, Valid: true}
	}

	return dbClientTelemetry{
		ClientID:         ct.ClientID,
		DomainID:         ct.DomainID,
		InboundMessages:  ct.InboundMessages,
		OutboundMessages: ct.OutboundMessages,
		FirstSeen:        ct.FirstSeen,
		LastSeen:         lastSeen,
	}, nil
}

func toClientsTelemetry(dbct dbClientTelemetry) (journal.ClientTelemetry, error) {
	var lastSeen time.Time
	if dbct.LastSeen.Valid {
		lastSeen = dbct.LastSeen.Time
	}

	return journal.ClientTelemetry{
		ClientID:         dbct.ClientID,
		DomainID:         dbct.DomainID,
		InboundMessages:  dbct.InboundMessages,
		OutboundMessages: dbct.OutboundMessages,
		FirstSeen:        dbct.FirstSeen,
		LastSeen:         lastSeen,
	}, nil
}
