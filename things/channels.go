//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

// Channel represents a Mainflux "communication group". This group contains the
// things that can exchange messages between eachother.
type Channel struct {
	ID       uint64
	Owner    string
	Name     string
	Things   []Thing
	Metadata string
}

// ChannelRepository specifies a channel persistence API.
type ChannelRepository interface {
	// Save persists the channel. Successful operation is indicated by unique
	// identifier accompanied by nil error response. A non-nil error is
	// returned to indicate operation failure.
	Save(Channel) (uint64, error)

	// Update performs an update to the existing channel. A non-nil error is
	// returned to indicate operation failure.
	Update(Channel) error

	// RetrieveByID retrieves the channel having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(string, uint64) (Channel, error)

	// RetrieveAll retrieves the subset of channels owned by the specified user.
	RetrieveAll(string, uint64, uint64) []Channel

	// Remove removes the channel having the provided identifier, that is owned
	// by the specified user.
	Remove(string, uint64) error

	// Connect adds thing to the channel's list of connected things.
	Connect(string, uint64, uint64) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(string, uint64, uint64) error

	// HasThing determines whether the thing with the provided access key, is
	// "connected" to the specified channel. If that's the case, it returns
	// thing's ID.
	HasThing(uint64, string) (uint64, error)
}

// ChannelCache contains channel-thing connection caching interface.
type ChannelCache interface {
	// Connect channel thing connection.
	Connect(uint64, uint64) error

	// HasThing checks if thing is connected to channel.
	HasThing(uint64, uint64) bool

	// Disconnects thing from channel.
	Disconnect(uint64, uint64) error

	// Removes channel from cache.
	Remove(uint64) error
}
