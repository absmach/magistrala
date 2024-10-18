// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package channels

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/roles"
)

// Channel represents a Mainflux "communication group". This group contains the
// things that can exchange messages between each other.
type Channel struct {
	ID          string           `json:"id"`
	Name        string           `json:"name,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Domain      string           `json:"domain_id,omitempty"`
	Metadata    clients.Metadata `json:"metadata,omitempty"`
	CreatedAt   time.Time        `json:"created_at,omitempty"`
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`
	UpdatedBy   string           `json:"updated_by,omitempty"`
	Status      clients.Status   `json:"status,omitempty"`      // 1 for enabled, 0 for disabled
	Permissions []string         `json:"permissions,omitempty"` // 1 for enabled, 0 for disabled
}

type PageMetadata struct {
	Total      uint64           `json:"total"`
	Offset     uint64           `json:"offset"`
	Limit      uint64           `json:"limit"`
	Name       string           `json:"name,omitempty"`
	Id         string           `json:"id,omitempty"`
	Order      string           `json:"order,omitempty"`
	Dir        string           `json:"dir,omitempty"`
	Metadata   clients.Metadata `json:"metadata,omitempty"`
	Domain     string           `json:"domain,omitempty"`
	Tag        string           `json:"tag,omitempty"`
	Permission string           `json:"permission,omitempty"`
	Status     clients.Status   `json:"status,omitempty"`
	IDs        []string         `json:"ids,omitempty"`
	ListPerms  bool             `json:"-"`
	ThingID    string           `json:"-"`
}

// ChannelsPage contains page related metadata as well as list of channels that
// belong to this page.
type Page struct {
	PageMetadata
	Channels []Channel
}

//go:generate mockery --name Service  --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateChannels adds channels to the user identified by the provided key.
	CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error)

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(ctx context.Context, token, id string) (Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(ctx context.Context, token string, channel Channel) (Channel, error)

	// UpdateChannelTags updates the channel's tags.
	UpdateChannelTags(ctx context.Context, token string, channel Channel) (Channel, error)

	EnableChannel(ctx context.Context, token string, id string) (Channel, error)

	DisableChannel(ctx context.Context, token string, id string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(ctx context.Context, token string, pm PageMetadata) (Page, error)

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected or not connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (Page, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(ctx context.Context, token, id string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, token string, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected things.
	Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error

	// SetParentGroup(ctx context.Context, token string, parentGroupID string, id string) error

	// RemoveParentGroup(ctx context.Context, token string, parentGroupID string, id string) error

	roles.Roles
}

// ChannelRepository specifies a channel persistence API.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go  --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// Save persists multiple channels. Channels are saved using a transaction. If one channel
	// fails then none will be saved. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, chs ...Channel) ([]Channel, error)

	// Update performs an update to the existing channel.
	Update(ctx context.Context, c Channel) (Channel, error)

	UpdateTags(ctx context.Context, ch Channel) (Channel, error)

	ChangeStatus(ctx context.Context, channel Channel) (Channel, error)

	// RetrieveByID retrieves the channel having the provided identifier
	RetrieveByID(ctx context.Context, id string) (Channel, error)

	// RetrieveAll retrieves the subset of channels.
	RetrieveAll(ctx context.Context, pm PageMetadata) (Page, error)

	// RetrieveByThing retrieves the subset of channels and have specified thing connected or not connected to them.
	RetrieveByThing(ctx context.Context, thID string, pm PageMetadata) (Page, error)

	// Remove removes the channel having the provided identifier
	Remove(ctx context.Context, ids ...string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected things.
	Disconnect(ctx context.Context, chIDs, thIDs []string) error

	roles.Repository
}
