package clients

// Channel represents a Mainflux "communication group". This group contains the
// clients that can exchange messages between eachother.
type Channel struct {
	ID      string   `json:"id"`
	Owner   string   `json:"-"`
	Name    string   `json:"name,omitempty"`
	Clients []Client `json:"connected,omitempty"`
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

	// Connect adds client to the channel's list of connected clients.
	Connect(string, string, string) error

	// Disconnect removes client from the channel's list of connected
	// clients.
	Disconnect(string, string, string) error

	// HasClient determines whether the client with the provided identifier, is
	// "connected" to the specified channel.
	HasClient(string, string) bool
}
