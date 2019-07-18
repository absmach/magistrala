//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

import (
	"context"
	"errors"

	"github.com/mainflux/mainflux"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddThing adds new thing to the user identified by the provided key.
	AddThing(context.Context, string, Thing) (Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(context.Context, string, Thing) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(context.Context, string, string, string) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(context.Context, string, string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(context.Context, string, uint64, uint64, string) (ThingsPage, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(context.Context, string, string, uint64, uint64) (ThingsPage, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(context.Context, string, string) error

	// CreateChannel adds new channel to the user identified by the provided key.
	CreateChannel(context.Context, string, Channel) (Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(context.Context, string, Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(context.Context, string, string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(context.Context, string, uint64, uint64, string) (ChannelsPage, error)

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(context.Context, string, string, uint64, uint64) (ChannelsPage, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(context.Context, string, string) error

	// Connect adds thing to the channel's list of connected things.
	Connect(context.Context, string, string, string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(context.Context, string, string, string) error

	// CanAccess determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccess(context.Context, string, string) (string, error)

	// CanAccessByID determines whether the channnel can be accessed by
	// the given thing and returns error if it cannot.
	CanAccessByID(context.Context, string, string) error

	// Identify returns thing ID for given thing key.
	Identify(context.Context, string) (string, error)
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total  uint64
	Offset uint64
	Limit  uint64
	Name   string
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

func (ts *thingsService) AddThing(ctx context.Context, token string, thing Thing) (Thing, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	thing.ID, err = ts.idp.ID()
	if err != nil {
		return Thing{}, err
	}

	thing.Owner = res.GetValue()

	if thing.Key == "" {
		thing.Key, err = ts.idp.ID()
		if err != nil {
			return Thing{}, err
		}
	}

	id, err := ts.things.Save(ctx, thing)
	if err != nil {
		return Thing{}, err
	}

	thing.ID = id
	return thing, nil
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	owner := res.GetValue()

	return ts.things.UpdateKey(ctx, owner, id, key)

}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveByID(ctx, res.GetValue(), id)
}

func (ts *thingsService) ListThings(ctx context.Context, token string, offset, limit uint64, name string) (ThingsPage, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ThingsPage{}, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveAll(ctx, res.GetValue(), offset, limit, name)
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, channel string, offset, limit uint64) (ThingsPage, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ThingsPage{}, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveByChannel(ctx, res.GetValue(), channel, offset, limit)
}

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.thingCache.Remove(ctx, id)
	return ts.things.Remove(ctx, res.GetValue(), id)
}

func (ts *thingsService) CreateChannel(ctx context.Context, token string, channel Channel) (Channel, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	channel.ID, err = ts.idp.ID()
	if err != nil {
		return Channel{}, err
	}

	channel.Owner = res.GetValue()

	id, err := ts.channels.Save(ctx, channel)
	if err != nil {
		return Channel{}, err
	}

	channel.ID = id
	return channel, nil
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveByID(ctx, res.GetValue(), id)
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, offset, limit uint64, name string) (ChannelsPage, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveAll(ctx, res.GetValue(), offset, limit, name)
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thing string, offset, limit uint64) (ChannelsPage, error) {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveByThing(ctx, res.GetValue(), thing, offset, limit)
}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Remove(ctx, id)
	return ts.channels.Remove(ctx, res.GetValue(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token, chanID, thingID string) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Connect(ctx, res.GetValue(), chanID, thingID)
}

func (ts *thingsService) Disconnect(ctx context.Context, token, chanID, thingID string) error {
	res, err := ts.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Disconnect(ctx, chanID, thingID)
	return ts.channels.Disconnect(ctx, res.GetValue(), chanID, thingID)
}

func (ts *thingsService) CanAccess(ctx context.Context, chanID, key string) (string, error) {
	thingID, err := ts.hasThing(ctx, chanID, key)
	if err == nil {
		return thingID, nil
	}

	thingID, err = ts.channels.HasThing(ctx, chanID, key)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	ts.thingCache.Save(ctx, key, thingID)
	ts.channelCache.Connect(ctx, chanID, thingID)
	return thingID, nil
}

func (ts *thingsService) CanAccessByID(ctx context.Context, chanID, thingID string) error {
	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); connected {
		return nil
	}

	if err := ts.channels.HasThingByID(ctx, chanID, thingID); err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Connect(ctx, chanID, thingID)
	return nil
}

func (ts *thingsService) Identify(ctx context.Context, key string) (string, error) {
	id, err := ts.thingCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	id, err = ts.things.RetrieveByKey(ctx, key)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	ts.thingCache.Save(ctx, key, id)
	return id, nil
}

func (ts *thingsService) hasThing(ctx context.Context, chanID, key string) (string, error) {
	thingID, err := ts.thingCache.ID(ctx, key)
	if err != nil {
		return "", err
	}

	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); !connected {
		return "", ErrUnauthorizedAccess
	}

	return thingID, nil
}
