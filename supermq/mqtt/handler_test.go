// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/absmach/mgate/pkg/session"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	chmocks "github.com/absmach/supermq/channels/mocks"
	climocks "github.com/absmach/supermq/clients/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/mqtt"
	"github.com/absmach/supermq/mqtt/mocks"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	password              = "password"
	password1             = "password1"
	chanID                = "123e4567-e89b-12d3-a456-000000000001"
	invalidID             = "invalidID"
	invalidValue          = "invalidValue"
	clientID              = "clientID"
	clientID1             = "clientID1"
	subtopic              = "testSubtopic"
	invalidChannelIDTopic = "channels/**/messages"
)

var (
	topicMsg            = "channels/%s/messages"
	topic               = fmt.Sprintf(topicMsg, chanID)
	invalidTopic        = invalidValue
	payload             = []byte("[{'n':'test-name', 'v': 1.2}]")
	topics              = []string{topic}
	invalidTopics       = []string{invalidValue}
	invalidChanIDTopics = []string{fmt.Sprintf(topicMsg, invalidValue)}
	// Test log messages for cases the handler does not provide a return value.
	logBuffer     = bytes.Buffer{}
	sessionClient = session.Session{
		ID:       clientID,
		Username: clientID,
		Password: []byte(password),
	}
	sessionClientSub = session.Session{
		ID:       clientID1,
		Username: clientID1,
		Password: []byte(password1),
	}
	invalidClientSessionClient = session.Session{
		ID:       clientID,
		Username: invalidID,
		Password: []byte(password),
	}
	errInvalidUserId = errors.New("invalid user id")
)

var (
	clients  = new(climocks.ClientsServiceClient)
	channels = new(chmocks.ChannelsServiceClient)
)

func TestAuthConnect(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc     string
		session  *session.Session
		authNRes *grpcClientsV1.AuthnRes
		authNErr error
		err      error
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
				Username: clientID,
				Password: []byte(password),
			},
		},
		{
			desc: "connect with empty password",
			session: &session.Session{
				ID:       clientID,
				Username: clientID,
				Password: []byte(""),
			},
			authNErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "connect with invalid password",
			session: &session.Session{
				ID:       clientID,
				Username: clientID,
				Password: []byte("invalid"),
			},
			authNRes: &grpcClientsV1.AuthnRes{
				Authenticated: false,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:    "connect with valid password and invalid username",
			session: &invalidClientSessionClient,
			authNRes: &grpcClientsV1.AuthnRes{
				Authenticated: true,
				Id:            testsutil.GenerateUUID(t),
			},
			err: errInvalidUserId,
		},
		{
			desc:    "connect with valid username and password",
			err:     nil,
			session: &sessionClient,
			authNRes: &grpcClientsV1.AuthnRes{
				Authenticated: true,
				Id:            clientID,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			password := ""
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
				password = string(tc.session.Password)
			}
			clientsCall := clients.On("Authenticate", mock.Anything, &grpcClientsV1.AuthnReq{ClientSecret: password}).Return(tc.authNRes, tc.authNErr)
			err := handler.AuthConnect(ctx)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			clientsCall.Unset()
		})
	}
}

func TestAuthPublish(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc     string
		session  *session.Session
		err      error
		topic    *string
		payload  []byte
		authZRes *grpcChannelsV1.AuthzRes
		authZErr error
	}{
		{
			desc:     "publish successfully",
			session:  &sessionClient,
			err:      nil,
			topic:    &topic,
			payload:  payload,
			authZRes: &grpcChannelsV1.AuthzRes{Authorized: true},
		},
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
			desc:     "publish with authorization error",
			session:  &sessionClient,
			err:      svcerr.ErrAuthorization,
			topic:    &topic,
			payload:  payload,
			authZRes: &grpcChannelsV1.AuthzRes{Authorized: false},
			authZErr: svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			channelsCall := channels.On("Authorize", mock.Anything, &grpcChannelsV1.AuthzReq{
				ChannelId:  chanID,
				ClientId:   clientID,
				ClientType: policies.ClientType,
				Type:       uint32(connections.Publish),
			}).Return(tc.authZRes, tc.authZErr)
			err := handler.AuthPublish(ctx, tc.topic, &tc.payload)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			channelsCall.Unset()
		})
	}
}

