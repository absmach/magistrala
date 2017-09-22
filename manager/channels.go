package manager

// Channel represents a Mainflux "communication group". This group contains the
// clients that can exchange messages between eachother.
type Channel struct {
	Owner     string   `json:"-"`
	ID        string   `json:"id"`
	Name      string   `json:"name,omitempty"`
	Connected []string `json:"connected"`
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

	// All retrieves the channels owned by the specified user.
	All(string) []Channel

	// Remove removes the channel having the provided identifier, that is owned
	// by the specified user.
	Remove(string, string) error

	// HasClient determines whether the client with the provided identifier, is
	// "connected" to the specified channel.
	HasClient(string, string) bool
}
