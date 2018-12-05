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

func (es eventStore) AddThing(key string, thing things.Thing) (things.Thing, error) {
	sth, err := es.svc.AddThing(key, thing)
	if err != nil {
		return sth, err
	}

	event := createThingEvent{
		id:       sth.ID,
		owner:    sth.Owner,
		kind:     sth.Type,
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

func (es eventStore) UpdateThing(key string, thing things.Thing) error {
	if err := es.svc.UpdateThing(key, thing); err != nil {
		return err
	}

	event := updateThingEvent{
		id:       thing.ID,
		kind:     thing.Type,
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

func (es eventStore) ViewThing(key, id string) (things.Thing, error) {
	return es.svc.ViewThing(key, id)
}

func (es eventStore) ListThings(key string, offset, limit uint64) ([]things.Thing, error) {
	return es.svc.ListThings(key, offset, limit)
}

func (es eventStore) RemoveThing(key, id string) error {
	if err := es.svc.RemoveThing(key, id); err != nil {
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

func (es eventStore) CreateChannel(key string, channel things.Channel) (things.Channel, error) {
	sch, err := es.svc.CreateChannel(key, channel)
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

func (es eventStore) UpdateChannel(key string, channel things.Channel) error {
	if err := es.svc.UpdateChannel(key, channel); err != nil {
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

func (es eventStore) ViewChannel(key, id string) (things.Channel, error) {
	return es.svc.ViewChannel(key, id)
}

func (es eventStore) ListChannels(key string, offset, limit uint64) ([]things.Channel, error) {
	return es.svc.ListChannels(key, offset, limit)
}

func (es eventStore) RemoveChannel(key, id string) error {
	if err := es.svc.RemoveChannel(key, id); err != nil {
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

func (es eventStore) Connect(key, chanID, thingID string) error {
	if err := es.svc.Connect(key, chanID, thingID); err != nil {
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

func (es eventStore) Disconnect(key, chanID, thingID string) error {
	if err := es.svc.Disconnect(key, chanID, thingID); err != nil {
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
