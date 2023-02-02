// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	httpmock "github.com/mainflux/mainflux/http/mocks"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	chanID   = "1"
	id       = "1"
	thingKey = "thing_key"
	subTopic = "subtopic"
	protocol = "ws"
)

var msg = messaging.Message{
	Channel:   chanID,
	Publisher: id,
	Subtopic:  "",
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(cc mainflux.ThingsServiceClient) (ws.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return ws.New(cc, pubsub), pubsub
}

func TestPublish(t *testing.T) {
	thingsClient := httpmock.NewThingsClient(map[string]string{thingKey: chanID})
	svc, _ := newService(thingsClient)

	cases := []struct {
		desc     string
		thingKey string
		msg      *messaging.Message
		err      error
	}{
		{
			desc:     "publish a valid message with valid thingKey",
			thingKey: thingKey,
			msg:      &msg,
			err:      nil,
		},
		{
			desc:     "publish a valid message with empty thingKey",
			thingKey: "",
			msg:      &msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish a valid message with invalid thingKey",
			thingKey: "invalid",
			msg:      &msg,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with valid thingKey",
			thingKey: thingKey,
			msg:      &messaging.Message{},
			err:      ws.ErrFailedMessagePublish,
		},
		{
			desc:     "publish an empty message with empty thingKey",
			thingKey: "",
			msg:      &messaging.Message{},
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with invalid thingKey",
			thingKey: "invalid",
			msg:      &messaging.Message{},
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Publish(context.Background(), tc.thingKey, tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	thingsClient := httpmock.NewThingsClient(map[string]string{thingKey: chanID})
	svc, pubsub := newService(thingsClient)

	c := ws.NewClient(nil)

	cases := []struct {
		desc     string
		thingKey string
		chanID   string
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "subscribe to channel with valid thingKey, chanID, subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe again to channel with valid thingKey, chanID, subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "subscribe to channel with subscribe set to fail",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			fail:     true,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with invalid chanID and invalid thingKey",
			thingKey: "invalid",
			chanID:   "0",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnsubscribe(t *testing.T) {
	thingsClient := httpmock.NewThingsClient(map[string]string{thingKey: chanID})
	svc, pubsub := newService(thingsClient)

	cases := []struct {
		desc     string
		thingKey string
		chanID   string
		subtopic string
		fail     bool
		err      error
	}{
		{
			desc:     "unsubscribe from channel with valid thingKey, chanID, subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      nil,
		},
		{
			desc:     "unsubscribe from channel with valid thingKey, chanID, and empty subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: "",
			fail:     false,
			err:      nil,
		},
		{
			desc:     "unsubscribe from channel with unsubscribe set to fail",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			fail:     true,
			err:      ws.ErrFailedUnsubscribe,
		},
		{
			desc:     "unsubscribe from channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "unsubscribe from channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "unsubscribe from channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Unsubscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
