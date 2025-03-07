// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	mghttp "github.com/absmach/mgate/pkg/http"
	"github.com/absmach/mgate/pkg/session"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	clmocks "github.com/absmach/supermq/clients/mocks"
	mhttp "github.com/absmach/supermq/http"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	clientID              = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	clientID1             = "513d02d2-16c1-4f23-98be-9e12f8fee899"
	clientKey             = "password"
	chanID                = "123e4567-e89b-12d3-a456-000000000001"
	invalidID             = "invalidID"
	invalidValue          = "invalidValue"
	invalidChannelIDTopic = "channels/**/messages"
)

var (
	topicMsg      = "channels/%s/messages"
	subtopicMsg   = "channels/%s/messages/subtopic"
	topic         = fmt.Sprintf(topicMsg, chanID)
	subtopic      = fmt.Sprintf(subtopicMsg, chanID)
	invalidTopic  = invalidValue
	payload       = []byte("[{'n':'test-name', 'v': 1.2}]")
	sessionClient = session.Session{
		ID:       clientID,
		Password: []byte(clientKey),
	}
	validToken                  = "token"
	validID                     = testsutil.GenerateUUID(&testing.T{})
	errClientNotInitialized     = errors.New("client is not initialized")
	errFailedPublish            = errors.New("failed to publish")
	errMissingTopicPub          = errors.New("failed to publish due to missing topic")
	errMalformedTopic           = errors.New("malformed topic")
	errFailedParseSubtopic      = errors.New("failed to parse subtopic")
	errMalformedSubtopic        = errors.New("malformed subtopic")
	errFailedPublishToMsgBroker = errors.New("failed to publish to supermq message broker")
)

var (
	clients   = new(clmocks.ClientsServiceClient)
	channels  = new(chmocks.ChannelsServiceClient)
	authn     = new(authnmocks.Authentication)
	publisher = new(mocks.PubSub)
)

func newHandler() session.Handler {
	logger := smqlog.NewMock()
	authn = new(authnmocks.Authentication)
	clients = new(clmocks.ClientsServiceClient)
	channels = new(chmocks.ChannelsServiceClient)
	publisher = new(mocks.PubSub)

	return mhttp.NewHandler(publisher, authn, clients, channels, logger)
}

