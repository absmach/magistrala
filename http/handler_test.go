// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	mgate "github.com/absmach/mgate/pkg/http"
	"github.com/absmach/mgate/pkg/session"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	clmocks "github.com/absmach/supermq/clients/mocks"
	dmocks "github.com/absmach/supermq/domains/mocks"
	smqhttp "github.com/absmach/supermq/http"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	invalidValue   = "invalid"
	topicMsg       = "/m/%s/c/%s"
	subtopicMsg    = "/m/%s/c/%s/subtopic"
	topic          = fmt.Sprintf(topicMsg, domainID, chanID)
	subtopic       = fmt.Sprintf(subtopicMsg, domainID, chanID)
	hcTopicFmt     = "/hc/%s"
	hcTopic        = fmt.Sprintf(hcTopicFmt, domainID)
	invalidHCTopic = "/hc"
	invalidTopic   = invalidValue
	topics         = []string{topic}
	payload        = []byte("[{'n':'test-name', 'v': 1.2}]")
	sessionClient  = session.Session{
		ID:       clientID,
		Password: []byte(clientKey),
	}
	invalidChannelIDTopic   = "m/**/c"
	validToken              = "token"
	errClientNotInitialized = errors.New("client is not initialized")
	errMissingTopicPub      = errors.New("failed to publish due to missing topic")
	errMissingTopicSub      = errors.New("failed to subscribe due to missing topic")
)

var (
	clients   = new(clmocks.ClientsServiceClient)
	channels  = new(chmocks.ChannelsServiceClient)
	authn     = new(authnmocks.Authentication)
	publisher = new(mocks.PubSub)
	domains   = new(dmocks.DomainsServiceClient)
)

func newHandler(t *testing.T) session.Handler {
	logger := smqlog.NewMock()
	authn = new(authnmocks.Authentication)
	clients = new(clmocks.ClientsServiceClient)
	channels = new(chmocks.ChannelsServiceClient)
	publisher = new(mocks.PubSub)
	parser, err := messaging.NewTopicParser(messaging.DefaultCacheConfig, channels, domains)
	assert.Nil(t, err, fmt.Sprintf("unexpected error while creating topic parser: %v", err))

	return smqhttp.NewHandler(publisher, logger, authn, clients, channels, parser)
}

