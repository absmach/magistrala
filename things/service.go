// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/mainflux/mainflux/internal/groups"
	"github.com/mainflux/mainflux/pkg/errors"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/ulid"
)

const things = "things"

var (
	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrCreateUUID indicates error in creating uuid for entity creation
	ErrCreateUUID = errors.New("uuid creation failed")

	// ErrCreateEntity indicates error in creating entity or entities
	ErrCreateEntity = errors.New("create entity failed")

	// ErrUpdateEntity indicates error in updating entity or entities
	ErrUpdateEntity = errors.New("update entity failed")

	// ErrViewEntity indicates error in viewing entity or entities
	ErrViewEntity = errors.New("view entity failed")

	// ErrRemoveEntity indicates error in removing entity
	ErrRemoveEntity = errors.New("remove entity failed")

	// ErrConnect indicates error in adding connection
	ErrConnect = errors.New("add connection failed")

	// ErrDisconnect indicates error in removing connection
	ErrDisconnect = errors.New("remove connection failed")

	// ErrCreateGroup indicates error in creating group.
	ErrCreateGroup = errors.New("failed to create group")

	// ErrGenerateGroupID indicates error in creating group.
	ErrGenerateGroupID = errors.New("failed to generate group id")

	// ErrFailedToRetrieveThings failed to retrieve things.
	ErrFailedToRetrieveThings = errors.New("failed to retrieve group members")
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
	ListThings(ctx context.Context, token string, pm PageMetadata) (Page, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(ctx context.Context, token, channel string, offset, limit uint64, connected bool) (Page, error)

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
	ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error)

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected or not connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(ctx context.Context, token, thing string, offset, limit uint64, connected bool) (ChannelsPage, error)

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

	groups.Service
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Name     string
	Order    string
	Dir      string
	Metadata map[string]interface{}
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         mainflux.AuthNServiceClient
	things       ThingRepository
	channels     ChannelRepository
	groups       groups.Repository
	channelCache ChannelCache
	thingCache   ThingCache
	uuidProvider mainflux.IDProvider
	ulidProvider mainflux.IDProvider
}

// New instantiates the things service implementation.
func New(auth mainflux.AuthNServiceClient, things ThingRepository, channels ChannelRepository, groups groups.Repository, ccache ChannelCache, tcache ThingCache, up mainflux.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		things:       things,
		groups:       groups,
		channels:     channels,
		channelCache: ccache,
		thingCache:   tcache,
		uuidProvider: up,
		ulidProvider: ulid.New(),
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Thing{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	for i := range things {
		things[i].ID, err = ts.uuidProvider.ID()
		if err != nil {
			return []Thing{}, errors.Wrap(ErrCreateUUID, err)
		}

		things[i].Owner = res.GetEmail()

		if things[i].Key == "" {
			things[i].Key, err = ts.uuidProvider.ID()
			if err != nil {
				return []Thing{}, errors.Wrap(ErrCreateUUID, err)
			}
		}
	}

	return ts.things.Save(ctx, things...)
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	thing.Owner = res.GetEmail()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	owner := res.GetEmail()

	return ts.things.UpdateKey(ctx, owner, id, key)
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.things.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListThings(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.things.RetrieveAll(ctx, res.GetEmail(), pm)
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, channel string, offset, limit uint64, connected bool) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.things.RetrieveByChannel(ctx, res.GetEmail(), channel, offset, limit, connected)
}

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	if err := ts.thingCache.Remove(ctx, id); err != nil {
		return err
	}
	return ts.things.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Channel{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	for i := range channels {
		channels[i].ID, err = ts.uuidProvider.ID()
		if err != nil {
			return []Channel{}, errors.Wrap(ErrCreateUUID, err)
		}

		channels[i].Owner = res.GetEmail()
	}

	return ts.channels.Save(ctx, channels...)
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	channel.Owner = res.GetEmail()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.channels.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.channels.RetrieveAll(ctx, res.GetEmail(), pm)
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thing string, offset, limit uint64, connected bool) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.channels.RetrieveByThing(ctx, res.GetEmail(), thing, offset, limit, connected)
}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	if err := ts.channelCache.Remove(ctx, id); err != nil {
		return err
	}

	return ts.channels.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.channels.Connect(ctx, res.GetEmail(), chIDs, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token, chanID, thingID string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	if err := ts.channelCache.Disconnect(ctx, chanID, thingID); err != nil {
		return err
	}

	return ts.channels.Disconnect(ctx, res.GetEmail(), chanID, thingID)
}

