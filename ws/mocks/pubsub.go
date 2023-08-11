// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/ws"
)

var _ messaging.PubSub = (*mockPubSub)(nil)

type MockPubSub interface {
	// Publish publishes message to the channel.
	Publish(context.Context, string, *messaging.Message) error

	// Subscribe subscribes messages from the channel.
	Subscribe(context.Context, string, string, messaging.MessageHandler) error

	// Unsubscribe unsubscribes messages from the channel.
	Unsubscribe(context.Context, string, string) error

	// SetFail sets the fail flag.
	SetFail(bool)

	// SetConn sets the connection.
	SetConn(*websocket.Conn)

	// Close closes the connection.
	Close() error
}

type mockPubSub struct {
	fail bool
	conn *websocket.Conn
}

// NewPubSub returns mock message publisher-subscriber.
func NewPubSub() MockPubSub {
	return &mockPubSub{false, nil}
}

func (pubsub *mockPubSub) Publish(ctx context.Context, s string, msg *messaging.Message) error {
	if pubsub.conn != nil {
		data, err := json.Marshal(msg)
		if err != nil {
			return ws.ErrFailedMessagePublish
		}
		return pubsub.conn.WriteMessage(websocket.BinaryMessage, data)
	}
	if pubsub.fail {
		return ws.ErrFailedMessagePublish
	}
	return nil
}

func (pubsub *mockPubSub) Subscribe(context.Context, string, string, messaging.MessageHandler) error {
	if pubsub.fail {
		return ws.ErrFailedSubscription
	}
	return nil
}

func (pubsub *mockPubSub) Unsubscribe(context.Context, string, string) error {
	if pubsub.fail {
		return ws.ErrFailedUnsubscribe
	}
	return nil
}

func (pubsub *mockPubSub) SetFail(fail bool) {
	pubsub.fail = fail
}

func (pubsub *mockPubSub) SetConn(c *websocket.Conn) {
	pubsub.conn = c
}

func (pubsub *mockPubSub) Close() error {
	return nil
}
