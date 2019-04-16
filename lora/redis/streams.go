package redis

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"
)

const (
	protocol = "lora"

	group  = "mainflux.lora"
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
	errMetadataType = errors.New("metadatada is not of type lora")

	errMetadataAppID = errors.New("application ID not found in channel metadatada")

	errMetadataDevEUI = errors.New("device EUI not found in thing metadatada")
)

// EventStore represents event source for things and channels provisioning.
type EventStore interface {
	// Subscribes to geven subject and receives events.
	Subscribe(string) error
}

type eventStore struct {
	svc      lora.Service
	client   *redis.Client
	consumer string
	logger   logger.Logger
}

// NewEventStore returns new event store instance.
func NewEventStore(svc lora.Service, client *redis.Client, consumer string, log logger.Logger) EventStore {
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
				cte, derr := decodeCreateThing(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleCreateThing(cte)
			case thingUpdate:
				ute, derr := decodeUpdateThing(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleUpdateThing(ute)
			case thingRemove:
				rte := decodeRemoveThing(event)
				err = es.handleRemoveThing(rte)
			case channelCreate:
				cce, derr := decodeCreateChannel(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleCreateChannel(cce)
			case channelUpdate:
				uce, derr := decodeUpdateChannel(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleUpdateChannel(uce)
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
	var metadata map[string]thingMetadata
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createThingEvent{}, err
	}

	cte := createThingEvent{
		id:   read(event, "id", ""),
		kind: read(event, "type", ""),
	}

	val, ok := metadata["lora"]
	if !ok {
		return createThingEvent{}, errMetadataType
	}
	if val.DevEUI == "" {
		return createThingEvent{}, errMetadataDevEUI
	}

	cte.metadata = val
	return cte, nil
}

func decodeUpdateThing(event map[string]interface{}) (updateThingEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]thingMetadata
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return updateThingEvent{}, errMetadataType
	}

	ute := updateThingEvent{
		id:   read(event, "id", ""),
		kind: read(event, "type", ""),
	}

	val, ok := metadata["lora"]
	if !ok {
		return updateThingEvent{}, errMetadataType
	}
	if val.DevEUI == "" {
		return updateThingEvent{}, errMetadataDevEUI
	}

	ute.metadata = val
	return ute, nil
}

func decodeRemoveThing(event map[string]interface{}) removeThingEvent {
	return removeThingEvent{
		id: read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) (createChannelEvent, error) {
	strmeta := read(event, "metadata", "{}")

	var metadata map[string]channelMetadata
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createChannelEvent{}, err
	}

	cce := createChannelEvent{
		id: read(event, "id", ""),
	}

	val, ok := metadata["lora"]
	if !ok {
		return createChannelEvent{}, errMetadataType
	}
	if val.AppID == "" {
		return createChannelEvent{}, errMetadataAppID
	}

	cce.metadata = val
	return cce, nil
}

func decodeUpdateChannel(event map[string]interface{}) (updateChannelEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]channelMetadata
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return updateChannelEvent{}, err
	}

	uce := updateChannelEvent{
		id: read(event, "id", ""),
	}

	val, ok := metadata["lora"]
	if !ok {
		return updateChannelEvent{}, errMetadataType
	}
	if val.AppID == "" {
		return updateChannelEvent{}, errMetadataAppID
	}

	uce.metadata = val
	return uce, nil
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func (es eventStore) handleCreateThing(cte createThingEvent) error {
	return es.svc.CreateThing(cte.id, cte.metadata.DevEUI)
}

func (es eventStore) handleUpdateThing(ute updateThingEvent) error {
	return es.svc.CreateThing(ute.id, ute.metadata.DevEUI)
}

func (es eventStore) handleRemoveThing(rte removeThingEvent) error {
	return es.svc.RemoveThing(rte.id)
}

func (es eventStore) handleCreateChannel(cce createChannelEvent) error {
	return es.svc.CreateChannel(cce.id, cce.metadata.AppID)
}

func (es eventStore) handleUpdateChannel(uce updateChannelEvent) error {
	return es.svc.UpdateChannel(uce.id, uce.metadata.AppID)
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
