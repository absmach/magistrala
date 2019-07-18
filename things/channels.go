//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

import "context"

// Channel represents a Mainflux "communication group". This group contains the
// things that can exchange messages between eachother.
type Channel struct {
	ID       string
	Owner    string
	Name     string
	Metadata map[string]interface{}
}

// ChannelsPage contains page related metadata as well as list of channels that
// belong to this page.
type ChannelsPage struct {
	PageMetadata
	Channels []Channel
}

// ChannelRepository specifies a channel persistence API.
type ChannelRepository interface {
	// Save persists the channel. Successful operation is indicated by unique
	// identifier accompanied by nil error response. A non-nil error is
	// returned to indicate operation failure.
	Save(context.Context, Channel) (string, error)

	// Update performs an update to the existing channel. A non-nil error is
	// returned to indicate operation failure.
	Update(context.Context, Channel) error

	// RetrieveByID retrieves the channel having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(context.Context, string, string) (Channel, error)

	// RetrieveAll retrieves the subset of channels owned by the specified user.
	RetrieveAll(context.Context, string, uint64, uint64, string) (ChannelsPage, error)

	// RetrieveByThing retrieves the subset of channels owned by the specified
	// user and have specified thing connected to them.
	RetrieveByThing(context.Context, string, string, uint64, uint64) (ChannelsPage, error)

	// Remove removes the channel having the provided identifier, that is owned
	// by the specified user.
	Remove(context.Context, string, string) error

	// Connect adds thing to the channel's list of connected things.
	Connect(context.Context, string, string, string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(context.Context, string, string, string) error

	// HasThing determines whether the thing with the provided access key, is
	// "connected" to the specified channel. If that's the case, it returns
	// thing's ID.
	HasThing(context.Context, string, string) (string, error)

	// HasThingByID determines whether the thing with the provided ID, is
	// "connected" to the specified channel. If that's the case, then
	// returned error will be nil.
	HasThingByID(context.Context, string, string) error
}

// ChannelCache contains channel-thing connection caching interface.
type ChannelCache interface {
	// Connect channel thing connection.
	Connect(context.Context, string, string) error

	// HasThing checks if thing is connected to channel.
	HasThing(context.Context, string, string) bool

	// Disconnects thing from channel.
	Disconnect(context.Context, string, string) error

	// Removes channel from cache.
	Remove(context.Context, string) error
}
