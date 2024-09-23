// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/clients"
)

type AuthzReq struct {
	ChannelID  string
	ThingID    string
	ThingKey   string
	Permission string
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateThings creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateThings(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, error)

	// ViewClient retrieves client info for a given client ID and an authorized token.
	ViewClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// ViewClientPerms retrieves permissions on the client id for the given authorized token.
	ViewClientPerms(ctx context.Context, session authn.Session, id string) ([]string, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error)

	// ListClientsByGroup retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm clients.Page) (clients.MembersPage, error)

	// UpdateClient updates the client's name and metadata.
	UpdateClient(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientTags updates the client's tags.
	UpdateClientTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientSecret updates the client's secret
	UpdateClientSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error)

	// EnableClient logically enableds the client identified with the provided ID
	EnableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// DisableClient logically disables the client identified with the provided ID
	DisableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// Share add share policy to thing id with given relation for given user ids
	Share(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Unshare remove share policy to thing id with given relation for given user ids
	Unshare(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// Authorize used for Things authorization.
	Authorize(ctx context.Context, req AuthzReq) (string, error)

	// DeleteClient deletes client with given ID.
	DeleteClient(ctx context.Context, session authn.Session, id string) error
}

// Cache contains thing caching interface.
//
//go:generate mockery --name Cache --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	// Save stores pair thing secret, thing id.
	Save(ctx context.Context, thingSecret, thingID string) error

	// ID returns thing ID for given thing secret.
	ID(ctx context.Context, thingSecret string) (string, error)

	// Removes thing from cache.
	Remove(ctx context.Context, thingID string) error
}

// Things Service separate from Client Struct
type Things struct {
	ID          string         `json:"id"`
	Name        string         `json:"name,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Domain      string         `json:"domain_id,omitempty"`
	Credentials Credentials    `json:"credentials,omitempty"`
	Metadata    Metadata       `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
	UpdatedBy   string         `json:"updated_by,omitempty"`
	Status      clients.Status `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Permissions []string       `json:"permissions,omitempty"`
}

type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}
