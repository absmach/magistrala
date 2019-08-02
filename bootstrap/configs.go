//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	Unknown      bool
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
	Save(Config, []string) (string, error)

	// RetrieveByID retrieves the Config having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(string, string) (Config, error)

	// RetrieveAll retrieves a subset of Configs that are owned
	// by the specific user, with given filter parameters.
	RetrieveAll(string, Filter, uint64, uint64) ConfigsPage

	// RetrieveByExternalID returns Config for given external ID.
	RetrieveByExternalID(string) (Config, error)

	// Update updates an existing Config. A non-nil error is returned
	// to indicate operation failure.
	Update(Config) error

	// UpdateCerts updates an existing Config certificate and key.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(string, string, string, string, string) error

	// UpdateConnections updates a list of Channels the Config is connected to
	// adding new Channels if needed.
	UpdateConnections(string, string, []Channel, []string) error

	// Remove removes the Config having the provided identifier, that is owned
	// by the specified user.
	Remove(string, string) error

	// ChangeState changes of the Config, that is owned by the specific user.
	ChangeState(string, string, State) error

	// SaveUnknown saves Thing which unsuccessfully bootstrapped.
	SaveUnknown(string, string) error

	// RetrieveUnknown returns a subset of unsuccessfully bootstrapped Things.
	RetrieveUnknown(uint64, uint64) ConfigsPage

	// ListExisting retrieves those channels from the given list that exist in DB.
	ListExisting(string, []string) ([]Channel, error)

	// Methods RemoveThing, UpdateChannel, and RemoveChannel are related to
	// event sourcing. That's why these methods surpass ownership check.

	// RemoveThing removes Config of the Thing with the given ID.
	RemoveThing(string) error

	// UpdateChannel updates channel with the given ID.
	UpdateChannel(Channel) error

	// RemoveChannel removes channel with the given ID.
	RemoveChannel(string) error

	// DisconnectHandler changes state of the Config when the corresponding Thing is
	// disconnected from the Channel.
	DisconnectThing(string, string) error
}
