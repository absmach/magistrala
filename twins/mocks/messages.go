// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
)

var _ messaging.Publisher = (*mockBroker)(nil)

type mockBroker struct {
	subscriptions map[string]string
}

// NewBroker returns mock message publisher.
func NewBroker(sub map[string]string) messaging.Publisher {
	return &mockBroker{
		subscriptions: sub,
	}
}

func (mb mockBroker) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if len(msg.GetPayload()) == 0 {
		return errors.New("failed to publish")
	}
	return nil
}

func (mb mockBroker) Close() error {
	return nil
}
