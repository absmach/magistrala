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
)

var (
	errMetadataType = errors.New("metadatada is not of type lora")

	errMetadataAppID = errors.New("application ID not found in channel metadatada")

	errMetadataDevEUI = errors.New("device EUI not found in thing metadatada")
)

// EventStore represents event source for things and channels provisioning.
type EventStore interface {
	// Subscribes to geven subject and receives events.
	Subscribe(string)
}

type thingLoraMetadata struct {
	Type   string `json:"type"`
	DevEUI string `json:"devEUI"`
}

type channelLoraMetadata struct {
	Type  string `json:"type"`
	AppID string `json:"appID"`
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

func (es eventStore) Subscribe(subject string) {
	es.client.XGroupCreate(stream, group, "$").Err()
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
				cte := decodeCreateThing(event)
				err = es.handleCreateThing(cte)
			case thingUpdate:
				ute := decodeUpdateThing(event)
				err = es.handleUpdateThing(ute)
			case thingRemove:
				rte := decodeRemoveThing(event)
				err = es.handleRemoveThing(rte)
			case channelCreate:
				cce := decodeCreateChannel(event)
				err = es.handleCreateChannel(cce)
			case channelUpdate:
				uce := decodeUpdateChannel(event)
				err = es.handleUpdateChannel(uce)
			case channelRemove:
				rce := decodeRemoveChannel(event)
				err = es.handleRemoveChannel(rce)
			}
			if err != nil {
				es.logger.Warn(fmt.Sprintf("Failed to handle event sourcing: %s", err.Error()))
				break
			}
			es.client.XAck(stream, group, msg.ID)
		}
	}
}

func decodeCreateThing(event map[string]interface{}) createThingEvent {
	return createThingEvent{
		id:       read(event, "id", ""),
		kind:     read(event, "type", ""),
		metadata: read(event, "metadata", ""),
	}
}

func decodeUpdateThing(event map[string]interface{}) updateThingEvent {
	return updateThingEvent{
		id:       read(event, "id", ""),
		kind:     read(event, "type", ""),
		metadata: read(event, "metadata", ""),
	}
}

func decodeRemoveThing(event map[string]interface{}) removeThingEvent {
	return removeThingEvent{
		id: read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) createChannelEvent {
	return createChannelEvent{
		id:       read(event, "id", ""),
		metadata: read(event, "metadata", ""),
	}
}

func decodeUpdateChannel(event map[string]interface{}) updateChannelEvent {
	return updateChannelEvent{
		id:       read(event, "id", ""),
		metadata: read(event, "metadata", ""),
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func (es eventStore) handleCreateThing(cte createThingEvent) error {
	em := thingLoraMetadata{}
	if err := json.Unmarshal([]byte(cte.metadata), &em); err != nil {
		return err
	}

	if em.Type != protocol {
		return errMetadataType
	}
	if em.DevEUI == "" {
		return errMetadataDevEUI
	}

	return es.svc.CreateThing(cte.id, em.DevEUI)
}

func (es eventStore) handleUpdateThing(ute updateThingEvent) error {
	em := thingLoraMetadata{}
	if err := json.Unmarshal([]byte(ute.metadata), &em); err != nil {
		return err
	}

	if em.Type != protocol {
		return errMetadataType
	}
	if em.DevEUI == "" {
		return errMetadataDevEUI
	}

	return es.svc.CreateThing(ute.id, em.DevEUI)
}

func (es eventStore) handleRemoveThing(rte removeThingEvent) error {
	return es.svc.RemoveThing(rte.id)
}

func (es eventStore) handleCreateChannel(cce createChannelEvent) error {
	cm := channelLoraMetadata{}
	if err := json.Unmarshal([]byte(cce.metadata), &cm); err != nil {
		return err
	}

	if cm.Type != protocol {
		return errMetadataType
	}
	if cm.AppID == "" {
		return errMetadataAppID
	}

	return es.svc.CreateChannel(cce.id, cm.AppID)
}

func (es eventStore) handleUpdateChannel(uce updateChannelEvent) error {
	cm := channelLoraMetadata{}
	if err := json.Unmarshal([]byte(uce.metadata), &cm); err != nil {
		return err
	}

	if cm.Type != protocol {
		return errMetadataType
	}
	if cm.AppID == "" {
		return errMetadataAppID
	}

	return es.svc.UpdateChannel(uce.id, cm.AppID)
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
