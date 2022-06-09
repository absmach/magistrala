// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/logger"
)

const (
	stream = "mainflux.things"
	group  = "mainflux.bootstrap"

	thingPrefix     = "thing."
	thingRemove     = thingPrefix + "remove"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "channel."
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"

	exists = "BUSYGROUP Consumer Group name already exists"
)

// Subscriber represents event source for things and channels provisioning.
type Subscriber interface {
	// Subscribes to given subject and receives events.
	Subscribe(context.Context, string) error
}

type eventStore struct {
	svc      bootstrap.Service
	client   *redis.Client
	consumer string
	logger   logger.Logger
}

// NewEventStore returns new event store instance.
func NewEventStore(svc bootstrap.Service, client *redis.Client, consumer string, log logger.Logger) Subscriber {
	return eventStore{
		svc:      svc,
		client:   client,
		consumer: consumer,
		logger:   log,
	}
}

func (es eventStore) Subscribe(ctx context.Context, subject string) error {
	err := es.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	for {
		streams, err := es.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: es.consumer,
			Streams:  []string{stream, ">"},
			Count:    100,
		}).Result()
		if err != nil || len(streams) == 0 {
			continue
		}

		for _, msg := range streams[0].Messages {
			event := msg.Values

			var err error
			switch event["operation"] {
			case thingRemove:
				rte := decodeRemoveThing(event)
				err = es.svc.RemoveConfigHandler(ctx, rte.id)
			case thingDisconnect:
				dte := decodeDisconnectThing(event)
				err = es.svc.DisconnectThingHandler(ctx, dte.channelID, dte.thingID)
			case channelUpdate:
				uce := decodeUpdateChannel(event)
				err = es.handleUpdateChannel(ctx, uce)
			case channelRemove:
				rce := decodeRemoveChannel(event)
				err = es.svc.RemoveChannelHandler(ctx, rce.id)
			}
			if err != nil {
				es.logger.Warn(fmt.Sprintf("Failed to handle event sourcing: %s", err.Error()))
				break
			}
			es.client.XAck(ctx, stream, group, msg.ID)
		}
	}
}

func decodeRemoveThing(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: read(event, "id", ""),
	}
}

func decodeUpdateChannel(event map[string]interface{}) updateChannelEvent {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		metadata = map[string]interface{}{}
	}

	return updateChannelEvent{
		id:       read(event, "id", ""),
		name:     read(event, "name", ""),
		metadata: metadata,
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: read(event, "id", ""),
	}
}

func decodeDisconnectThing(event map[string]interface{}) disconnectEvent {
	return disconnectEvent{
		channelID: read(event, "chan_id", ""),
		thingID:   read(event, "thing_id", ""),
	}
}

func (es eventStore) handleUpdateChannel(ctx context.Context, uce updateChannelEvent) error {
	channel := bootstrap.Channel{
		ID:       uce.id,
		Name:     uce.name,
		Metadata: uce.metadata,
	}
	return es.svc.UpdateChannelHandler(ctx, channel)
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