func TestAuthSubscribe(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc      string
		session   *session.Session
		err       error
		topic     *[]string
		channelID string
		authZRes  *grpcChannelsV1.AuthzRes
		authZErr  error
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
			desc:      "subscribe with invalid channel ID",
			session:   &sessionClientSub,
			err:       svcerr.ErrAuthorization,
			topic:     &invalidChanIDTopics,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			channelID: invalidValue,
		},
		{
			desc:      "subscribe successfully",
			session:   &sessionClientSub,
			err:       nil,
			topic:     &topics,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			channelID: chanID,
		},
		{
			desc:      "subscribe with failed authorization",
			session:   &sessionClientSub,
			err:       svcerr.ErrAuthorization,
			topic:     &topics,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			channelID: chanID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			channelsCall := channels.On("Authorize", mock.Anything, &grpcChannelsV1.AuthzReq{
				ChannelId:  tc.channelID,
				ClientId:   clientID1,
				ClientType: policies.ClientType,
				Type:       uint32(connections.Subscribe),
			}).Return(tc.authZRes, tc.authZErr)
			err := handler.AuthSubscribe(ctx, tc.topic)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			channelsCall.Unset()
		})
	}
}

func TestConnect(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc    string
		session *session.Session
		err     error
		logMsg  string
	}{
		{
			desc:    "connect without active session",
			session: nil,
			err:     errors.Wrap(mqtt.ErrFailedConnect, mqtt.ErrClientNotInitialized),
		},
		{
			desc:    "connect with active session",
			session: &sessionClient,
			logMsg:  fmt.Sprintf(mqtt.LogInfoConnected, clientID),
			err:     nil,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		err := handler.Connect(ctx)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
		assert.Equal(t, tc.err, err)
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
		err     error
	}{
		{
			desc:    "publish without active session",
			session: nil,
			topic:   topic,
			payload: payload,
			err:     errors.Wrap(mqtt.ErrFailedPublish, mqtt.ErrClientNotInitialized),
		},
		{
			desc:    "publish with invalid topic",
			session: &sessionClient,
			topic:   invalidTopic,
			payload: payload,
			logMsg:  fmt.Sprintf(mqtt.LogInfoPublished, clientID, invalidTopic),
			err:     errors.Wrap(mqtt.ErrFailedPublish, mqtt.ErrMalformedTopic),
		},
		{
			desc:    "publish with invalid channel ID",
			session: &sessionClient,
			topic:   invalidChannelIDTopic,
			payload: payload,
			err:     errors.Wrap(mqtt.ErrFailedPublish, mqtt.ErrMalformedTopic),
		},
		{
			desc:    "publish with malformed subtopic",
			session: &sessionClient,
			topic:   malformedSubtopics,
			payload: payload,
			err:     errors.Wrap(mqtt.ErrFailedParseSubtopic, mqtt.ErrMalformedSubtopic),
		},
		{
			desc:    "publish with subtopic containing wrong character",
			session: &sessionClient,
			topic:   wrongCharSubtopics,
			payload: payload,
			err:     errors.Wrap(mqtt.ErrFailedParseSubtopic, mqtt.ErrMalformedSubtopic),
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
		err := handler.Publish(ctx, &tc.topic, &tc.payload)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
		assert.Equal(t, tc.err, err)
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
		err     error
	}{
		{
			desc:    "subscribe without active session",
			session: nil,
			topic:   topics,
			err:     errors.Wrap(mqtt.ErrFailedSubscribe, mqtt.ErrClientNotInitialized),
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
		err := handler.Subscribe(ctx, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
		assert.Equal(t, tc.err, err)
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
		err     error
	}{
		{
			desc:    "unsubscribe without active session",
			session: nil,
			topic:   topics,
			err:     errors.Wrap(mqtt.ErrFailedUnsubscribe, mqtt.ErrClientNotInitialized),
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
		err := handler.Unsubscribe(ctx, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
		assert.Equal(t, tc.err, err)
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
		err     error
	}{
		{
			desc:    "disconnect without active session",
			session: nil,
			topic:   topics,
			err:     errors.Wrap(mqtt.ErrFailedDisconnect, mqtt.ErrClientNotInitialized),
		},
		{
			desc:    "disconnect with valid session",
			session: &sessionClient,
			topic:   topics,
			err:     nil,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		err := handler.Disconnect(ctx)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
		assert.Equal(t, tc.err, err)
	}
}

func newHandler() session.Handler {
	logger, err := smqlog.New(&logBuffer, "debug")
	if err != nil {
		log.Fatalf("failed to create logger: %s", err)
	}
	clients = new(climocks.ClientsServiceClient)
	channels = new(chmocks.ChannelsServiceClient)
	return mqtt.NewHandler(mocks.NewPublisher(), logger, clients, channels)
}
