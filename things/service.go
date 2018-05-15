package things

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

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddThing adds new thing to the user identified by the provided key.
	AddThing(string, Thing) (string, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(string, Thing) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(string, string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(string, int, int) ([]Thing, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(string, string) error

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

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(string, string) error

	// Connect adds thing to the channel's list of connected things.
	Connect(string, string, string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(string, string, string) error

	// CanAccess determines whether the channel can be accessed using the
	// provided key and returns thing's id.
	CanAccess(string, string) (string, error)
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	users    mainflux.UsersServiceClient
	things   ThingRepository
	channels ChannelRepository
	idp      IdentityProvider
}

// New instantiates the things service implementation.
func New(users mainflux.UsersServiceClient, things ThingRepository, channels ChannelRepository, idp IdentityProvider) Service {
	return &thingsService{
		users:    users,
		things:   things,
		channels: channels,
		idp:      idp,
	}
}

func (ms *thingsService) AddThing(key string, thing Thing) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	thing.ID = ms.things.ID()
	thing.Owner = res.GetValue()
	thing.Key, _ = ms.idp.PermanentKey(thing.ID)

	return thing.ID, ms.things.Save(thing)
}

func (ms *thingsService) UpdateThing(key string, thing Thing) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()

	return ms.things.Update(thing)
}

func (ms *thingsService) ViewThing(key, id string) (Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	return ms.things.One(res.GetValue(), id)
}

func (ms *thingsService) ListThings(key string, offset, limit int) ([]Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.things.All(res.GetValue(), offset, limit), nil
}

func (ms *thingsService) RemoveThing(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.things.Remove(res.GetValue(), id)
}

func (ms *thingsService) CreateChannel(key string, channel Channel) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ms.channels.Save(channel)
}

func (ms *thingsService) UpdateChannel(key string, channel Channel) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ms.channels.Update(channel)
}

func (ms *thingsService) ViewChannel(key, id string) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ms.channels.One(res.GetValue(), id)
}

func (ms *thingsService) ListChannels(key string, offset, limit int) ([]Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.channels.All(res.GetValue(), offset, limit), nil
}

func (ms *thingsService) RemoveChannel(key, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Remove(res.GetValue(), id)
}

func (ms *thingsService) Connect(key, chanID, thingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Connect(res.GetValue(), chanID, thingID)
}

func (ms *thingsService) Disconnect(key, chanID, thingID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ms.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Disconnect(res.GetValue(), chanID, thingID)
}

func (ms *thingsService) CanAccess(key, channel string) (string, error) {
	thing, err := ms.idp.Identity(key)
	if err != nil {
		return "", err
	}

	if !ms.channels.HasThing(channel, thing) {
		return "", ErrUnauthorizedAccess
	}

	return thing, nil
}
