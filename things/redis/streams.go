// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/things"
)

const (
	streamID  = "mainflux.things"
	streamLen = 1000
)

var _ things.Service = (*eventStore)(nil)

type eventStore struct {
	svc    things.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(svc things.Service, client *redis.Client) things.Service {
	return eventStore{
		svc:    svc,
		client: client,
	}
}

func (es eventStore) CreateThings(ctx context.Context, token string, ths ...things.Thing) ([]things.Thing, error) {
	sths, err := es.svc.CreateThings(ctx, token, ths...)
	if err != nil {
		return sths, err
	}

	for _, thing := range sths {
		event := createThingEvent{
			id:       thing.ID,
			owner:    thing.Owner,
			name:     thing.Name,
			metadata: thing.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return sths, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	if err := es.svc.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	event := updateThingEvent{
		id:       thing.ID,
		name:     thing.Name,
		metadata: thing.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

// UpdateKey doesn't send event because key shouldn't be sent over stream.
// Maybe we can start publishing this event at some point, without key value
// in order to notify adapters to disconnect connected things after key update.
func (es eventStore) UpdateKey(ctx context.Context, token, id, key string) error {
	return es.svc.UpdateKey(ctx, token, id, key)
}

func (es eventStore) ShareThing(ctx context.Context, token, thingID string, actions, userIDs []string) error {
	return es.svc.ShareThing(ctx, token, thingID, actions, userIDs)
}

func (es eventStore) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	return es.svc.ViewThing(ctx, token, id)
}

func (es eventStore) ListThings(ctx context.Context, token string, pm things.PageMetadata) (things.Page, error) {
	return es.svc.ListThings(ctx, token, pm)
}

func (es eventStore) ListThingsByChannel(ctx context.Context, token, chID string, pm things.PageMetadata) (things.Page, error) {
	return es.svc.ListThingsByChannel(ctx, token, chID, pm)
}

func (es eventStore) RemoveThing(ctx context.Context, token, id string) error {
	if err := es.svc.RemoveThing(ctx, token, id); err != nil {
		return err
	}

	event := removeThingEvent{
		id: id,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) CreateChannels(ctx context.Context, token string, channels ...things.Channel) ([]things.Channel, error) {
	schs, err := es.svc.CreateChannels(ctx, token, channels...)
	if err != nil {
		return schs, err
	}

	for _, channel := range schs {
		event := createChannelEvent{
			id:       channel.ID,
			owner:    channel.Owner,
			name:     channel.Name,
			metadata: channel.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return schs, nil
}

func (es eventStore) UpdateChannel(ctx context.Context, token string, channel things.Channel) error {
	if err := es.svc.UpdateChannel(ctx, token, channel); err != nil {
		return err
	}

	event := updateChannelEvent{
		id:       channel.ID,
		name:     channel.Name,
		metadata: channel.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) ViewChannel(ctx context.Context, token, id string) (things.Channel, error) {
	return es.svc.ViewChannel(ctx, token, id)
}

func (es eventStore) ListChannels(ctx context.Context, token string, pm things.PageMetadata) (things.ChannelsPage, error) {
	return es.svc.ListChannels(ctx, token, pm)
}

func (es eventStore) ListChannelsByThing(ctx context.Context, token, thID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	return es.svc.ListChannelsByThing(ctx, token, thID, pm)
}

func (es eventStore) RemoveChannel(ctx context.Context, token, id string) error {
	if err := es.svc.RemoveChannel(ctx, token, id); err != nil {
		return err
	}

	event := removeChannelEvent{
		id: id,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	if err := es.svc.Connect(ctx, token, chIDs, thIDs); err != nil {
		return err
	}

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			event := connectThingEvent{
				chanID:  chID,
				thingID: thID,
			}
			record := &redis.XAddArgs{
				Stream:       streamID,
				MaxLenApprox: streamLen,
				Values:       event.Encode(),
			}
			es.client.XAdd(ctx, record).Err()
		}
	}

	return nil
}

func (es eventStore) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	if err := es.svc.Disconnect(ctx, token, chIDs, thIDs); err != nil {
		return err
	}

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			event := disconnectThingEvent{
				chanID:  chID,
				thingID: thID,
			}
			record := &redis.XAddArgs{
				Stream:       streamID,
				MaxLenApprox: streamLen,
				Values:       event.Encode(),
			}
			es.client.XAdd(ctx, record).Err()
		}
	}

	return nil
}

func (es eventStore) CanAccessByKey(ctx context.Context, chanID string, key string) (string, error) {
	return es.svc.CanAccessByKey(ctx, chanID, key)
}

func (es eventStore) CanAccessByID(ctx context.Context, chanID string, thingID string) error {
	return es.svc.CanAccessByID(ctx, chanID, thingID)
}

func (es eventStore) IsChannelOwner(ctx context.Context, owner, chanID string) error {
	return es.svc.IsChannelOwner(ctx, owner, chanID)
}

func (es eventStore) Identify(ctx context.Context, key string) (string, error) {
	return es.svc.Identify(ctx, key)
}

func (es eventStore) ListMembers(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.Page, error) {
	return es.svc.ListMembers(ctx, token, groupID, pm)
}
