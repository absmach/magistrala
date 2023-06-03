// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/mocks"
	httpmock "github.com/mainflux/mainflux/http/mocks"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/stretchr/testify/assert"
)

const (
	chanID   = "1"
	id       = "1"
	thingKey = "thing_key"
	subTopic = "subtopic"
	protocol = "coap"
	token    = "token"
)

var msg = messaging.Message{
	Channel:   chanID,
	Publisher: id,
	Subtopic:  "",
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService(cc mainflux.ThingsServiceClient) (coap.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return coap.New(cc, pubsub), pubsub
}

func NewThingsClient() mainflux.ThingsServiceClient {
	return httpmock.NewThingsClient(map[string]string{thingKey: chanID})
}

func TestPublish(t *testing.T) {
	thingsClient := NewThingsClient()
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
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish a valid message with invalid thingKey",
			thingKey: "invalid",
			msg:      &msg,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with valid thingKey",
			thingKey: thingKey,
			msg:      &messaging.Message{},
			err:      coap.ErrFailedMessagePublish,
		},
		{
			desc:     "publish an empty message with empty thingKey",
			thingKey: "",
			msg:      &messaging.Message{},
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "publish an empty message with invalid thingKey",
			thingKey: "invalid",
			msg:      &messaging.Message{},
			err:      coap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Publish(context.Background(), tc.thingKey, tc.msg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSubscribe(t *testing.T) {
	thingsClient := NewThingsClient()
	svc, pubsub := newService(thingsClient)
	logger, err := logger.New(os.Stdout, "info")
	if err != nil {
		log.Fatalf(err.Error())
	}

	c := coap.NewClient(nil, nil, logger)

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
			err:      coap.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with invalid chanID and invalid thingKey",
			thingKey: "invalid",
			chanID:   "0",
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
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
			err:      coap.ErrFailedUnsubscribe,
		},
		{
			desc:     "unsubscribe from channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "unsubscribe from channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
		{
			desc:     "unsubscribe from channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      coap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		err := svc.Unsubscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic, token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