func (ts *thingsService) CanAccessByKey(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.hasThing(ctx, chanID, thingKey)
	if err == nil {
		return thingID, nil
	}

	thingID, err = ts.channels.HasThing(ctx, chanID, thingKey)
	if err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, thingKey, thingID); err != nil {
		return "", err
	}
	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return "", err
	}
	return thingID, nil
}

func (ts *thingsService) CanAccessByID(ctx context.Context, chanID, thingID string) error {
	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); connected {
		return nil
	}

	if err := ts.channels.HasThingByID(ctx, chanID, thingID); err != nil {
		return err
	}

	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return err
	}
	return nil
}

func (ts *thingsService) Identify(ctx context.Context, key string) (string, error) {
	id, err := ts.thingCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	id, err = ts.things.RetrieveByKey(ctx, key)
	if err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, key, id); err != nil {
		return "", err
	}
	return id, nil
}

func (ts *thingsService) hasThing(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.thingCache.ID(ctx, thingKey)
	if err != nil {
		return "", err
	}

	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); !connected {
		return "", ErrEntityConnected
	}
	return thingID, nil
}

func (ts *thingsService) CreateGroup(ctx context.Context, token string, g groups.Group) (string, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}

	ulid, err := ts.ulidProvider.ID()
	if err != nil {
		return "", errors.Wrap(ErrGenerateGroupID, err)
	}

	g.ID = ulid
	g.OwnerID = user.GetId()
	if _, err := ts.groups.Save(ctx, g); err != nil {
		return "", err
	}

	return g.ID, nil
}

func (ts *thingsService) ListGroups(ctx context.Context, token string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.GroupPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.RetrieveAll(ctx, level, gm)

}

func (ts *thingsService) ListParents(ctx context.Context, token string, childID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.GroupPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.RetrieveAllParents(ctx, childID, level, gm)
}

func (ts *thingsService) ListChildren(ctx context.Context, token string, parentID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.GroupPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.RetrieveAllChildren(ctx, parentID, level, gm)
}

func (ts *thingsService) ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, gm groups.Metadata) (groups.MemberPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.MemberPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	p, err := ts.groups.Members(ctx, groupID, offset, limit, gm)
	if err != nil {
		return groups.MemberPage{}, errors.Wrap(ErrFailedToRetrieveThings, err)
	}
	mp := groups.MemberPage{
		PageMetadata: groups.PageMetadata{
			Total:  p.Total,
			Offset: p.Offset,
			Limit:  p.Limit,
			Name:   things,
		},
		Members: make([]groups.Member, 0),
	}
	mp.Members = append(mp.Members, p.Members)
	return mp, nil
}

func (ts *thingsService) RemoveGroup(ctx context.Context, token, id string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.Delete(ctx, id)
}

func (ts *thingsService) Unassign(ctx context.Context, token, memberID, groupID string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.Unassign(ctx, memberID, groupID)
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.Group{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ts.groups.Update(ctx, g)
}

func (ts *thingsService) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.Group{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.RetrieveByID(ctx, id)
}

func (ts *thingsService) Assign(ctx context.Context, token, memberID, groupID string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.Assign(ctx, memberID, groupID)
}

func (ts *thingsService) ListMemberships(ctx context.Context, token string, memberID string, offset, limit uint64, gm groups.Metadata) (groups.GroupPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return groups.GroupPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return ts.groups.Memberships(ctx, memberID, offset, limit, gm)
}