func TestAuthConnect(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		session *session.Session
		status  int
		err     error
	}{
		{
			desc:    "connect with valid username and password",
			err:     nil,
			session: &sessionClient,
		},
		{
			desc:    "connect without active session",
			session: nil,
			status:  http.StatusUnauthorized,
			err:     errClientNotInitialized,
		},
		{
			desc: "connect with empty key",
			session: &session.Session{
				ID:       clientID,
				Password: []byte(""),
			},
			status: http.StatusBadRequest,
			err:    errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey),
		},
		{
			desc: "connect with client key",
			session: &session.Session{
				ID:       clientID,
				Password: []byte("Client " + clientKey),
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			err := handler.AuthConnect(ctx)
			hpe, ok := err.(mghttp.HTTPProxyError)
			if ok {
				assert.Equal(t, tc.status, hpe.StatusCode())
			}
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestPublish(t *testing.T) {
	handler := newHandler()

	malformedSubtopics := topic + "/" + subtopic + "%"

	clientKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}

	tokenSession := session.Session{
		Password: []byte(apiutil.BearerPrefix + validToken),
	}
	cases := []struct {
		desc       string
		topic      *string
		channelID  string
		payload    *[]byte
		password   string
		session    *session.Session
		status     int
		authNRes   *grpcClientsV1.AuthnRes
		authNRes1  smqauthn.Session
		authNErr   error
		authZRes   *grpcChannelsV1.AuthzRes
		authZErr   error
		publishErr error
		err        error
	}{
		{
			desc:      "publish  with key successfully",
			topic:     &topic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:  nil,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			authZErr:  nil,
			err:       nil,
		},
		{
			desc:      "publish  with token successfully",
			topic:     &topic,
			payload:   &payload,
			password:  validToken,
			session:   &tokenSession,
			channelID: chanID,
			authNRes1: smqauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			authNErr:  nil,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			authZErr:  nil,
			err:       nil,
		},
		{
			desc:      "publish  with key and subtopic successfully",
			topic:     &subtopic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:  nil,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			authZErr:  nil,
			err:       nil,
		},
		{
			desc:      "publish with empty topic",
			topic:     nil,
			payload:   &payload,
			session:   &clientKeySession,
			channelID: chanID,
			status:    http.StatusBadRequest,
			err:       errMissingTopicPub,
		},
		{
			desc:      "publish with invalid session",
			topic:     &topic,
			payload:   &payload,
			session:   nil,
			channelID: chanID,
			status:    http.StatusUnauthorized,
			err:       errClientNotInitialized,
		},
		{
			desc:     "publish with invalid topic",
			topic:    &invalidTopic,
			status:   http.StatusBadRequest,
			password: clientKey,
			session:  &clientKeySession,
			authNRes: &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr: nil,
			err:      errors.Wrap(errFailedPublish, errMalformedTopic),
		},
		{
			desc:     "publish with malformwd subtopic",
			topic:    &malformedSubtopics,
			status:   http.StatusBadRequest,
			password: clientKey,
			session:  &clientKeySession,
			authNRes: &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr: nil,
			err:      errors.Wrap(errFailedParseSubtopic, errMalformedSubtopic),
		},
		{
			desc:    "publish with empty password",
			topic:   &topic,
			payload: &payload,
			session: &session.Session{
				Password: []byte(""),
			},
			channelID: chanID,
			status:    http.StatusUnauthorized,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "publish with client key and failed to authenticate",
			topic:     &topic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			status:    http.StatusUnauthorized,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: false},
			authNErr:  nil,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "publish with client key and failed to authenticate with error",
			topic:     &topic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			status:    http.StatusUnauthorized,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: false},
			authNErr:  svcerr.ErrAuthentication,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "publish with  token and failed to authenticate",
			topic:     &topic,
			payload:   &payload,
			password:  validToken,
			session:   &tokenSession,
			channelID: chanID,
			status:    http.StatusUnauthorized,
			authNRes1: smqauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			authNErr:  svcerr.ErrAuthentication,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "publish with unauthorized client",
			topic:     &topic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			status:    http.StatusUnauthorized,
			authNErr:  nil,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			authZErr:  nil,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:      "publish with authorization error",
			topic:     &topic,
			payload:   &payload,
			password:  clientKey,
			session:   &clientKeySession,
			channelID: chanID,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			status:    http.StatusBadRequest,
			authNErr:  nil,
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			authZErr:  svcerr.ErrAuthorization,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:       "publish with failed to publish",
			topic:      &topic,
			payload:    &payload,
			password:   clientKey,
			session:    &clientKeySession,
			channelID:  chanID,
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			authZErr:   nil,
			publishErr: errors.New("failed to publish"),
			err:        errFailedPublishToMsgBroker,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			clientsCall := clients.On("Authenticate", ctx, &grpcClientsV1.AuthnReq{ClientSecret: tc.password}).Return(tc.authNRes, tc.authNErr)
			authCall := authn.On("Authenticate", ctx, mock.Anything).Return(tc.authNRes1, tc.authNErr)
			channelsCall := channels.On("Authorize", ctx, mock.Anything).Return(tc.authZRes, tc.authZErr)
			repoCall := publisher.On("Publish", ctx, tc.channelID, mock.Anything).Return(tc.publishErr)
			err := handler.Publish(ctx, tc.topic, tc.payload)
			hpe, ok := err.(mghttp.HTTPProxyError)
			if ok {
				assert.Equal(t, tc.status, hpe.StatusCode())
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected: %v, got: %v", tc.err, err))
			authCall.Unset()
			repoCall.Unset()
			clientsCall.Unset()
			channelsCall.Unset()
		})
	}
}
