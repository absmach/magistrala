package things

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/logger"
	"github.com/mitchellh/mapstructure"
)

const (
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

	msgEventMessage = "Failed to parse the event message %s : %v"
	msgEventHandler = "Failed to execute the event handler of event %s : %v"
)

type EventHandler interface {
	ThingCreated(ctx context.Context, cte CreateThingEvent) error
	ThingUpdated(ctx context.Context, ute UpdateThingEvent) error
	ThingRemoved(ctx context.Context, rte RemoveThingEvent) error

	ChannelCreated(ctx context.Context, cce CreateChannelEvent) error
	ChannelUpdated(ctx context.Context, uce UpdateChannelEvent) error
	ChannelRemoved(ctx context.Context, rce RemoveChannelEvent) error

	ThingConnected(ctx context.Context, cte ConnectThingEvent) error
	ThingDisconnected(ctx context.Context, dte DisconnectThingEvent) error
}

type Event struct {
	eh       EventHandler
	client   *redis.Client
	consumer string
	logger   logger.Logger
}

func NewEventStore(eh EventHandler, client *redis.Client, consumer string, log logger.Logger) Event {
	return Event{
		eh:       eh,
		client:   client,
		consumer: consumer,
		logger:   log,
	}
}

func (e Event) Subscribe(ctx context.Context, group string) error {
	err := e.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			streams, err := e.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: e.consumer,
				Streams:  []string{stream, ">"},
				Count:    100,
			}).Result()
			if err != nil || len(streams) == 0 {
				continue
			}

			for _, msg := range streams[0].Messages {
				event := msg.Values

				switch event["operation"] {
				case thingCreate:
					cte := CreateThingEvent{}
					if err := decodeEvent(event, &cte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, thingCreate, err))
						break
					}
					if err = e.eh.ThingCreated(ctx, cte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, thingCreate, err))
						break
					}

				case thingUpdate:
					ute := UpdateThingEvent{}
					if err := decodeEvent(event, &ute); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, thingUpdate, err))
						break
					}
					if err = e.eh.ThingUpdated(ctx, ute); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, thingUpdate, err))
						break
					}

				case thingRemove:
					rte := RemoveThingEvent{}
					if err := decodeEvent(event, &rte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, thingRemove, err))
						break
					}
					if err = e.eh.ThingRemoved(ctx, rte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, thingRemove, err))
						break
					}

				case channelCreate:
					cce := CreateChannelEvent{}
					if err := decodeEvent(event, &cce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, channelCreate, err))
						break
					}
					if err = e.eh.ChannelCreated(ctx, cce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, channelCreate, err))
						break
					}

				case channelUpdate:
					uce := UpdateChannelEvent{}
					if err := decodeEvent(event, &uce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, channelUpdate, err))
						break
					}
					if err = e.eh.ChannelUpdated(ctx, uce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, channelUpdate, err))
						break
					}

				case channelRemove:
					rce := RemoveChannelEvent{}
					if err := decodeEvent(event, &rce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, channelRemove, err))
						break
					}
					if err = e.eh.ChannelRemoved(ctx, rce); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, channelRemove, err))
						break
					}

				case thingConnect:
					cte := ConnectThingEvent{}
					if err := decodeEvent(event, &cte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, thingConnect, err))
						break
					}
					if err = e.eh.ThingConnected(ctx, cte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, thingConnect, err))
						break
					}

				case thingDisconnect:
					dte := DisconnectThingEvent{}
					if err := decodeEvent(event, &dte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventMessage, thingConnect, err))
						break
					}
					if err = e.eh.ThingDisconnected(ctx, dte); err != nil {
						e.logger.Error(fmt.Sprintf(msgEventHandler, thingConnect, err))
						break
					}
				}
				e.client.XAck(ctx, stream, group, msg.ID)
			}
		}
	}
}

func decodeEvent[T Type](event map[string]interface{}, obj *T) error {
	return mapstructure.Decode(event, obj)
}
