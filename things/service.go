// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/mainflux/mainflux/errors"

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

	// ErrScanMetadata indicates problem with metadata in db
	ErrScanMetadata = errors.New("failed to scan metadata")

	// ErrCreateThings indicates error in creating Thing
	ErrCreateThings = errors.New("create thing failed")

	// ErrCreateChannels indicates error in creating Channel
	ErrCreateChannels = errors.New("create channel failed")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings adds a list of things to the user identified by the provided key.
	CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(ctx context.Context, token string, thing Thing) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(ctx context.Context, token, id, key string) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(ctx context.Context, token, id string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(ctx context.Context, token string, offset, limit uint64, name string, metadata Metadata) (ThingsPage, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(ctx context.Context, token, channel string, offset, limit uint64) (ThingsPage, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(ctx context.Context, token, id string) error

	// CreateChannels adds a list of channels to the user identified by the provided key.
	CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(ctx context.Context, token string, channel Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(ctx context.Context, token, id string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(ctx context.Context, token string, offset, limit uint64, name string, m Metadata) (ChannelsPage, error)

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(ctx context.Context, token, thing string, offset, limit uint64) (ChannelsPage, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(ctx context.Context, token, id string) error

	// Connect adds things to the channel's list of connected things.
	Connect(ctx context.Context, token string, chIDs, thIDs []string) error

	// Disconnect removes thing from the channel's list of connected
	// things.
	Disconnect(ctx context.Context, token, chanID, thingID string) error

	// CanAccessByKey determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccessByKey(ctx context.Context, chanID, key string) (string, error)

	// CanAccessByID determines whether the channel can be accessed by
	// the given thing and returns error if it cannot.
	CanAccessByID(ctx context.Context, chanID, thingID string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)
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
	auth         mainflux.AuthNServiceClient
	things       ThingRepository
	channels     ChannelRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idp          IdentityProvider
}

// New instantiates the things service implementation.
func New(auth mainflux.AuthNServiceClient, things ThingRepository, channels ChannelRepository, ccache ChannelCache, tcache ThingCache, idp IdentityProvider) Service {
	return &thingsService{
		auth:         auth,
		things:       things,
		channels:     channels,
		channelCache: ccache,
		thingCache:   tcache,
		idp:          idp,
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Thing{}, ErrUnauthorizedAccess
	}

	for i := range things {
		things[i].ID, err = ts.idp.ID()
		if err != nil {
			return []Thing{}, errors.Wrap(ErrCreateThings, err)
		}

		things[i].Owner = res.GetValue()

		if things[i].Key == "" {
			things[i].Key, err = ts.idp.ID()
			if err != nil {
				return []Thing{}, errors.Wrap(ErrCreateThings, err)
			}
		}
	}

	return ts.things.Save(ctx, things...)
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	thing.Owner = res.GetValue()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	owner := res.GetValue()

	return ts.things.UpdateKey(ctx, owner, id, key)

}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, ErrUnauthorizedAccess
	}

	return ts.things.RetrieveByID(ctx, res.GetValue(), id)
}

func (ts *thingsService) ListThings(ctx context.Context, token string, offset, limit uint64, name string, metadata Metadata) (ThingsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ThingsPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	// tp, err := ts.things.RetrieveAll(ctx, res.GetValue(), offset, limit, name, metadata)
	// return tp, errors.Wrap(ErrUnauthorizedAccess, err)
	return ts.things.RetrieveAll(ctx, res.GetValue(), offset, limit, name, metadata)
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, channel string, offset, limit uint64) (ThingsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ThingsPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.things.RetrieveByChannel(ctx, res.GetValue(), channel, offset, limit)
}

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	ts.thingCache.Remove(ctx, id)
	return ts.things.Remove(ctx, res.GetValue(), id)
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Channel{}, ErrUnauthorizedAccess
	}

	for i := range channels {
		channels[i].ID, err = ts.idp.ID()
		if err != nil {
			return []Channel{}, errors.Wrap(ErrCreateChannels, err)
		}

		channels[i].Owner = res.GetValue()
	}

	return ts.channels.Save(ctx, channels...)
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = res.GetValue()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveByID(ctx, res.GetValue(), id)
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, offset, limit uint64, name string, m Metadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveAll(ctx, res.GetValue(), offset, limit, name, m)
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thing string, offset, limit uint64) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, ErrUnauthorizedAccess
	}

	return ts.channels.RetrieveByThing(ctx, res.GetValue(), thing, offset, limit)
}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Remove(ctx, id)
	return ts.channels.Remove(ctx, res.GetValue(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	return ts.channels.Connect(ctx, res.GetValue(), chIDs, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token, chanID, thingID string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	ts.channelCache.Disconnect(ctx, chanID, thingID)
	return ts.channels.Disconnect(ctx, res.GetValue(), chanID, thingID)
}

func (ts *thingsService) CanAccessByKey(ctx context.Context, chanID, key string) (string, error) {
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
