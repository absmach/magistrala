// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bootstrap

// Config represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Mainflux entities.
// MFThing represents corresponding Mainflux Thing ID.
// MFKey is key of corresponding Mainflux Thing.
// MFChannels is a list of Mainflux Channels corresponding Mainflux Thing connects to.
type Config struct {
	MFThing     string
	Owner       string
	Name        string
	ClientCert  string
	ClientKey   string
	CACert      string
	MFKey       string
	MFChannels  []Channel
	ExternalID  string
	ExternalKey string
	Content     string
	State       State
}

// Channel represents Mainflux channel corresponding Mainflux Thing is connected to.
type Channel struct {
	ID       string
	Name     string
	Metadata map[string]interface{}
}

// Filter is used for the search filters.
type Filter struct {
	FullMatch    map[string]string
	PartialMatch map[string]string
}

// ConfigsPage contains page related metadata as well as list of Configs that
// belong to this page.
type ConfigsPage struct {
	Total   uint64
	Offset  uint64
	Limit   uint64
	Configs []Config
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
