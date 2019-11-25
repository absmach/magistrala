// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

const (
	keyProtocol   = "opcua"
	keyIdentifier = "identifier"
	keyNamespace  = "namespace"

	group  = "mainflux.opcua"
	stream = "mainflux.things"

	thingPrefix = "thing."
	thingCreate = thingPrefix + "create"
	thingUpdate = thingPrefix + "update"
	thingRemove = thingPrefix + "remove"

	channelPrefix = "channel."
	channelCreate = channelPrefix + "create"
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"

	exists = "BUSYGROUP Consumer Group name already exists"
)

var (
	errMetadataType = errors.New("metadatada is not of type opcua")

	errMetadataFormat = errors.New("malformed metadata")

	errMetadataNamespace = errors.New("Node Namespace not found in channel metadatada")

	errMetadataIdentifier = errors.New("Node Identifier not found in thing metadatada")
)

var _ opcua.EventStore = (*eventStore)(nil)

type eventStore struct {
	svc      opcua.Service
	client   *redis.Client
	consumer string
	logger   logger.Logger
}

// NewEventStore returns new event store instance.
func NewEventStore(svc opcua.Service, client *redis.Client, consumer string, log logger.Logger) opcua.EventStore {
	return eventStore{
		svc:      svc,
		client:   client,
		consumer: consumer,
		logger:   log,
	}
}

func (es eventStore) Subscribe(subject string) error {
	err := es.client.XGroupCreateMkStream(stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	for {
		streams, err := es.client.XReadGroup(&redis.XReadGroupArgs{
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
			case thingCreate:
				cte, err := decodeCreateThing(event)
				if err != nil {
					break
				}
				err = es.handleCreateThing(cte)
			case thingUpdate:
				ute, err := decodeCreateThing(event)
				if err != nil {
					break
				}
				err = es.handleCreateThing(ute)
			case thingRemove:
				rte := decodeRemoveThing(event)
				err = es.handleRemoveThing(rte)
			case channelCreate:
				cce, err := decodeCreateChannel(event)
				if err != nil {
					break
				}
				err = es.handleCreateChannel(cce)
			case channelUpdate:
				uce, err := decodeCreateChannel(event)
				if err != nil {
					break
				}
				err = es.handleCreateChannel(uce)
			case channelRemove:
				rce := decodeRemoveChannel(event)
				err = es.handleRemoveChannel(rce)
			}
			if err != nil && err != errMetadataType {
				es.logger.Warn(fmt.Sprintf("Failed to handle event sourcing: %s", err.Error()))
				break
			}
			es.client.XAck(stream, group, msg.ID)
		}
	}
}

func decodeCreateThing(event map[string]interface{}) (createThingEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createThingEvent{}, err
	}

	cte := createThingEvent{
		id: read(event, "id", ""),
	}

	metadataOpcua, ok := metadata[keyProtocol]
	if !ok {
		return createThingEvent{}, errMetadataType
	}

	metadataVal, ok := metadataOpcua.(map[string]interface{})
	if !ok {
		return createThingEvent{}, errMetadataFormat
	}

	val, ok := metadataVal[keyIdentifier].(string)
	if !ok {
		return createThingEvent{}, errMetadataIdentifier
	}

	cte.opcuaNodeIdentifier = val
	return cte, nil
}

func decodeRemoveThing(event map[string]interface{}) removeThingEvent {
	return removeThingEvent{
		id: read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) (createChannelEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createChannelEvent{}, err
	}

	cce := createChannelEvent{
		id: read(event, "id", ""),
	}

	metadataOpcua, ok := metadata[keyProtocol]
	if !ok {
		return createChannelEvent{}, errMetadataType
	}

	metadataVal, ok := metadataOpcua.(map[string]interface{})
	if !ok {
		return createChannelEvent{}, errMetadataFormat
	}

	val, ok := metadataVal[keyNamespace].(string)
	if !ok {
		return createChannelEvent{}, errMetadataNamespace
	}

	cce.opcuaNodeNamespace = val
	return cce, nil
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func (es eventStore) handleCreateThing(cte createThingEvent) error {
	return es.svc.CreateThing(cte.id, cte.opcuaNodeIdentifier)
}

func (es eventStore) handleRemoveThing(rte removeThingEvent) error {
	return es.svc.RemoveThing(rte.id)
}

func (es eventStore) handleCreateChannel(cce createChannelEvent) error {
	return es.svc.CreateChannel(cce.id, cce.opcuaNodeNamespace)
}

func (es eventStore) handleRemoveChannel(rce removeChannelEvent) error {
	return es.svc.RemoveChannel(rce.id)
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
