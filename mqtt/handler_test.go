// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt"
	"github.com/mainflux/mainflux/mqtt/mocks"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/things/policies"
	"github.com/mainflux/mproxy/pkg/session"
	"github.com/stretchr/testify/assert"
)

const (
	thingID               = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	thingID1              = "513d02d2-16c1-4f23-98be-9e12f8fee899"
	password              = "password"
	password1             = "password1"
	chanID                = "123e4567-e89b-12d3-a456-000000000001"
	invalidID             = "invalidID"
	clientID              = "clientID"
	clientID1             = "clientID1"
	subtopic              = "testSubtopic"
	invalidChannelIDTopic = "channels/**/messages"
)

var (
	topicMsg            = "channels/%s/messages"
	topic               = fmt.Sprintf(topicMsg, chanID)
	invalidTopic        = "invalidTopic"
	payload             = []byte("[{'n':'test-name', 'v': 1.2}]")
	topics              = []string{topic}
	invalidTopics       = []string{invalidTopic}
	invalidChanIDTopics = []string{fmt.Sprintf(topicMsg, invalidTopic)}
	// Test log messages for cases the handler does not provide a return value.
	logBuffer     = bytes.Buffer{}
	sessionClient = session.Session{
		ID:       clientID,
		Username: thingID,
		Password: []byte(password),
	}
	sessionClientSub = session.Session{
		ID:       clientID1,
		Username: thingID1,
		Password: []byte(password1),
	}
	invalidThingSessionClient = session.Session{
		ID:       clientID,
		Username: invalidID,
		Password: []byte(password),
	}
)

