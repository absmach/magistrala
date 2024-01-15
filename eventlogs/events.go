// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package eventlogs

import (
	"context"
	"encoding/json"
	"time"
)

// Event represents an event.
type Event struct {
	ID         string                 `json:"id,omitempty" db:"id,omitempty"`
	Operation  string                 `json:"operation,omitempty" db:"operation,omitempty"`
	OccurredAt time.Time              `json:"occurred_at,omitempty" db:"occurred_at,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty" db:"payload,omitempty"`
}

// EventsPage represents a page of events.
type EventsPage struct {
	Total  uint64  `json:"total"`
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Events []Event `json:"events"`
}

// Page is used to filter events.
type Page struct {
	Offset      uint64    `json:"offset" db:"offset"`
	Limit       uint64    `json:"limit" db:"limit"`
	ID          string    `json:"id,omitempty" db:"id,omitempty"`
	EntityType  string    `json:"entity_type,omitempty"`
	Operation   string    `json:"operation,omitempty" db:"operation,omitempty"`
	From        time.Time `json:"from,omitempty" db:"from,omitempty"`
	To          time.Time `json:"to,omitempty" db:"to,omitempty"`
	WithPayload bool      `json:"with_payload,omitempty"`
}

func (page EventsPage) MarshalJSON() ([]byte, error) {
	type Alias EventsPage
	a := struct {
		Alias
	}{
		Alias: Alias(page),
	}

	if a.Events == nil {
		a.Events = make([]Event, 0)
	}

	return json.Marshal(a)
}

// Service provides access to the event log service.
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// ReadAll retrieves all events from the database with the given page.
	ReadAll(ctx context.Context, token string, page Page) (EventsPage, error)
}

// Repository provides access to the event log database.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// Save persists the event to a database.
	Save(ctx context.Context, event Event) error

	// RetrieveAll retrieves all events from the database with the given page.
	RetrieveAll(ctx context.Context, page Page) (EventsPage, error)
}
