//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package ws_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/mocks"
	broker "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"

	"github.com/mainflux/mainflux"
)

const (
	chanID   = "1"
	pubID    = "1"
	protocol = "ws"
)

var msg = mainflux.RawMessage{
	Channel:   chanID,
	Publisher: pubID,
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(channel *ws.Channel) ws.Service {
	subs := map[string]*ws.Channel{chanID: channel}
	pubsub := mocks.NewService(subs, broker.ErrInvalidMsg)
	return ws.New(pubsub)
}

func TestPublish(t *testing.T) {
	channel := ws.NewChannel()
	svc := newService(channel)

	cases := []struct {
		desc string
		msg  mainflux.RawMessage
		err  error
	}{
		{
			desc: "publish valid message",
			msg:  msg,
			err:  nil,
		},
		{
			desc: "publish empty message",
			msg:  mainflux.RawMessage{},
			err:  ws.ErrFailedMessagePublish,
		},
	}

	for _, tc := range cases {
		// Check if message was sent.
		go func(desc string, tcMsg mainflux.RawMessage) {
			receivedMsg := <-channel.Messages
			assert.Equal(t, tcMsg, receivedMsg, fmt.Sprintf("%s: expected %v got %v\n", desc, tcMsg, receivedMsg))
		}(tc.desc, tc.msg)

		// Check if publish succeeded.
		err := svc.Publish(tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	channel := ws.NewChannel()
	svc := newService(channel)

	cases := []struct {
		desc    string
		chanID  string
		channel *ws.Channel
		err     error
	}{
		{
			desc:    "subscription to valid channel",
			chanID:  chanID,
			channel: channel,
			err:     nil,
		},
		{
			desc:    "subscription to channel that should fail",
			chanID:  "0",
			channel: channel,
			err:     ws.ErrFailedSubscription,
		},
	}

	for _, tc := range cases {
		err := svc.Subscribe(tc.chanID, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSend(t *testing.T) {
	channel := ws.NewChannel()
	go func(channel *ws.Channel) {
		receivedMsg := <-channel.Messages
		assert.Equal(t, msg, receivedMsg, fmt.Sprintf("send message to channel: expected %v got %v\n", msg, receivedMsg))
	}(channel)

	channel.Send(msg)
}

func TestClose(t *testing.T) {
	channel := ws.NewChannel()
	go func() {
		closed := <-channel.Closed
		assert.True(t, closed, "channel closed stayed open")
	}()
	channel.Close()
}
