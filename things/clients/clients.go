// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"

	"github.com/mainflux/mainflux/pkg/clients"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateThings(ctx context.Context, token string, client ...clients.Client) ([]clients.Client, error)

	// ViewClient retrieves client info for a given client ID and an authorized token.
	ViewClient(ctx context.Context, token, id string) (clients.Client, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, token string, pm clients.Page) (clients.ClientsPage, error)

	// ListClientsByGroup retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListClientsByGroup(ctx context.Context, token, groupID string, pm clients.Page) (clients.MembersPage, error)

	// UpdateClient updates the client's name and metadata.
	UpdateClient(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// UpdateClientTags updates the client's tags.
	UpdateClientTags(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// UpdateClientSecret updates the client's secret
	UpdateClientSecret(ctx context.Context, token, id, key string) (clients.Client, error)

	// UpdateClientOwner updates the client's owner.
	UpdateClientOwner(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// EnableClient logically enableds the client identified with the provided ID
	EnableClient(ctx context.Context, token, id string) (clients.Client, error)

	// DisableClient logically disables the client identified with the provided ID
	DisableClient(ctx context.Context, token, id string) (clients.Client, error)

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)
}

// Cache contains thing caching interface.
type Cache interface {
	// Save stores pair thing secret, thing id.
	Save(ctx context.Context, thingSecret, thingID string) error

	// ID returns thing ID for given thing secret.
	ID(ctx context.Context, thingSecret string) (string, error)

	// Removes thing from cache.
	Remove(ctx context.Context, thingID string) error
}
