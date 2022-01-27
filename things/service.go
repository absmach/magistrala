// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux/pkg/errors"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/ulid"
)

const (
	usersObjectKey    = "users"
	authoritiesObject = "authorities"
	memberRelationKey = "member"
	readRelationKey   = "read"
	writeRelationKey  = "write"
	deleteRelationKey = "delete"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings adds things to the user identified by the provided key.
	CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(ctx context.Context, token string, thing Thing) error

	// ShareThing gives actions associated with the thing to the given user IDs.
	// The requester user identified by the token has to have a "write" relation
	// on the thing in order to share the thing.
	ShareThing(ctx context.Context, token, thingID string, actions, userIDs []string) error

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
	ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(ctx context.Context, token, id string) error

	// CreateChannels adds channels to the user identified by the provided key.
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
	ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(ctx context.Context, token, id string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, token string, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected
	// things.
	Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error

	// CanAccessByKey determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccessByKey(ctx context.Context, chanID, key string) (string, error)

	// CanAccessByID determines whether the channel can be accessed by
	// the given thing and returns error if it cannot.
	CanAccessByID(ctx context.Context, chanID, thingID string) error

	// IsChannelOwner determines whether the channel can be accessed by
	// the given user and returns error if it cannot.
	IsChannelOwner(ctx context.Context, owner, chanID string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID string, pm PageMetadata) (Page, error)
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total             uint64
	Offset            uint64                 `json:"offset,omitempty"`
	Limit             uint64                 `json:"limit,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Order             string                 `json:"order,omitempty"`
	Dir               string                 `json:"dir,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Disconnected      bool                   // Used for connected or disconnected lists
	FetchSharedThings bool                   // Used for identifying fetching either all or shared things.
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         mainflux.AuthServiceClient
	things       ThingRepository
	channels     ChannelRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idProvider   mainflux.IDProvider
	ulidProvider mainflux.IDProvider
}

// New instantiates the things service implementation.
func New(auth mainflux.AuthServiceClient, things ThingRepository, channels ChannelRepository, ccache ChannelCache, tcache ThingCache, idp mainflux.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		things:       things,
		channels:     channels,
		channelCache: ccache,
		thingCache:   tcache,
		idProvider:   idp,
		ulidProvider: ulid.New(),
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Thing{}, err
	}

	if err := ts.authorize(ctx, res.GetId(), usersObjectKey, memberRelationKey); err != nil {
		return []Thing{}, err
	}

	ths := []Thing{}
	for _, thing := range things {
		th, err := ts.createThing(ctx, &thing, res)

		if err != nil {
			return []Thing{}, err
		}
		ths = append(ths, th)
	}

	return ths, nil
}

// createThing saves the Thing and adds identity as an owner(Read, Write, Delete policies) of the Thing.
func (ts *thingsService) createThing(ctx context.Context, thing *Thing, identity *mainflux.UserIdentity) (Thing, error) {

	thing.Owner = identity.GetEmail()

	if thing.ID == "" {
		id, err := ts.idProvider.ID()
		if err != nil {
			return Thing{}, err
		}
		thing.ID = id
	}

	if thing.Key == "" {
		key, err := ts.idProvider.ID()

		if err != nil {
			return Thing{}, err
		}
		thing.Key = key
	}

	ths, err := ts.things.Save(ctx, *thing)
	if err != nil {
		return Thing{}, err
	}
	if len(ths) == 0 {
		return Thing{}, errors.ErrCreateEntity
	}

	ss := fmt.Sprintf("%s:%s#%s", "members", authoritiesObject, memberRelationKey)
	if err := ts.claimOwnership(ctx, ths[0].ID, []string{readRelationKey, writeRelationKey, deleteRelationKey}, []string{identity.GetId(), ss}); err != nil {
		return Thing{}, err
	}

	return ths[0], nil
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), thing.ID, writeRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	thing.Owner = res.GetEmail()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) ShareThing(ctx context.Context, token, thingID string, actions, userIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), thingID, writeRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	return ts.claimOwnership(ctx, thingID, actions, userIDs)
}

