//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis

import (
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

func (es eventStore) AddThing(token string, thing things.Thing) (things.Thing, error) {
	sth, err := es.svc.AddThing(token, thing)
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

func (es eventStore) UpdateThing(token string, thing things.Thing) error {
	if err := es.svc.UpdateThing(token, thing); err != nil {
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
func (es eventStore) UpdateKey(token, id, key string) error {
	return es.svc.UpdateKey(token, id, key)
}

func (es eventStore) ViewThing(token, id string) (things.Thing, error) {
	return es.svc.ViewThing(token, id)
}

func (es eventStore) ListThings(token string, offset, limit uint64, name string) (things.ThingsPage, error) {
	return es.svc.ListThings(token, offset, limit, name)
}

func (es eventStore) ListThingsByChannel(token, id string, offset, limit uint64) (things.ThingsPage, error) {
	return es.svc.ListThingsByChannel(token, id, offset, limit)
}

func (es eventStore) RemoveThing(token, id string) error {
	if err := es.svc.RemoveThing(token, id); err != nil {
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

func (es eventStore) CreateChannel(token string, channel things.Channel) (things.Channel, error) {
	sch, err := es.svc.CreateChannel(token, channel)
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

func (es eventStore) UpdateChannel(token string, channel things.Channel) error {
	if err := es.svc.UpdateChannel(token, channel); err != nil {
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

func (es eventStore) ViewChannel(token, id string) (things.Channel, error) {
	return es.svc.ViewChannel(token, id)
}

func (es eventStore) ListChannels(token string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	return es.svc.ListChannels(token, offset, limit, name)
}

func (es eventStore) ListChannelsByThing(token, id string, offset, limit uint64) (things.ChannelsPage, error) {
	return es.svc.ListChannelsByThing(token, id, offset, limit)
}

func (es eventStore) RemoveChannel(token, id string) error {
	if err := es.svc.RemoveChannel(token, id); err != nil {
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

func (es eventStore) Connect(token, chanID, thingID string) error {
	if err := es.svc.Connect(token, chanID, thingID); err != nil {
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

func (es eventStore) Disconnect(token, chanID, thingID string) error {
	if err := es.svc.Disconnect(token, chanID, thingID); err != nil {
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

func (es eventStore) CanAccess(chanID string, key string) (string, error) {
	return es.svc.CanAccess(chanID, key)
}

func (es eventStore) Identify(key string) (string, error) {
	return es.svc.Identify(key)
}
