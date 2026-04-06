// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/messaging"
)

var _ messaging.Publisher = (*publisherES)(nil)

type publisherES struct {
	ep  events.Publisher
	pub messaging.Publisher
}

func NewPublisherMiddleware(ctx context.Context, pub messaging.Publisher, url string) (messaging.Publisher, error) {
	publisher, err := store.NewPublisher(ctx, url, "msg-es-pub")
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
		domainID:  msg.Domain,
		channelID: msg.Channel,
		clientID:  msg.ClientIdentity(),
		subtopic:  msg.Subtopic,
	}

	return es.ep.Publish(ctx, publishStream, me)
}

func (es *publisherES) Close() error {
	return es.pub.Close()
}
