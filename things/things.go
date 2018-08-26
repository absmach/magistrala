//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

import "strings"

// Thing represents a Mainflux thing. Each thing is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Thing struct {
	ID       uint64 `json:"id"`
	Owner    string `json:"-"`
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Key      string `json:"key"`
	Metadata string `json:"metadata,omitempty"`
}

var thingTypes = map[string]bool{
	"app":    true,
	"device": true,
}

// Validate returns an error if thing representation is invalid.
func (c *Thing) Validate() error {
	if c.Type = strings.ToLower(c.Type); !thingTypes[c.Type] {
		return ErrMalformedEntity
	}

	return nil
}

// ThingRepository specifies a thing persistence API.
type ThingRepository interface {
	// Save persists the thing. Successful operation is indicated by non-nil
	// error response.
	Save(Thing) (uint64, error)

	// Update performs an update to the existing thing. A non-nil error is
	// returned to indicate operation failure.
	Update(Thing) error

	// RetrieveByID retrieves the thing having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(string, uint64) (Thing, error)

	// RetrieveByKey returns thing ID for given thing key.
	RetrieveByKey(string) (uint64, error)

	// RetrieveAll retrieves the subset of things owned by the specified user.
	RetrieveAll(string, int, int) []Thing

	// Remove removes the thing having the provided identifier, that is owned
	// by the specified user.
	Remove(string, uint64) error
}