func (ts *thingsService) claimOwnership(ctx context.Context, objectID string, actions, userIDs []string) error {
	var errs error
	for _, userID := range userIDs {
		for _, action := range actions {
			apr, err := ts.auth.AddPolicy(ctx, &mainflux.AddPolicyReq{Obj: objectID, Act: action, Sub: userID})
			if err != nil {
				errs = errors.Wrap(fmt.Errorf("cannot claim ownership on object '%s' by user '%s': %s", objectID, userID, err), errs)
			}
			if !apr.GetAuthorized() {
				errs = errors.Wrap(fmt.Errorf("cannot claim ownership on object '%s' by user '%s': unauthorized", objectID, userID), errs)
			}
		}
	}
	return errs
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), id, writeRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	owner := res.GetEmail()

	return ts.things.UpdateKey(ctx, owner, id, key)
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, err
	}

	if err := ts.authorize(ctx, res.GetId(), id, readRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return Thing{}, err
		}
	}

	return ts.things.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListThings(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, err
	}

	subject := res.GetId()
	// If the user is admin, fetch all things from database.
	if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err == nil {
		pm.FetchSharedThings = true
		page, err := ts.things.RetrieveAll(ctx, res.GetEmail(), pm)
		if err != nil {
			return Page{}, err
		}
		return page, err
	}

	// If the user is not admin, check 'shared' parameter from page metadata.
	// If user provides 'shared' key, fetch things from policies. Otherwise,
	// fetch things from the database based on thing's 'owner' field.
	if pm.FetchSharedThings {
		req := &mainflux.ListPoliciesReq{Act: "read", Sub: subject}
		lpr, err := ts.auth.ListPolicies(ctx, req)
		if err != nil {
			return Page{}, err
		}

		var page Page
		for _, thingID := range lpr.Policies {
			page.Things = append(page.Things, Thing{ID: thingID})
		}
		return page, nil
	}

	// By default, fetch things from Things service.
	page, err := ts.things.RetrieveAll(ctx, res.GetEmail(), pm)
	if err != nil {
		return Page{}, err
	}

	return page, nil
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, err
	}

	return ts.things.RetrieveByChannel(ctx, res.GetEmail(), chID, pm)
}

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), id, deleteRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	if err := ts.thingCache.Remove(ctx, id); err != nil {
		return err
	}
	return ts.things.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Channel{}, err
	}

	chs := []Channel{}
	for _, channel := range channels {
		ch, err := ts.createChannel(ctx, &channel, res)
		if err != nil {
			return []Channel{}, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

func (ts *thingsService) createChannel(ctx context.Context, channel *Channel, identity *mainflux.UserIdentity) (Channel, error) {
	if channel.ID == "" {
		chID, err := ts.idProvider.ID()
		if err != nil {
			return Channel{}, err
		}
		channel.ID = chID
	}
	channel.Owner = identity.GetEmail()

	chs, err := ts.channels.Save(ctx, *channel)
	if err != nil {
		return Channel{}, err
	}
	if len(chs) == 0 {
		return Channel{}, errors.ErrCreateEntity
	}

	ss := fmt.Sprintf("%s:%s#%s", "members", authoritiesObject, memberRelationKey)
	if err := ts.claimOwnership(ctx, chs[0].ID, []string{readRelationKey, writeRelationKey, deleteRelationKey}, []string{identity.GetId(), ss}); err != nil {
		return Channel{}, err
	}
	return chs[0], nil
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), channel.ID, writeRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	channel.Owner = res.GetEmail()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, err
	}

	if err := ts.authorize(ctx, res.GetId(), id, readRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return Channel{}, err
		}
	}

	return ts.channels.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, err
	}

	// If the user is admin, fetch all channels from the database.
	if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err == nil {
		pm.FetchSharedThings = true
		page, err := ts.channels.RetrieveAll(ctx, res.GetEmail(), pm)
		if err != nil {
			return ChannelsPage{}, err
		}
		return page, err
	}

	// By default, fetch channels from database based on the owner field.
	return ts.channels.RetrieveAll(ctx, res.GetEmail(), pm)
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, err
	}

	return ts.channels.RetrieveByThing(ctx, res.GetEmail(), thID, pm)
}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, res.GetId(), id, deleteRelationKey); err != nil {
		if err := ts.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err != nil {
			return err
		}
	}

	if err := ts.channelCache.Remove(ctx, id); err != nil {
		return err
	}

	return ts.channels.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	return ts.channels.Connect(ctx, res.GetEmail(), chIDs, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			if err := ts.channelCache.Disconnect(ctx, chID, thID); err != nil {
				return err
			}
		}
	}

	return ts.channels.Disconnect(ctx, res.GetEmail(), chIDs, thIDs)
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

func (ts *thingsService) IsChannelOwner(ctx context.Context, owner, chanID string) error {
	if _, err := ts.channels.RetrieveByID(ctx, owner, chanID); err != nil {
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

func (ts *thingsService) ListMembers(ctx context.Context, token, groupID string, pm PageMetadata) (Page, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Page{}, err
	}

	res, err := ts.members(ctx, token, groupID, "things", pm.Offset, pm.Limit)
	if err != nil {
		return Page{}, nil
	}

	return ts.things.RetrieveByIDs(ctx, res, pm)
}

func (ts *thingsService) members(ctx context.Context, token, groupID, groupType string, limit, offset uint64) ([]string, error) {
	req := mainflux.MembersReq{
		Token:   token,
		GroupID: groupID,
		Offset:  offset,
		Limit:   limit,
		Type:    groupType,
	}

	res, err := ts.auth.Members(ctx, &req)
	if err != nil {
		return nil, nil
	}
	return res.Members, nil
}

func (ts *thingsService) authorize(ctx context.Context, subject, object string, relation string) error {
	req := &mainflux.AuthorizeReq{
		Sub: subject,
		Obj: object,
		Act: relation,
	}
	res, err := ts.auth.Authorize(ctx, req)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return err
	}
	return nil
}
