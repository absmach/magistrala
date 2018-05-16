package things

// Channel represents a Mainflux "communication group". This group contains the
// things that can exchange messages between eachother.
type Channel struct {
	ID     string  `json:"id"`
	Owner  string  `json:"-"`
	Name   string  `json:"name,omitempty"`
	Things []Thing `json:"connected,omitempty"`
}

// ChannelRepository specifies a channel persistence API.
type ChannelRepository interface {
	// Save persists the channel. Successful operation is indicated by unique
	// identifier accompanied by nil error response. A non-nil error is
	// returned to indicate operation failure.
	Save(Channel) (string, error)

	// Update performs an update to the existing channel. A non-nil error is
	// returned to indicate operation failure.
	Update(Channel) error

	// One retrieves the channel having the provided identifier, that is owned
	// by the specified user.
	One(string, string) (Channel, error)

	// All retrieves the subset of channels owned by the specified user.
	All(string, int, int) []Channel

	// Remove removes the channel having the provided identifier, that is owned
	// by the specified user.
	Remove(string, string) error

	// Connect adds thing to the channel's list of connected things.
	Connect(string, string, string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(string, string, string) error

	// HasThing determines whether the thing with the provided access key, is
	// "connected" to the specified channel.
	HasThing(string, string) (string, error)
}
