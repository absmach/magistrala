package clients

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
)

var _ Service = (*clientsService)(nil)

type clientsService struct {
	users    mainflux.UsersServiceClient
	clients  ClientRepository
	channels ChannelRepository
	hasher   Hasher
	idp      IdentityProvider
}

// New instantiates the domain service implementation.
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
		return "", ErrUnauthorizedAccess
	}

	return client, nil
}
