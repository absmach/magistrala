// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/broker"
	"github.com/mainflux/mainflux/ws"
	"github.com/nats-io/nats.go"
)

var _ broker.Nats = (*mockBroker)(nil)

type mockBroker struct {
	subscriptions map[string]string
}

// New returns mock message publisher.
func New(sub map[string]string) broker.Nats {
	return &mockBroker{
		subscriptions: sub,
	}
}

func (mb mockBroker) Publish(_ context.Context, _ string, msg broker.Message) error {
	if len(msg.Payload) == 0 {
		return ws.ErrFailedMessagePublish
	}
	return nil
}

func (mb mockBroker) Subscribe(subject string, f func(*nats.Msg)) (*nats.Subscription, error) {
	if _, ok := mb.subscriptions[subject]; !ok {
		return nil, ws.ErrFailedSubscription
	}

	return nil, nil
}

func (mb mockBroker) QueueSubscribe(chanID, subtopic string, f func(*nats.Msg)) (*nats.Subscription, error) {
	return nil, nil
}

func (mb mockBroker) Close() {
}
