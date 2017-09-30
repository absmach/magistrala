package manager

import "errors"

var (
	// ErrConflict indicates usage of the existing email during account
	// registration.
	ErrConflict error = errors.New("email already taken")

	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity error = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess error = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound error = errors.New("non-existent entity")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned.
	Register(User) error

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(User) (string, error)

	// Identity retrieves Client ID for a given client token
	Identity(string) (string, error)

	// AddClient adds new client to the user identified by the provided key.
	AddClient(string, Client) (string, error)

	// UpdateClient updates the client identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateClient(string, Client) error

	// ViewClient retrieves data about the client identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewClient(string, string) (Client, error)

	// ListClients retrieves data about all clients that belongs to the user
	// identified by the provided key.
	ListClients(string) ([]Client, error)

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

	// ListChannels retrieves data about all clients that belongs to the user
	// identified by the provided key.
	ListChannels(string) ([]Channel, error)

	// RemoveChannel removes the client identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(string, string) error

	// CanAccess determines whether or not the channel can be accessed with the
	// provided key.
	CanAccess(string, string) bool
}