func TestAuthConnect(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		err     error
		session *session.Session
	}{
		{
			desc:    "connect without active session",
			err:     mqtt.ErrClientNotInitialized,
			session: nil,
		},
		{
			desc: "connect without clientID",
			err:  mqtt.ErrMissingClientID,
			session: &session.Session{
				ID:       "",
				Username: thingID,
				Password: []byte(password),
			},
		},
		{
			desc: "connect with invalid password",
			err:  errors.ErrAuthentication,
			session: &session.Session{
				ID:       clientID,
				Username: thingID,
				Password: []byte(""),
			},
		},
		{
			desc:    "connect with valid password and invalid username",
			err:     errors.ErrAuthentication,
			session: &invalidThingSessionClient,
		},
		{
			desc:    "connect with valid username and password",
			err:     nil,
			session: &sessionClient,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		err := handler.AuthConnect(ctx)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAuthPublish(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		session *session.Session
		err     error
		topic   *string
		payload []byte
	}{
		{
			desc:    "publish with an inactive client",
			session: nil,
			err:     mqtt.ErrClientNotInitialized,
			topic:   &topic,
			payload: payload,
		},
		{
			desc:    "publish without topic",
			session: &sessionClient,
			err:     mqtt.ErrMissingTopicPub,
			topic:   nil,
			payload: payload,
		},
		{
			desc:    "publish with malformed topic",
			session: &sessionClient,
			err:     mqtt.ErrMalformedTopic,
			topic:   &invalidTopic,
			payload: payload,
		},
		{
			desc:    "publish with invalid access rights",
			session: &sessionClientSub,
			err:     errors.ErrAuthorization,
			topic:   &topic,
			payload: payload,
		},
		{
			desc:    "publish successfully",
			session: &sessionClient,
			err:     nil,
			topic:   &topic,
			payload: payload,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		err := handler.AuthPublish(ctx, tc.topic, &tc.payload)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAuthSubscribe(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		session *session.Session
		err     error
		topic   *[]string
	}{
		{
			desc:    "subscribe without active session",
			session: nil,
			err:     mqtt.ErrClientNotInitialized,
			topic:   &topics,
		},
		{
			desc:    "subscribe without topics",
			session: &sessionClient,
			err:     mqtt.ErrMissingTopicSub,
			topic:   nil,
		},
		{
			desc:    "subscribe with invalid topics",
			session: &sessionClient,
			err:     mqtt.ErrMalformedTopic,
			topic:   &invalidTopics,
		},
		{
			desc:    "subscribe with invalid channel ID",
			session: &sessionClient,
			err:     errors.ErrAuthorization,
			topic:   &invalidChanIDTopics,
		},
		{
			desc:    "subscribe with active session, valid topics, but invalid access rights",
			session: &sessionClient,
			err:     errors.ErrAuthorization,
			topic:   &topics,
		},
		{
			desc:    "subscribe successfully",
			session: &sessionClientSub,
			err:     nil,
			topic:   &topics,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		err := handler.AuthSubscribe(ctx, tc.topic)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc    string
		session *session.Session
		logMsg  string
	}{
		{
			desc:    "connect without active session",
			session: nil,
			logMsg:  errors.Wrap(mqtt.ErrFailedConnect, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:    "connect with active session",
			session: &sessionClient,
			logMsg:  fmt.Sprintf(mqtt.LogInfoConnected, clientID),
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		handler.Connect(ctx)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestPublish(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	malformedSubtopics := topic + "/" + subtopic + "%"
	wrongCharSubtopics := topic + "/" + subtopic + ">"
	validSubtopic := topic + "/" + subtopic

	cases := []struct {
		desc    string
		session *session.Session
		topic   string
		payload []byte
		logMsg  string
	}{
		{
			desc:    "publish without active session",
			session: nil,
			topic:   topic,
			payload: payload,
			logMsg:  mqtt.ErrClientNotInitialized.Error(),
		},
		{
			desc:    "publish with invalid topic",
			session: &sessionClient,
			topic:   invalidTopic,
			payload: payload,
			logMsg:  fmt.Sprintf(mqtt.LogInfoPublished, clientID, invalidTopic),
		},
		{
			desc:    "publish with invalid channel ID",
			session: &sessionClient,
			topic:   invalidChannelIDTopic,
			payload: payload,
			logMsg:  errors.Wrap(mqtt.ErrFailedPublish, mqtt.ErrMalformedTopic).Error(),
		},
		{
			desc:    "publish with malformed subtopic",
			session: &sessionClient,
			topic:   malformedSubtopics,
			payload: payload,
			logMsg:  mqtt.ErrMalformedSubtopic.Error(),
		},
		{
			desc:    "publish with subtopic containing wrong character",
			session: &sessionClient,
			topic:   wrongCharSubtopics,
			payload: payload,
			logMsg:  mqtt.ErrMalformedSubtopic.Error(),
		},
		{
			desc:    "publish with subtopic",
			session: &sessionClient,
			topic:   validSubtopic,
			payload: payload,
			logMsg:  subtopic,
		},
		{
			desc:    "publish without subtopic",
			session: &sessionClient,
			topic:   topic,
			payload: payload,
			logMsg:  "",
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		handler.Publish(ctx, &tc.topic, &tc.payload)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestSubscribe(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc    string
		session *session.Session
		topic   []string
		logMsg  string
	}{
		{
			desc:    "subscribe without active session",
			session: nil,
			topic:   topics,
			logMsg:  errors.Wrap(mqtt.ErrFailedSubscribe, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:    "subscribe with valid session and topics",
			session: &sessionClient,
			topic:   topics,
			logMsg:  fmt.Sprintf(mqtt.LogInfoSubscribed, clientID, topics[0]),
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		handler.Subscribe(ctx, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestUnsubscribe(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc    string
		session *session.Session
		topic   []string
		logMsg  string
	}{
		{
			desc:    "unsubscribe without active session",
			session: nil,
			topic:   topics,
			logMsg:  errors.Wrap(mqtt.ErrFailedUnsubscribe, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:    "unsubscribe with valid session and topics",
			session: &sessionClient,
			topic:   topics,
			logMsg:  fmt.Sprintf(mqtt.LogInfoUnsubscribed, clientID, topics[0]),
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		handler.Unsubscribe(ctx, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestDisconnect(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc    string
		session *session.Session
		topic   []string
		logMsg  string
	}{
		{
			desc:    "disconnect without active session",
			session: nil,
			topic:   topics,
			logMsg:  errors.Wrap(mqtt.ErrFailedDisconnect, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:    "disconnect with valid session",
			session: &sessionClient,
			topic:   topics,
			logMsg:  mqtt.ErrClientNotInitialized.Error(),
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		handler.Disconnect(ctx)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func newHandler() session.Handler {
	logger, err := logger.New(&logBuffer, "debug")
	if err != nil {
		log.Fatalf("failed to create logger: %s", err)
	}
	k := mocks.Key(&policies.AuthorizeReq{Subject: password, Object: chanID})
	elems := map[string][]string{k: {policies.WriteAction}}
	k = mocks.Key(&policies.AuthorizeReq{Subject: password1, Object: chanID})
	elems[k] = []string{policies.ReadAction}
	authClient := mocks.NewClient(map[string]string{password: thingID, password1: thingID1}, elems)
	eventStore := mocks.NewEventStore()
	return mqtt.NewHandler([]messaging.Publisher{mocks.NewPublisher()}, eventStore, logger, authClient)
}
