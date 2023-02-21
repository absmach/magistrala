// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/logger"
)

const (
	stream = "mainflux.things"
	group  = "mainflux.certs"

	thingPrefix = "thing."
	thingRemove = thingPrefix + "remove"

	exists = "BUSYGROUP Consumer Group name already exists"
)

// Subscriber represents event source for things and channels provisioning.
type Subscriber interface {
	// Subscribes to given subject and receives events.
	Subscribe(context.Context, string) error
}

type eventStore struct {
	svc      certs.Service
	client   *redis.Client
	consumer string
	logger   logger.Logger
}

// NewEventStore returns new event store instance.
func NewEventStore(svc certs.Service, client *redis.Client, consumer string, log logger.Logger) Subscriber {
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

			switch event["operation"] {
			case thingRemove:
				// rte := decodeRemoveThing(event)
				// err := make(chan error)
				// go es.svc.EventHandlerDeleteThing(ctx, rte.id, err)
				// for e := range err {
				// 	es.logger.Info(fmt.Sprintf("Error on thing remove event handled , Thing ID %s error : %v", rte.id, e))
				// }
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

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
