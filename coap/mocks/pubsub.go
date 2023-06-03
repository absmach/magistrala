package mocks

import (
	"context"

	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/pkg/messaging"
)

var _ messaging.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	Publish(context.Context, string, *messaging.Message) error
	Subscribe(context.Context, string, string, messaging.MessageHandler) error
	Unsubscribe(context.Context, string, string) error
	SetFail(bool)
	Close() error
}

type mockPubSub struct {
	fail bool
}

// NewPubSub returns mock message publisher-subscriber
func NewPubSub() MockPubSub {
	return &mockPubSub{false}
}

func (pubsub *mockPubSub) Publish(context.Context, string, *messaging.Message) error {
	if pubsub.fail {
		return coap.ErrFailedMessagePublish
	}
	return nil
}

func (pubsub *mockPubSub) Subscribe(context.Context, string, string, messaging.MessageHandler) error {
	if pubsub.fail {
		return coap.ErrFailedSubscription
	}
	return nil
}

func (pubsub *mockPubSub) Unsubscribe(context.Context, string, string) error {
	if pubsub.fail {
		return coap.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) SetFail(fail bool) {
	pubsub.fail = fail
}

func (pubsub mockPubSub) Close() error {
	return nil
}
