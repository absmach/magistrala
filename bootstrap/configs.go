// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"time"

	"github.com/mainflux/mainflux/pkg/clients"
)

// Config represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Mainflux entities.
// MFThing represents corresponding Mainflux Thing ID.
// MFKey is key of corresponding Mainflux Thing.
// MFChannels is a list of Mainflux Channels corresponding Mainflux Thing connects to.
type Config struct {
	MFThing     string    `json:"mainflux_thing"`
	Owner       string    `json:"owner,omitempty"`
	Name        string    `json:"name,omitempty"`
	ClientCert  string    `json:"client_cert,omitempty"`
	ClientKey   string    `json:"client_key,omitempty"`
	CACert      string    `json:"ca_cert,omitempty"`
	MFKey       string    `json:"mainflux_key"`
	MFChannels  []Channel `json:"mainflux_channels,omitempty"`
	ExternalID  string    `json:"external_id"`
	ExternalKey string    `json:"external_key"`
	Content     string    `json:"content,omitempty"`
	State       State     `json:"state"`
}

// Channel represents Mainflux channel corresponding Mainflux Thing is connected to.
type Channel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Owner       string                 `json:"owner_id"`
	Parent      string                 `json:"parent_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	UpdatedBy   string                 `json:"updated_by,omitempty"`
	Status      clients.Status         `json:"status"`
}

// Filter is used for the search filters.
type Filter struct {
	FullMatch    map[string]string
	PartialMatch map[string]string
}

// ConfigsPage contains page related metadata as well as list of Configs that
// belong to this page.
type ConfigsPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Configs []Config `json:"configs"`
}

// ConfigRepository specifies a Config persistence API.
type ConfigRepository interface {
	// Save persists the Config. Successful operation is indicated by non-nil
	// error response.
	Save(cfg Config, chsConnIDs []string) (string, error)

	// RetrieveByID retrieves the Config having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(owner, id string) (Config, error)

	// RetrieveAll retrieves a subset of Configs that are owned
	// by the specific user, with given filter parameters.
	RetrieveAll(owner string, filter Filter, offset, limit uint64) ConfigsPage

	// RetrieveByExternalID returns Config for given external ID.
	RetrieveByExternalID(externalID string) (Config, error)

	// Update updates an existing Config. A non-nil error is returned
	// to indicate operation failure.
	Update(cfg Config) error

	// UpdateCerts updates an existing Config certificate and owner.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(owner, thingID, clientCert, clientKey, caCert string) error

	// UpdateConnections updates a list of Channels the Config is connected to
	// adding new Channels if needed.
	UpdateConnections(owner, id string, channels []Channel, connections []string) error

	// Remove removes the Config having the provided identifier, that is owned
	// by the specified user.
	Remove(owner, id string) error

	// ChangeState changes of the Config, that is owned by the specific user.
	ChangeState(owner, id string, state State) error

	// ListExisting retrieves those channels from the given list that exist in DB.
	ListExisting(owner string, ids []string) ([]Channel, error)

	// Methods RemoveThing, UpdateChannel, and RemoveChannel are related to
	// event sourcing. That's why these methods surpass ownership check.

	// RemoveThing removes Config of the Thing with the given ID.
	RemoveThing(id string) error

	// UpdateChannel updates channel with the given ID.
	UpdateChannel(c Channel) error

	// RemoveChannel removes channel with the given ID.
	RemoveChannel(id string) error

	// DisconnectHandler changes state of the Config when the corresponding Thing is
	// disconnected from the Channel.
	DisconnectThing(channelID, thingID string) error
}
