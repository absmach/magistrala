package clients

import "errors"

var (
	// ErrConflict indicates usage of the existing email during account
	// registration.
	ErrConflict = errors.New("email already taken")

	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddClient adds new client to the user identified by the provided key.
	AddClient(string, Client) (string, error)

	// UpdateClient updates the client identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateClient(string, Client) error

	// ViewClient retrieves data about the client identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewClient(string, string) (Client, error)

	// ListClients retrieves data about subset of clients that belongs to the
	// user identified by the provided key.
	ListClients(string, int, int) ([]Client, error)

	// RemoveClient removes the client identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveClient(string, string) error

	// CreateChannel adds new channel to the user identified by the provided key.
	CreateChannel(string, Channel) (string, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(string, Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(string, string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(string, int, int) ([]Channel, error)

	// RemoveChannel removes the client identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(string, string) error

	// Connect adds client to the channel's list of connected clients.
	Connect(string, string, string) error

	// Disconnect removes client from the channel's list of connected
	// clients.
	Disconnect(string, string, string) error

	// CanAccess determines whether the channel can be accessed using the
	// provided key and returns client's id.
	CanAccess(string, string) (string, error)
}
