//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	AddThing(string, Thing) (Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(string, Thing) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(string, uint64) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(string, int, int) ([]Thing, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(string, uint64) error

	// CreateChannel adds new channel to the user identified by the provided key.
	CreateChannel(string, Channel) (Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(string, Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(string, uint64) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(string, int, int) ([]Channel, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(string, uint64) error

	// Connect adds thing to the channel's list of connected things.
	Connect(string, uint64, uint64) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(string, uint64, uint64) error

	// CanAccess determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccess(uint64, string) (uint64, error)

	// Identify returns thing ID for given thing key.
	Identify(string) (uint64, error)
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	users        mainflux.UsersServiceClient
	things       ThingRepository
	channels     ChannelRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idp          IdentityProvider
}

// New instantiates the things service implementation.
func New(users mainflux.UsersServiceClient, things ThingRepository, channels ChannelRepository, ccache ChannelCache, tcache ThingCache, idp IdentityProvider) Service {
	return &thingsService{
		users:        users,
		things:       things,
		channels:     channels,
		channelCache: ccache,
		thingCache:   tcache,
		idp:          idp,
	}
}

func (ts *thingsService) AddThing(key string, thing Thing) (Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()
	thing.Key = ts.idp.ID()

	id, err := ts.things.Save(thing)
	if err != nil {
		return Thing{}, err
	}

	thing.ID = id
	return thing, nil
}

func (ts *thingsService) UpdateThing(key string, thing Thing) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()

	return ts.things.Update(thing)
}

func (ts *thingsService) ViewThing(key string, id uint64) (Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveByID(res.GetValue(), id)
}

func (ts *thingsService) ListThings(key string, offset, limit int) ([]Thing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveAll(res.GetValue(), offset, limit), nil
}

func (ts *thingsService) RemoveThing(key string, id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.thingCache.Remove(id)
	return ts.things.Remove(res.GetValue(), id)
}

func (ts *thingsService) CreateChannel(key string, channel Channel) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()

	id, err := ts.channels.Save(channel)
	if err != nil {
		return Channel{}, err
	}

	channel.ID = id
	return channel, nil
}

func (ts *thingsService) UpdateChannel(key string, channel Channel) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ts.channels.Update(channel)
}

func (ts *thingsService) ViewChannel(key string, id uint64) (Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveByID(res.GetValue(), id)
}

func (ts *thingsService) ListChannels(key string, offset, limit int) ([]Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveAll(res.GetValue(), offset, limit), nil
}

func (ts *thingsService) RemoveChannel(key string, id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Remove(id)
	return ts.channels.Remove(res.GetValue(), id)
}

func (ts *thingsService) Connect(key string, chanID, thingID uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Connect(res.GetValue(), chanID, thingID)
}

func (ts *thingsService) Disconnect(key string, chanID, thingID uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: key})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Disconnect(chanID, thingID)
	return ts.channels.Disconnect(res.GetValue(), chanID, thingID)
}

func (ts *thingsService) CanAccess(chanID uint64, key string) (uint64, error) {
	thingID, err := ts.hasThing(chanID, key)
	if err == nil {
		return thingID, nil
	}

	thingID, err = ts.channels.HasThing(chanID, key)
	if err != nil {
		return 0, ErrUnauthorizedAccess
	}

	ts.thingCache.Save(key, thingID)
	ts.channelCache.Connect(chanID, thingID)
	return thingID, nil
}

func (ts *thingsService) Identify(key string) (uint64, error) {
	id, err := ts.thingCache.ID(key)
	if err == nil {
		return id, nil
	}

	id, err = ts.things.RetrieveByKey(key)
	if err != nil {
		return 0, ErrUnauthorizedAccess
	}

	ts.thingCache.Save(key, id)
	return id, nil
}

func (ts *thingsService) hasThing(chanID uint64, key string) (uint64, error) {
	thingID, err := ts.thingCache.ID(key)
	if err != nil {
		return 0, err
	}

	if connected := ts.channelCache.HasThing(chanID, thingID); !connected {
		return 0, ErrUnauthorizedAccess
	}

	return thingID, nil
}
