// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
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
	// CreateThings creates new thing. In case of the failed registration, a
	// non-nil error value is returned.
	CreateThings(ctx context.Context, session authn.Session, thing ...Thing) ([]Thing, error)

	// View retrieves thing info for a given thing ID and an authorized token.
	View(ctx context.Context, session authn.Session, id string) (Thing, error)

	// ViewPerms retrieves permissions on the thing id for the given authorized token.
	ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error)

	// ListThings retrieves clients list for a valid auth token.
	ListThings(ctx context.Context, session authn.Session, reqUserID string, pm Page) (ThingsPage, error)

	// ListThingsByGroup retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByGroup(ctx context.Context, session authn.Session, groupID string, pm Page) (MembersPage, error)

	// Update updates the thing's name and metadata.
	Update(ctx context.Context, session authn.Session, thing Thing) (Thing, error)

	// UpdateTags updates the thing's tags.
	UpdateTags(ctx context.Context, session authn.Session, thing Thing) (Thing, error)

	// UpdateSecret updates the thing's secret
	UpdateSecret(ctx context.Context, session authn.Session, id, key string) (Thing, error)

	// Enable logically enableds the thing identified with the provided ID
	Enable(ctx context.Context, session authn.Session, id string) (Thing, error)

	// Disable logically disables the thing identified with the provided ID
	Disable(ctx context.Context, session authn.Session, id string) (Thing, error)

	// Share add share policy to thing id with given relation for given user ids
	Share(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Unshare remove share policy to thing id with given relation for given user ids
	Unshare(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// Authorize used for Things authorization.
	Authorize(ctx context.Context, req AuthzReq) (string, error)

	// Delete deletes thing with given ID.
	Delete(ctx context.Context, session authn.Session, id string) error
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

// Thing Struct represents a thing.

type Thing struct {
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

// ThingsPage contains page related metadata as well as list
type ThingsPage struct {
	Page
	Things []Thing
}

// MembersPage contains page related metadata as well as list of members that
// belong to this page.

type MembersPage struct {
	Page
	Things []Thing
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

// Credentials represent thing credentials: its
// "identity" which can be a username, email, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}
