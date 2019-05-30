//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

// Thing represents a Mainflux thing. Each thing is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Thing struct {
	ID       string
	Owner    string
	Name     string
	Key      string
	Metadata map[string]interface{}
}

// ThingsPage contains page related metadata as well as list of things that
// belong to this page.
type ThingsPage struct {
	PageMetadata
	Things []Thing
}

// Validate returns an error if thing representation is invalid.
func (c *Thing) Validate() error {
	return nil
}

// ThingRepository specifies a thing persistence API.
type ThingRepository interface {
	// Save persists the thing. Successful operation is indicated by non-nil
	// error response.
	Save(Thing) (string, error)

	// Update performs an update to the existing thing. A non-nil error is
	// returned to indicate operation failure.
	Update(Thing) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(string, string, string) error

	// RetrieveByID retrieves the thing having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(string, string) (Thing, error)

	// RetrieveByKey returns thing ID for given thing key.
	RetrieveByKey(string) (string, error)

	// RetrieveAll retrieves the subset of things owned by the specified user.
	RetrieveAll(string, uint64, uint64, string) (ThingsPage, error)

	// RetrieveByChannel retrieves the subset of things owned by the specified
	// user and connected to specified channel.
	RetrieveByChannel(string, string, uint64, uint64) (ThingsPage, error)

	// Remove removes the thing having the provided identifier, that is owned
	// by the specified user.
	Remove(string, string) error
}

// ThingCache contains thing caching interface.
type ThingCache interface {
	// Save stores pair thing key, thing id.
	Save(string, string) error

	// ID returns thing ID for given key.
	ID(string) (string, error)

	// Removes thing from cache.
	Remove(string) error
}
