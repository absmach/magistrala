// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ messaging.Publisher = (*publisherES)(nil)

type publisherES struct {
	ep  events.Publisher
	pub messaging.Publisher
}

func NewPublisherMiddleware(ctx context.Context, pub messaging.Publisher, url string) (messaging.Publisher, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &publisherES{
		ep:  publisher,
		pub: pub,
	}, nil
}

func (es *publisherES) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if err := es.pub.Publish(ctx, topic, msg); err != nil {
		return err
	}

	me := publishEvent{
		channelID: msg.Channel,
		clientID:  msg.Publisher,
		subtopic:  msg.Subtopic,
	}

	return es.ep.Publish(ctx, me)
}

func (es *publisherES) Close() error {
	return es.pub.Close()
}
