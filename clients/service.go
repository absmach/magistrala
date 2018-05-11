package clients

import (
	"context"
	"errors"
	"time"

	"github.com/mainflux/mainflux"
)

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

	// ErrAccessForbidden indicates valid credentials provided but
	// accessing a protected resource is not able due to permission issue.
	ErrAccessForbidden = errors.New("access forbidden")

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

var _ Service = (*clientsService)(nil)

type clientsService struct {
	users    mainflux.UsersServiceClient
	clients  ClientRepository
	channels ChannelRepository
	hasher   Hasher
	idp      IdentityProvider
}

// New instantiates the clients service implementation.
func New(users mainflux.UsersServiceClient, clients ClientRepository, channels ChannelRepository, hasher Hasher, idp IdentityProvider) Service {
	return &clientsService{
		users:    users,
		clients:  clients,
		channels: channels,
		hasher:   hasher,
		idp:      idp,
	}
}

func (ms *clientsService) AddClient(key string, client Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	client.ID = ms.clients.ID()
	client.Owner = res.GetValue()
	client.Key, _ = ms.idp.PermanentKey(client.ID)

	return client.ID, ms.clients.Save(client)
}

func (ms *clientsService) UpdateClient(key string, client Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	client.Owner = res.GetValue()

	return ms.clients.Update(client)
}

func (ms *clientsService) ViewClient(key, id string) (Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Client{}, ErrUnauthorizedAccess
	}

	return ms.clients.One(res.GetValue(), id)
}

func (ms *clientsService) ListClients(key string, offset, limit int) ([]Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.clients.All(res.GetValue(), offset, limit), nil
}

func (ms *clientsService) RemoveClient(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.clients.Remove(res.GetValue(), id)
}

func (ms *clientsService) CreateChannel(key string, channel Channel) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ms.channels.Save(channel)
}

func (ms *clientsService) UpdateChannel(key string, channel Channel) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ms.channels.Update(channel)
}

func (ms *clientsService) ViewChannel(key, id string) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ms.channels.One(res.GetValue(), id)
}

func (ms *clientsService) ListChannels(key string, offset, limit int) ([]Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.channels.All(res.GetValue(), offset, limit), nil
}

func (ms *clientsService) RemoveChannel(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Remove(res.GetValue(), id)
}

func (ms *clientsService) Connect(key, chanID, clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Connect(res.GetValue(), chanID, clientID)
}

func (ms *clientsService) Disconnect(key, chanID, clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Disconnect(res.GetValue(), chanID, clientID)
}

func (ms *clientsService) CanAccess(key, channel string) (string, error) {
	client, err := ms.idp.Identity(key)
	if err != nil {
		return "", err
	}

	if !ms.channels.HasClient(channel, client) {
		return "", ErrAccessForbidden
	}

	return client, nil
}
