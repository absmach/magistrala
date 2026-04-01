// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/messaging"
)

const (
	supermqPrefix     = "supermq."
	publishStream     = supermqPrefix + "publish"
	subscribeStream   = supermqPrefix + "subscribe"
	unsubscribeStream = supermqPrefix + "unsubscribe"
)

var _ messaging.PubSub = (*pubsubES)(nil)

type pubsubES struct {
	ep     events.Publisher
	pubsub messaging.PubSub
}

func NewPubSubMiddleware(ctx context.Context, pubsub messaging.PubSub, url string) (messaging.PubSub, error) {
	publisher, err := store.NewPublisher(ctx, url, "msg-es-pub")
	if err != nil {
		return nil, err
	}

	return &pubsubES{
		ep:     publisher,
		pubsub: pubsub,
	}, nil
}

func (es *pubsubES) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if err := es.pubsub.Publish(ctx, topic, msg); err != nil {
		return err
	}

	me := publishEvent{
		domainID:  msg.Domain,
		channelID: msg.Channel,
		clientID:  msg.ClientIdentity(),
		subtopic:  msg.Subtopic,
	}

	return es.ep.Publish(ctx, publishStream, me)
}

func (es *pubsubES) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if err := es.pubsub.Subscribe(ctx, cfg); err != nil {
		return err
	}

	se := subscribeEvent{
		operation:    clientSubscribe,
		subscriberID: cfg.ID,
		clientID:     cfg.ClientID,
		topic:        cfg.Topic,
	}

	return es.ep.Publish(ctx, subscribeStream, se)
}

func (es *pubsubES) Unsubscribe(ctx context.Context, id string, topic string) error {
	if err := es.pubsub.Unsubscribe(ctx, id, topic); err != nil {
		return err
	}

	se := subscribeEvent{
		operation:    clientUnsubscribe,
		subscriberID: id,
		topic:        topic,
	}

	return es.ep.Publish(ctx, unsubscribeStream, se)
}

func (es *pubsubES) Close() error {
	return es.pubsub.Close()
}
