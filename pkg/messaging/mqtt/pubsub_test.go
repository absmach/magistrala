// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	mqttpubsub "github.com/absmach/magistrala/pkg/messaging/mqtt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const (
	topic        = "topic"
	chansPrefix  = "channels"
	channel      = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
	subtopic     = "engine"
	tokenTimeout = 100 * time.Millisecond
)

var data = []byte("payload")

// ErrFailedHandleMessage indicates that the message couldn't be handled.
var errFailedHandleMessage = errors.New("failed to handle magistrala message")

func TestPublisher(t *testing.T) {
	msgChan := make(chan []byte)

	// Subscribing with topic, and with subtopic, so that we can publish messages.
	client, err := newClient(address, "clientID1", brokerTimeout)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	token := client.Subscribe(topic, qos, func(_ mqtt.Client, m mqtt.Message) {
		msgChan <- m.Payload()
	})
	if ok := token.WaitTimeout(tokenTimeout); !ok {
		assert.Fail(t, fmt.Sprintf("failed to subscribe to topic %s", topic))
	}
	assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

	token = client.Subscribe(fmt.Sprintf("%s.%s", topic, subtopic), qos, func(_ mqtt.Client, m mqtt.Message) {
		msgChan <- m.Payload()
	})
	if ok := token.WaitTimeout(tokenTimeout); !ok {
		assert.Fail(t, fmt.Sprintf("failed to subscribe to topic %s", fmt.Sprintf("%s.%s", topic, subtopic)))
	}
	assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

	t.Cleanup(func() {
		token := client.Unsubscribe(topic, fmt.Sprintf("%s.%s", topic, subtopic))
		token.WaitTimeout(tokenTimeout)
		assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

		client.Disconnect(100)
	})

	// Test publish with an empty topic.
	err = pubsub.Publish(context.TODO(), "", &messaging.Message{Payload: data})
	assert.Equal(t, err, mqttpubsub.ErrEmptyTopic, fmt.Sprintf("Publish with empty topic: expected: %s, got: %s", mqttpubsub.ErrEmptyTopic, err))

	cases := []struct {
		desc     string
		channel  string
		subtopic string
		payload  []byte
	}{
		{
			desc:    "publish message with nil payload",
			payload: nil,
		},
		{
			desc:    "publish message with string payload",
			payload: data,
		},
		{
			desc:    "publish message with channel",
			payload: data,
			channel: channel,
		},
		{
			desc:     "publish message with subtopic",
			payload:  data,
			subtopic: subtopic,
		},
		{
			desc:     "publish message with channel and subtopic",
			payload:  data,
			channel:  channel,
			subtopic: subtopic,
		},
	}
	for _, tc := range cases {
		expectedMsg := messaging.Message{
			Publisher: "clientID11",
			Channel:   tc.channel,
			Subtopic:  tc.subtopic,
			Payload:   tc.payload,
		}

		err := pubsub.Publish(context.TODO(), topic, &expectedMsg)
		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s\n", tc.desc, err))

		data, err := proto.Marshal(&expectedMsg)
		assert.Nil(t, err, fmt.Sprintf("%s: failed to serialize protobuf error: %s\n", tc.desc, err))

		receivedMsg := <-msgChan
		if tc.payload != nil {
			assert.Equal(t, expectedMsg.GetPayload(), receivedMsg, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, data, receivedMsg))
		}
	}
}

func TestSubscribe(t *testing.T) {
	msgChan := make(chan *messaging.Message)

	// Creating client to Publish messages to subscribed topic.
	client, err := newClient(address, "magistrala", brokerTimeout)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	t.Cleanup(func() {
		client.Unsubscribe()
		client.Disconnect(100)
	})

	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1", msgChan},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: "clientid2",
			err:      nil,
			handler:  handler{false, "clientid2", msgChan},
		},
		{
			desc:     "Subscribe to an already subscribed topic with an ID",
			topic:    topic,
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1", msgChan},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1", msgChan},
		},
		{
			desc:     "Subscribe to an already subscribed topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1", msgChan},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: "clientid1",
			err:      mqttpubsub.ErrEmptyTopic,
			handler:  handler{false, "clientid1", msgChan},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      mqttpubsub.ErrEmptyID,
			handler:  handler{false, "", msgChan},
		},
	}
	for _, tc := range cases {
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   tc.topic,
			Handler: tc.handler,
		}
		err = pubsub.Subscribe(context.TODO(), subCfg)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, err, tc.err))

		if tc.err == nil {
			expectedMsg := messaging.Message{
				Publisher: "clientID1",
				Channel:   channel,
				Subtopic:  subtopic,
				Payload:   data,
			}
			data, err := proto.Marshal(&expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("%s: failed to serialize protobuf error: %s\n", tc.desc, err))

			token := client.Publish(tc.topic, qos, false, data)
			token.WaitTimeout(tokenTimeout)
			assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg.Channel, receivedMsg.Channel, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Created, receivedMsg.Created, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Protocol, receivedMsg.Protocol, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Publisher, receivedMsg.Publisher, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Subtopic, receivedMsg.Subtopic, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		}
	}
}

