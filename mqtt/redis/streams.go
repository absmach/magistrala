// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

const (
	streamID  = "mainflux.mqtt"
	streamLen = 1000
)

// EventStore is a struct used to store event streams in Redis
type EventStore struct {
	client   *redis.Client
	instance string
}

// NewEventStore returns wrapper around mProxy service that sends
// events to event store.
func NewEventStore(client *redis.Client, instance string) EventStore {
	return EventStore{
		client:   client,
		instance: instance,
	}
}

func (es EventStore) storeEvent(clientID, eventType string) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	event := mqttEvent{
		clientID:  clientID,
		timestamp: timestamp,
		eventType: eventType,
		instance:  es.instance,
	}

	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}

	if err := es.client.XAdd(record).Err(); err != nil {
		return err
	}

	return nil
}

// Connect issues event on MQTT CONNECT
func (es EventStore) Connect(clientID string) error {
	return es.storeEvent(clientID, "connect")
}

// Disconnect issues event on MQTT CONNECT
func (es EventStore) Disconnect(clientID string) error {
	return es.storeEvent(clientID, "disconnect")
}
