// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
)

type repository struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) eventlogs.Repository {
	return &repository{db: db}
}

func (repo *repository) Save(ctx context.Context, event eventlogs.Event) (err error) {
	q := `INSERT INTO events (id, operation, occurred_at, payload)
		VALUES (:id, :operation, :occurred_at, :payload)`

	dbEvent, err := toDBEvent(event)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	if _, err = repo.db.NamedExecContext(ctx, q, dbEvent); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *repository) RetrieveAll(ctx context.Context, page eventlogs.Page) (eventlogs.EventsPage, error) {
	query := pageQuery(page)

	q := fmt.Sprintf("SELECT id, operation, occurred_at, payload FROM events %s ORDER BY occurred_at LIMIT :limit OFFSET :offset;", query)

	rows, err := repo.db.NamedQueryContext(ctx, q, page)
	if err != nil {
		return eventlogs.EventsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []eventlogs.Event
	for rows.Next() {
		var item dbEvent
		if err = rows.StructScan(&item); err != nil {
			return eventlogs.EventsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		event, err := toEvent(item)
		if err != nil {
			return eventlogs.EventsPage{}, err
		}
		items = append(items, event)
	}

	tq := fmt.Sprintf(`SELECT COUNT(*) FROM events %s`, query)

	total, err := postgres.Total(ctx, repo.db, tq, page)
	if err != nil {
		return eventlogs.EventsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	eventsPage := eventlogs.EventsPage{
		Total:  total,
		Offset: page.Offset,
		Limit:  page.Limit,
		Events: items,
	}

	return eventsPage, nil
}

func pageQuery(pm eventlogs.Page) string {
	var query []string
	var emq string
	if pm.ID != "" {
		query = append(query, "id = :id")
	}
	if pm.Operation != "" {
		query = append(query, "operation = :operation")
	}
	if !pm.From.IsZero() {
		query = append(query, "occurred_at >= :from")
	}
	if !pm.To.IsZero() {
		query = append(query, "occurred_at <= :to")
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq
}

type dbEvent struct {
	ID         string    `db:"id"`
	Operation  string    `db:"operation"`
	OccurredAt time.Time `db:"occurred_at"`
	Payload    []byte    `db:"payload"`
}

func toDBEvent(event eventlogs.Event) (dbEvent, error) {
	data := []byte("{}")
	if len(event.Payload) > 0 {
		b, err := json.Marshal(event.Payload)
		if err != nil {
			return dbEvent{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbEvent{
		ID:         event.ID,
		Operation:  event.Operation,
		OccurredAt: event.OccurredAt,
		Payload:    data,
	}, nil
}

func toEvent(event dbEvent) (eventlogs.Event, error) {
	var payload map[string]interface{}
	if event.Payload != nil {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return eventlogs.Event{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	return eventlogs.Event{
		ID:         event.ID,
		Operation:  event.Operation,
		OccurredAt: event.OccurredAt,
		Payload:    payload,
	}, nil
}