func TestAuthPublish(t *testing.T) {
	handler := newHandler(t)

	clientKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}
	unauthorizedKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}
	invalidClientKeySession := session.Session{
		Password: []byte("Client " + invalidKey),
	}
	tokenSession := session.Session{
		Password: []byte(apiutil.BearerPrefix + validToken),
	}
	invalidTokenSession := session.Session{
		Password: []byte(apiutil.BearerPrefix + invalidToken),
	}
	basicAuthSession := session.Session{
		Username: clientID,
		Password: []byte(clientKey),
	}
	invalidBasicAuthSession := session.Session{
		Username: clientID,
		Password: []byte(invalidValue),
	}
	creds := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientKey)))
	encodedCredsSession := session.Session{
		Password: []byte(apiutil.BasicAuthPrefix + creds),
	}
	invalidCreds := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, invalidValue)))
	invalidEncodedCredsSession := session.Session{
		Password: []byte(apiutil.BasicAuthPrefix + invalidCreds),
	}
	hcClientKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}

	tests := []struct {
		desc       string
		session    *session.Session
		topic      *string
		payload    *[]byte
		authKey    string
		status     int
		clientType string
		chanID     string
		domainID   string
		clientID   string
		authNToken string
		superAdmin bool
		authNRes   *grpcClientsV1.AuthnRes
		authNRes1  smqauthn.Session
		authNErr   error
		authZRes   *grpcChannelsV1.AuthzRes
		authZErr   error
		err        error
	}{
		{
			desc:       "publish with client key successfully",
			session:    &clientKeySession,
			topic:      &topic,
			authKey:    clientKey,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with invalid client key",
			session:    &invalidClientKeySession,
			topic:      &topic,
			authKey:    invalidKey,
			payload:    &payload,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, invalidKey),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:    "publish with nil session",
			session: nil,
			topic:   &topic,
			authKey: clientKey,
			status:  http.StatusInternalServerError,
			err:     errClientNotInitialized,
		},
		{
			desc:    "publish with empty topic",
			session: &clientKeySession,
			topic:   nil,
			authKey: clientKey,
			status:  http.StatusBadRequest,
			err:     errMissingTopicPub,
		},
		{
			desc:       "publish with unauthorized client key",
			session:    &unauthorizedKeySession,
			topic:      &topic,
			authKey:    clientKey,
			payload:    &payload,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with token successfully",
			session:    &tokenSession,
			topic:      &topic,
			authKey:    token,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{UserID: userID},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with superadmin token successfully",
			session:    &tokenSession,
			topic:      &topic,
			authKey:    token,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			superAdmin: true,
			authNRes1:  smqauthn.Session{UserID: userID, Role: smqauthn.SuperAdminRole},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with invalid token",
			session:    &invalidTokenSession,
			topic:      &topic,
			authKey:    invalidToken,
			payload:    &payload,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{},
			authNErr:   svcerr.ErrAuthentication,
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with unauthorized token",
			session:    &tokenSession,
			topic:      &topic,
			authKey:    token,
			payload:    &payload,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{UserID: userID},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with basic auth successfully",
			session:    &basicAuthSession,
			topic:      &topic,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with invalid basic auth",
			session:    &invalidBasicAuthSession,
			topic:      &topic,
			payload:    &payload,
			authKey:    invalidValue,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, invalidValue),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with b64 encoded credentials",
			session:    &encodedCredsSession,
			topic:      &topic,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with invalid b64 encoded credentials",
			session:    &invalidEncodedCredsSession,
			topic:      &topic,
			payload:    &payload,
			authKey:    invalidValue,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, invalidValue),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with health check topic successfully",
			session:    &hcClientKeySession,
			topic:      &hcTopic,
			authKey:    clientKey,
			payload:    &payload,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     "",
			domainID:   domainID,
			clientID:   userID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: true},
			authNErr:   nil,
		},
		{
			desc:       "publish with invalid health check topic",
			session:    &hcClientKeySession,
			topic:      &invalidHCTopic,
			authKey:    clientKey,
			payload:    &payload,
			status:     http.StatusBadRequest,
			err:        messaging.ErrMalformedTopic,
			clientType: policies.ClientType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			tc.clientType = policies.ClientType
			clientID := tc.clientID
			if tc.session != nil && strings.HasPrefix(string(tc.session.Password), apiutil.BearerPrefix) {
				tc.clientType = policies.UserType
				clientID = policies.EncodeDomainUserID(tc.domainID, tc.clientID)
				if tc.superAdmin {
					clientID = tc.clientID
				}
			}
			clientsCall := clients.On("Authenticate", ctx, &grpcClientsV1.AuthnReq{Token: tc.authNToken}).Return(tc.authNRes, tc.authNErr)
			authCall := authn.On("Authenticate", ctx, mock.Anything).Return(tc.authNRes1, tc.authNErr)
			channelsCall := channels.On("Authorize", mock.Anything, &grpcChannelsV1.AuthzReq{
				ClientType: tc.clientType,
				ClientId:   clientID,
				Type:       uint32(connections.Publish),
				ChannelId:  tc.chanID,
				DomainId:   tc.domainID,
			}).Return(tc.authZRes, tc.authZErr)
			err := handler.AuthPublish(ctx, tc.topic, tc.payload)
			hpe, ok := err.(mgate.HTTPProxyError)
			if ok {
				assert.Equal(t, tc.status, hpe.StatusCode())
			}
			if tc.err != nil {
				assert.Contains(t, err.Error(), tc.err.Error(), fmt.Sprintf("expected error message to contain: %v, got: %v", tc.err, err))
			}
			authCall.Unset()
			clientsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func TestAuthSubscribe(t *testing.T) {
	handler := newHandler(t)

	clientKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}
	unauthorizedKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}
	invalidClientKeySession := session.Session{
		Password: []byte("Client " + invalidKey),
	}
	tokenSession := session.Session{
		Password: []byte(apiutil.BearerPrefix + validToken),
	}
	invalidTokenSession := session.Session{
		Password: []byte(apiutil.BearerPrefix + invalidToken),
	}
	basicAuthSession := session.Session{
		Username: clientID,
		Password: []byte(clientKey),
	}
	invalidBasicAuthSession := session.Session{
		Username: clientID,
		Password: []byte(invalidValue),
	}
	creds := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientKey)))
	encodedCredsSession := session.Session{
		Password: []byte(apiutil.BasicAuthPrefix + creds),
	}
	invalidCreds := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, invalidValue)))
	invalidEncodedCredsSession := session.Session{
		Password: []byte(apiutil.BasicAuthPrefix + invalidCreds),
	}
	hcClientKeySession := session.Session{
		Password: []byte("Client " + clientKey),
	}

	tests := []struct {
		desc       string
		session    *session.Session
		topics     *[]string
		authKey    string
		status     int
		clientType string
		chanID     string
		domainID   string
		clientID   string
		authNToken string
		superAdmin bool
		authNRes   *grpcClientsV1.AuthnRes
		authNRes1  smqauthn.Session
		authNErr   error
		authZRes   *grpcChannelsV1.AuthzRes
		authZErr   error
		err        error
	}{
		{
			desc:       "subscribe with client key successfully",
			session:    &clientKeySession,
			topics:     &topics,
			authKey:    clientKey,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "subscribe with invalid client key",
			session:    &invalidClientKeySession,
			topics:     &topics,
			authKey:    invalidKey,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, invalidKey),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:    "subscribe with empty topics",
			session: &clientKeySession,
			topics:  nil,
			authKey: clientKey,
			status:  http.StatusBadRequest,
			err:     errMissingTopicSub,
		},
		{
			desc:    "subscribe with nil session",
			session: nil,
			topics:  &topics,
			authKey: clientKey,
			status:  http.StatusInternalServerError,
			err:     errClientNotInitialized,
		},
		{
			desc:       "subscribe with unauthorized client key",
			session:    &unauthorizedKeySession,
			topics:     &topics,
			authKey:    clientKey,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "subscribe with token successfully",
			session:    &tokenSession,
			topics:     &topics,
			authKey:    token,
			status:     http.StatusOK,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{UserID: userID},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "subscribe with superadmin token successfully",
			session:    &tokenSession,
			topics:     &topics,
			authKey:    token,
			status:     http.StatusOK,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			superAdmin: true,
			authNRes1:  smqauthn.Session{UserID: userID, Role: smqauthn.SuperAdminRole},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "subscribe with invalid token",
			session:    &invalidTokenSession,
			topics:     &topics,
			authKey:    invalidToken,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{},
			authNErr:   svcerr.ErrAuthentication,
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "subscribe with unauthorized token",
			session:    &tokenSession,
			topics:     &topics,
			authKey:    token,
			clientType: policies.UserType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   userID,
			authNRes1:  smqauthn.Session{UserID: userID},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "subscribe with basic auth successfully",
			session:    &basicAuthSession,
			topics:     &topics,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "subscribe with invalid basic auth",
			session:    &invalidBasicAuthSession,
			topics:     &topics,
			authKey:    invalidValue,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, invalidValue),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "publish with b64 encoded credentials",
			session:    &encodedCredsSession,
			topics:     &topics,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authNErr:   nil,
			authZRes:   &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "publish with invalid b64 encoded credentials",
			session:    &invalidEncodedCredsSession,
			topics:     &topics,
			authKey:    invalidValue,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.BasicAuth, clientID, invalidValue),
			authNRes:   &grpcClientsV1.AuthnRes{Authenticated: false},
			status:     http.StatusUnauthorized,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "subscribe with health check topic successfully",
			session:    &hcClientKeySession,
			topics:     &[]string{hcTopic},
			authKey:    clientKey,
			status:     http.StatusOK,
			clientType: policies.ClientType,
			chanID:     chanID,
			domainID:   domainID,
			clientID:   clientID,
			authNToken: smqauthn.AuthPack(smqauthn.DomainAuth, domainID, clientKey),
			authNRes:   &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			err:        nil,
		},
		{
			desc:       "subscribe with invalid health check topic",
			session:    &hcClientKeySession,
			topics:     &[]string{invalidHCTopic},
			authKey:    clientKey,
			status:     http.StatusBadRequest,
			err:        messaging.ErrMalformedTopic,
			clientType: policies.ClientType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.TODO()
			if tc.session != nil {
				ctx = session.NewContext(ctx, tc.session)
			}
			tc.clientType = policies.ClientType
			clientID := tc.clientID
			if tc.session != nil && strings.HasPrefix(string(tc.session.Password), apiutil.BearerPrefix) {
				tc.clientType = policies.UserType
				clientID = policies.EncodeDomainUserID(tc.domainID, tc.clientID)
				if tc.superAdmin {
					clientID = tc.clientID
				}
			}
			clientsCall := clients.On("Authenticate", ctx, &grpcClientsV1.AuthnReq{Token: tc.authNToken}).Return(tc.authNRes, tc.authNErr)
			authCall := authn.On("Authenticate", ctx, mock.Anything).Return(tc.authNRes1, tc.authNErr)
			channelsCall := channels.On("Authorize", mock.Anything, &grpcChannelsV1.AuthzReq{
				ClientType: tc.clientType,
				ClientId:   clientID,
				Type:       uint32(connections.Subscribe),
				ChannelId:  tc.chanID,
				DomainId:   tc.domainID,
			}).Return(tc.authZRes, tc.authZErr)
			err := handler.AuthSubscribe(ctx, tc.topics)
			hpe, ok := err.(mgate.HTTPProxyError)
			if ok {
				assert.Equal(t, tc.status, hpe.StatusCode())
			}
			if tc.err != nil {
				assert.Contains(t, err.Error(), tc.err.Error(), fmt.Sprintf("expected error message to contain: %v, got: %v", tc.err, err))
			}
			authCall.Unset()
			clientsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func TestPublish(t *testing.T) {
	handler := newHandler(t)

	malformedSubtopics := topic + "/" + subtopic + "%"
	wrongCharSubtopics := topic + "/" + subtopic + ">"
	validSubtopic := topic + "/" + subtopic

	cases := []struct {
		desc    string
		session *session.Session
		topic   string
		payload []byte
		err     error
	}{
		{
			desc:    "publish without active session",
			session: nil,
			topic:   topic,
			payload: payload,
			err:     errClientNotInitialized,
		},
		{
			desc:    "publish with invalid topic",
			session: &sessionClient,
			topic:   invalidTopic,
			payload: payload,
			err:     messaging.ErrMalformedTopic,
		},
		{
			desc:    "publish with invalid channel ID",
			session: &sessionClient,
			topic:   invalidChannelIDTopic,
			payload: payload,
			err:     messaging.ErrMalformedTopic,
		},
		{
			desc:    "publish with malformed subtopic",
			session: &sessionClient,
			topic:   malformedSubtopics,
			payload: payload,
			err:     messaging.ErrMalformedTopic,
		},
		{
			desc:    "publish with subtopic containing wrong character",
			session: &sessionClient,
			topic:   wrongCharSubtopics,
			payload: payload,
			err:     messaging.ErrMalformedTopic,
		},
		{
			desc:    "publish with subtopic",
			session: &sessionClient,
			topic:   validSubtopic,
			payload: payload,
		},
		{
			desc:    "publish without subtopic",
			session: &sessionClient,
			topic:   topic,
			payload: payload,
		},
		{
			desc:    "publish with health check topic",
			session: &sessionClient,
			topic:   hcTopic,
			payload: payload,
		},
		{
			desc:    "puvlish with invalid health check topic",
			session: &sessionClient,
			topic:   invalidHCTopic,
			payload: payload,
			err:     messaging.ErrMalformedTopic,
		},
	}

	for _, tc := range cases {
		ctx := context.TODO()
		if tc.session != nil {
			ctx = session.NewContext(ctx, tc.session)
		}
		repoCall := publisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := handler.Publish(ctx, &tc.topic, &tc.payload)
		if tc.err != nil {
			assert.Contains(t, err.Error(), tc.err.Error(), fmt.Sprintf("expected error message to contain: %v, got: %v", tc.err, err))
		}
		repoCall.Unset()
	}
}