func TestPubSub(t *testing.T) {
	msgChan := make(chan *messaging.Message)

	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: "clientid7",
			err:      nil,
			handler:  handler{false, "clientid7", msgChan},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: "clientid8",
			err:      nil,
			handler:  handler{false, "clientid8", msgChan},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: "clientid7",
			err:      nil,
			handler:  handler{false, "clientid7", msgChan},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: "clientid7",
			err:      mqttpubsub.ErrEmptyTopic,
			handler:  handler{false, "clientid7", msgChan},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      mqttpubsub.ErrEmptyID,
			handler:  handler{false, "", msgChan},
		},
	}
	for _, tc := range cases {
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   tc.topic,
			Handler: tc.handler,
		}
		err := pubsub.Subscribe(context.TODO(), subCfg)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, err, tc.err))

		if tc.err == nil {
			// Use pubsub to subscribe to a topic, and then publish messages to that topic.
			expectedMsg := messaging.Message{
				Publisher: "clientID",
				Channel:   channel,
				Subtopic:  subtopic,
				Payload:   data,
			}
			data, err := proto.Marshal(&expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("%s: failed to serialize protobuf error: %s\n", tc.desc, err))

			msg := messaging.Message{
				Payload: data,
			}
			// Publish message, and then receive it on message channel.
			err = pubsub.Publish(context.TODO(), topic, &msg)
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s\n", tc.desc, err))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg.Channel, receivedMsg.Channel, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Created, receivedMsg.Created, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Protocol, receivedMsg.Protocol, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Publisher, receivedMsg.Publisher, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Subtopic, receivedMsg.Subtopic, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	msgChan := make(chan *messaging.Message)

	cases := []struct {
		desc      string
		topic     string
		clientID  string
		err       error
		subscribe bool // True for subscribe and false for unsubscribe.
		handler   messaging.MessageHandler
	}{
		{
			desc:      "Subscribe to a topic with an ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "clientid4",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Subscribe to the same topic with a different ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "clientid9",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientid9", msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with an ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "clientid4",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Unsubscribe from same topic with different ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "clientid9",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientid9", msgChan},
		},
		{
			desc:      "Unsubscribe from a non-existent topic with an ID",
			topic:     "h",
			clientID:  "clientid4",
			err:       mqttpubsub.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with an ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "clientid4",
			err:       mqttpubsub.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Subscribe to a topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", chansPrefix, topic, subtopic),
			clientID:  "clientidd4",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientidd4", msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", chansPrefix, topic, subtopic),
			clientID:  "clientidd4",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientidd4", msgChan},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", chansPrefix, topic, subtopic),
			clientID:  "clientid4",
			err:       mqttpubsub.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Unsubscribe from an empty topic with an ID",
			topic:     "",
			clientID:  "clientid4",
			err:       mqttpubsub.ErrEmptyTopic,
			subscribe: false,
			handler:   handler{false, "clientid4", msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with empty ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic),
			clientID:  "",
			err:       mqttpubsub.ErrEmptyID,
			subscribe: false,
			handler:   handler{false, "", msgChan},
		},
		{
			desc:      "Subscribe to a new topic with an ID",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic+"2"),
			clientID:  "clientid55",
			err:       nil,
			subscribe: true,
			handler:   handler{true, "clientid5", msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with an ID with failing handler",
			topic:     fmt.Sprintf("%s.%s", chansPrefix, topic+"2"),
			clientID:  "clientid55",
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, "clientid5", msgChan},
		},
		{
			desc:      "Subscribe to a new topic with subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", chansPrefix, topic+"2", subtopic),
			clientID:  "clientid55",
			err:       nil,
			subscribe: true,
			handler:   handler{true, "clientid5", msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with subtopic with an ID with failing handler",
			topic:     fmt.Sprintf("%s.%s.%s", chansPrefix, topic+"2", subtopic),
			clientID:  "clientid55",
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, "clientid5", msgChan},
		},
	}
	for _, tc := range cases {
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   tc.topic,
			Handler: tc.handler,
		}
		switch tc.subscribe {
		case true:
			err := pubsub.Subscribe(context.TODO(), subCfg)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		default:
			err := pubsub.Unsubscribe(context.TODO(), tc.clientID, tc.topic)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		}
	}
}

type handler struct {
	fail      bool
	publisher string
	msgChan   chan *messaging.Message
}

func (h handler) Handle(msg *messaging.Message) error {
	if msg.GetPublisher() != h.publisher {
		h.msgChan <- msg
	}
	return nil
}

func (h handler) Cancel() error {
	if h.fail {
		return errFailedHandleMessage
	}
	return nil
}
