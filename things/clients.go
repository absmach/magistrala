// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/postgres"
)

type AuthzReq struct {
	ChannelID  string
	ClientID   string
	ClientKey  string
	Permission string
}

type ClientRepository struct {
	DB postgres.Database
}

// Repository is the interface that wraps the basic methods for
// a client repository.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// RetrieveByID retrieves client by its unique ID.
	RetrieveByID(ctx context.Context, id string) (Client, error)

	// RetrieveAll retrieves all clients.
	RetrieveAll(ctx context.Context, pm Page) (ClientsPage, error)

	// SearchClients retrieves clients based on search criteria.
	SearchClients(ctx context.Context, pm Page) (ClientsPage, error)

	// RetrieveAllByIDs retrieves for given client IDs .
	RetrieveAllByIDs(ctx context.Context, pm Page) (ClientsPage, error)

	// Update updates the client name and metadata.
	Update(ctx context.Context, client Client) (Client, error)

	// UpdateTags updates the client tags.
	UpdateTags(ctx context.Context, client Client) (Client, error)

	// UpdateIdentity updates identity for client with given id.
	UpdateIdentity(ctx context.Context, client Client) (Client, error)

	// UpdateSecret updates secret for client with given identity.
	UpdateSecret(ctx context.Context, client Client) (Client, error)

	// ChangeStatus changes client status to enabled or disabled
	ChangeStatus(ctx context.Context, client Client) (Client, error)

	// Delete deletes client with given id
	Delete(ctx context.Context, id string) error

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client ...Client) ([]Client, error)

	// RetrieveBySecret retrieves a client based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (Client, error)
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateClients creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateClients(ctx context.Context, session authn.Session, client ...Client) ([]Client, error)

	// View retrieves client info for a given client ID and an authorized token.
	View(ctx context.Context, session authn.Session, id string) (Client, error)

	// ViewPerms retrieves permissions on the client id for the given authorized token.
	ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, session authn.Session, reqUserID string, pm Page) (ClientsPage, error)

	// ListClientsByGroup retrieves data about subset of clients that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm Page) (MembersPage, error)

	// Update updates the client's name and metadata.
	Update(ctx context.Context, session authn.Session, client Client) (Client, error)

	// UpdateTags updates the client's tags.
	UpdateTags(ctx context.Context, session authn.Session, client Client) (Client, error)

	// UpdateSecret updates the client's secret
	UpdateSecret(ctx context.Context, session authn.Session, id, key string) (Client, error)

	// Enable logically enableds the client identified with the provided ID
	Enable(ctx context.Context, session authn.Session, id string) (Client, error)

	// Disable logically disables the client identified with the provided ID
	Disable(ctx context.Context, session authn.Session, id string) (Client, error)

	// Share add share policy to client id with given relation for given user ids
	Share(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Unshare remove share policy to client id with given relation for given user ids
	Unshare(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Identify returns client ID for given client key.
	Identify(ctx context.Context, key string) (string, error)

	// Authorize used for Clients authorization.
	Authorize(ctx context.Context, req AuthzReq) (string, error)

	// Delete deletes client with given ID.
	Delete(ctx context.Context, session authn.Session, id string) error
}

// Cache contains client caching interface.
//
//go:generate mockery --name Cache --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	// Save stores pair client secret, client id.
	Save(ctx context.Context, clientSecret, clientID string) error

	// ID returns client ID for given client secret.
	ID(ctx context.Context, clientSecret string) (string, error)

	// Removes client from cache.
	Remove(ctx context.Context, clientID string) error
}

// Client Struct represents a client.

type Client struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Domain      string      `json:"domain_id,omitempty"`
	Credentials Credentials `json:"credentials,omitempty"`
	Metadata    Metadata    `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	UpdatedBy   string      `json:"updated_by,omitempty"`
	Status      Status      `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Permissions []string    `json:"permissions,omitempty"`
	Identity    string      `json:"identity,omitempty"`
}

// ClientsPage contains page related metadata as well as list.
type ClientsPage struct {
	Page
	Clients []Client
}

// MembersPage contains page related metadata as well as list of members that
// belong to this page.

type MembersPage struct {
	Page
	Members []Client
}

// Page contains the page metadata that helps navigation.

type Page struct {
	Total      uint64   `json:"total"`
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Name       string   `json:"name,omitempty"`
	Id         string   `json:"id,omitempty"`
	Order      string   `json:"order,omitempty"`
	Dir        string   `json:"dir,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Domain     string   `json:"domain,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	Permission string   `json:"permission,omitempty"`
	Status     Status   `json:"status,omitempty"`
	IDs        []string `json:"ids,omitempty"`
	Identity   string   `json:"identity,omitempty"`
	ListPerms  bool     `json:"-"`
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

// Credentials represent client credentials: its
// "identity" which can be a username, email, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}
