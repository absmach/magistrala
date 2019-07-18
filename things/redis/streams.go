//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis

import (
	"context"

	"github.com/go-redis/redis"
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

func (es eventStore) AddThing(ctx context.Context, token string, thing things.Thing) (things.Thing, error) {
	sth, err := es.svc.AddThing(ctx, token, thing)
	if err != nil {
		return sth, err
	}

	event := createThingEvent{
		id:       sth.ID,
		owner:    sth.Owner,
		name:     sth.Name,
		metadata: sth.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(record).Err()

	return sth, err
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
	es.client.XAdd(record).Err()

	return nil
}

// UpdateKey doesn't send event because key shouldn't be sent over stream.
// Maybe we can start publishing this event at some point, without key value
// in order to notify adapters to disconnect connected things after key update.
func (es eventStore) UpdateKey(ctx context.Context, token, id, key string) error {
	return es.svc.UpdateKey(ctx, token, id, key)
}

func (es eventStore) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	return es.svc.ViewThing(ctx, token, id)
}

func (es eventStore) ListThings(ctx context.Context, token string, offset, limit uint64, name string) (things.ThingsPage, error) {
	return es.svc.ListThings(ctx, token, offset, limit, name)
}

func (es eventStore) ListThingsByChannel(ctx context.Context, token, id string, offset, limit uint64) (things.ThingsPage, error) {
	return es.svc.ListThingsByChannel(ctx, token, id, offset, limit)
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
	es.client.XAdd(record).Err()

	return nil
}

func (es eventStore) CreateChannel(ctx context.Context, token string, channel things.Channel) (things.Channel, error) {
	sch, err := es.svc.CreateChannel(ctx, token, channel)
	if err != nil {
		return sch, err
	}

	event := createChannelEvent{
		id:       sch.ID,
		owner:    sch.Owner,
		name:     sch.Name,
		metadata: sch.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(record).Err()

	return sch, err
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
	es.client.XAdd(record).Err()

	return nil
}

func (es eventStore) ViewChannel(ctx context.Context, token, id string) (things.Channel, error) {
	return es.svc.ViewChannel(ctx, token, id)
}

func (es eventStore) ListChannels(ctx context.Context, token string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	return es.svc.ListChannels(ctx, token, offset, limit, name)
}

func (es eventStore) ListChannelsByThing(ctx context.Context, token, id string, offset, limit uint64) (things.ChannelsPage, error) {
	return es.svc.ListChannelsByThing(ctx, token, id, offset, limit)
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
	es.client.XAdd(record).Err()

	return nil
}

func (es eventStore) Connect(ctx context.Context, token, chanID, thingID string) error {
	if err := es.svc.Connect(ctx, token, chanID, thingID); err != nil {
		return err
	}

	event := connectThingEvent{
		chanID:  chanID,
		thingID: thingID,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(record).Err()

	return nil
}

func (es eventStore) Disconnect(ctx context.Context, token, chanID, thingID string) error {
	if err := es.svc.Disconnect(ctx, token, chanID, thingID); err != nil {
		return err
	}

	event := disconnectThingEvent{
		chanID:  chanID,
		thingID: thingID,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(record).Err()

	return nil
}

func (es eventStore) CanAccess(ctx context.Context, chanID string, key string) (string, error) {
	return es.svc.CanAccess(ctx, chanID, key)
}

func (es eventStore) CanAccessByID(ctx context.Context, chanID string, thingID string) error {
	return es.svc.CanAccessByID(ctx, chanID, thingID)
}

func (es eventStore) Identify(ctx context.Context, key string) (string, error) {
	return es.svc.Identify(ctx, key)
}
