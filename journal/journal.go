// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/policies"
)

type EntityType uint8

const (
	UserEntity EntityType = iota
	GroupEntity
	ClientEntity
	ChannelEntity
)

// String representation of the possible entity type values.
const (
	userEntityType    = "user"
	groupEntityType   = "group"
	clientEntityType  = "client"
	channelEntityType = "channel"
)

// String converts entity type to string literal.
func (e EntityType) String() string {
	switch e {
	case UserEntity:
		return userEntityType
	case GroupEntity:
		return groupEntityType
	case ClientEntity:
		return clientEntityType
	case ChannelEntity:
		return channelEntityType
	default:
		return ""
	}
}

// AuthString returns the entity type as a string for authorization.
func (e EntityType) AuthString() string {
	switch e {
	case UserEntity:
		return policies.UserType
	case GroupEntity, ChannelEntity:
		return policies.GroupType
	case ClientEntity:
		return policies.ClientType
	default:
		return ""
	}
}

// ToEntityType converts string value to a valid entity type.
func ToEntityType(entityType string) (EntityType, error) {
	switch entityType {
	case userEntityType:
		return UserEntity, nil
	case groupEntityType:
		return GroupEntity, nil
	case clientEntityType:
		return ClientEntity, nil
	case channelEntityType:
		return ChannelEntity, nil
	default:
		return EntityType(0), apiutil.ErrInvalidEntityType
	}
}

// Query returns the SQL condition for the entity type.
func (e EntityType) Query() string {
	switch e {
	case UserEntity:
		return "((operation LIKE 'user.%' AND attributes->>'id' = :entity_id) OR (attributes->>'user_id' = :entity_id))"
	case GroupEntity, ChannelEntity:
		return "((operation LIKE 'group.%' AND attributes->>'id' = :entity_id) OR (attributes->>'group_id' = :entity_id))"
	case ClientEntity:
		return "((operation LIKE 'client.%' AND attributes->>'id' = :entity_id) OR (attributes->>'client_id' = :entity_id))"
	default:
		return ""
	}
}

// Journal represents an event journal that occurred in the system.
type Journal struct {
	ID         string                 `json:"id,omitempty" db:"id"`
	Operation  string                 `json:"operation,omitempty" db:"operation,omitempty"`
	OccurredAt time.Time              `json:"occurred_at,omitempty" db:"occurred_at,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty" db:"attributes,omitempty"` // This is extra information about the journal for example client_id, user_id, group_id etc.
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata,omitempty"`     // This is decoded metadata from the journal.
}

// JournalsPage represents a page of journals.
type JournalsPage struct {
	Total    uint64    `json:"total"`
	Offset   uint64    `json:"offset"`
	Limit    uint64    `json:"limit"`
	Journals []Journal `json:"journals"`
}

// Page is used to filter journals.
type Page struct {
	Offset         uint64     `json:"offset" db:"offset"`
	Limit          uint64     `json:"limit" db:"limit"`
	Operation      string     `json:"operation,omitempty" db:"operation,omitempty"`
	From           time.Time  `json:"from,omitempty" db:"from,omitempty"`
	To             time.Time  `json:"to,omitempty" db:"to,omitempty"`
	WithAttributes bool       `json:"with_attributes,omitempty"`
	WithMetadata   bool       `json:"with_metadata,omitempty"`
	EntityID       string     `json:"entity_id,omitempty" db:"entity_id,omitempty"`
	EntityType     EntityType `json:"entity_type,omitempty" db:"entity_type,omitempty"`
	Direction      string     `json:"direction,omitempty"`
}

func (page JournalsPage) MarshalJSON() ([]byte, error) {
	type Alias JournalsPage
	a := struct {
		Alias
	}{
		Alias: Alias(page),
	}

	if a.Journals == nil {
		a.Journals = make([]Journal, 0)
	}

	return json.Marshal(a)
}

// Service provides access to the journal log service.
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// Save saves the journal to the database.
	Save(ctx context.Context, journal Journal) error

	// RetrieveAll retrieves all journals from the database with the given page.
	RetrieveAll(ctx context.Context, session mgauthn.Session, page Page) (JournalsPage, error)
}

// Repository provides access to the journal log database.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// Save persists the journal to a database.
	Save(ctx context.Context, journal Journal) error

	// RetrieveAll retrieves all journals from the database with the given page.
	RetrieveAll(ctx context.Context, page Page) (JournalsPage, error)
}
