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
	keyType   = "lora"
	keyDevEUI = "dev_eui"
	keyAppID  = "app_id"

	group  = "mainflux.lora"
	stream = "mainflux.things"

	thingPrefix     = "thing."
	thingCreate     = thingPrefix + "create"
	thingUpdate     = thingPrefix + "update"
	thingRemove     = thingPrefix + "remove"
	thingConnect    = thingPrefix + "connect"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "channel."
	channelCreate = channelPrefix + "create"
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"

	exists = "BUSYGROUP Consumer Group name already exists"
)

var (
	errMetadataType = errors.New("field lora is missing in the metadata")

	errMetadataFormat = errors.New("malformed metadata")

	errMetadataAppID = errors.New("application ID not found in channel metadatada")

	errMetadataDevEUI = errors.New("device EUI not found in thing metadatada")
)

// Subscriber represents event source for things and channels provisioning.
type Subscriber interface {
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
func NewEventStore(svc lora.Service, client *redis.Client, consumer string, log logger.Logger) Subscriber {
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
				ute, derr := decodeCreateThing(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleCreateThing(ute)
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
				uce, derr := decodeCreateChannel(event)
				if derr != nil {
					err = derr
					break
				}
				err = es.handleCreateChannel(uce)
			case channelRemove:
				rce := decodeRemoveChannel(event)
				err = es.handleRemoveChannel(rce)
			case thingConnect:
				rce := decodeConnectionThing(event)
				err = es.handleConnectThing(rce)
			case thingDisconnect:
				rce := decodeConnectionThing(event)
				err = es.handleDisconnectThing(rce)
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

	m, ok := metadata[keyType]
	if !ok {
		return createThingEvent{}, errMetadataType
	}

	lm, ok := m.(map[string]interface{})
	if !ok {
		return createThingEvent{}, errMetadataFormat
	}

	val, ok := lm[keyDevEUI].(string)
	if !ok {
		return createThingEvent{}, errMetadataDevEUI
	}

	cte.loraDevEUI = val
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

	m, ok := metadata[keyType]
	if !ok {
		return createChannelEvent{}, errMetadataType
	}

	lm, ok := m.(map[string]interface{})
	if !ok {
		return createChannelEvent{}, errMetadataFormat
	}

	val, ok := lm[keyAppID].(string)
	if !ok {
		return createChannelEvent{}, errMetadataAppID
	}

	cce.loraAppID = val
	return cce, nil
}

func decodeConnectionThing(event map[string]interface{}) connectionThingEvent {
	return connectionThingEvent{
		chanID:  read(event, "chan_id", ""),
		thingID: read(event, "thing_id", ""),
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func (es eventStore) handleCreateThing(cte createThingEvent) error {
	return es.svc.CreateThing(cte.id, cte.loraDevEUI)
}

func (es eventStore) handleRemoveThing(rte removeThingEvent) error {
	return es.svc.RemoveThing(rte.id)
}

func (es eventStore) handleCreateChannel(cce createChannelEvent) error {
	return es.svc.CreateChannel(cce.id, cce.loraAppID)
}

func (es eventStore) handleRemoveChannel(rce removeChannelEvent) error {
	return es.svc.RemoveChannel(rce.id)
}

func (es eventStore) handleConnectThing(rte connectionThingEvent) error {
	return es.svc.ConnectThing(rte.chanID, rte.thingID)
}

func (es eventStore) handleDisconnectThing(rte connectionThingEvent) error {
	return es.svc.DisconnectThing(rte.chanID, rte.thingID)
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
