// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	chmocks "github.com/absmach/supermq/channels/mocks"
	climocks "github.com/absmach/supermq/clients/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	chanID     = "1"
	invalidID  = "invalidID"
	invalidKey = "invalidKey"
	id         = "1"
	clientKey  = "client_key"
	subTopic   = "subtopic"
	protocol   = "ws"
)

var (
	msg = messaging.Message{
		Channel:   chanID,
		Publisher: id,
		Subtopic:  "",
		Protocol:  protocol,
		Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
	}
	clientID = testsutil.GenerateUUID(&testing.T{})
)

func newService() (ws.Service, *mocks.PubSub, *climocks.ClientsServiceClient, *chmocks.ChannelsServiceClient) {
	pubsub := new(mocks.PubSub)
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)

	return ws.New(clients, channels, pubsub), pubsub, clients, channels
}

func TestSubscribe(t *testing.T) {
	svc, pubsub, clients, channels := newService()

	c := ws.NewClient(nil)

	cases := []struct {
		desc      string
		clientKey string
		chanID    string
		subtopic  string
		authNRes  *grpcClientsV1.AuthnRes
		authNErr  error
		authZRes  *grpcChannelsV1.AuthzRes
		authZErr  error
		subErr    error
		err       error
	}{
		{
			desc:      "subscribe to channel with valid clientKey, chanID, subtopic",
			clientKey: clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			err:       nil,
		},
		{
			desc:      "subscribe again to channel with valid clientKey, chanID, subtopic",
			clientKey: clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			err:       nil,
		},
		{
			desc:      "subscribe to channel with subscribe set to fail",
			clientKey: clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			subErr:    ws.ErrFailedSubscription,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			err:       ws.ErrFailedSubscription,
		},
		{
			desc:      "subscribe to channel with invalid clientKey",
			clientKey: invalidKey,
			chanID:    invalidID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Authenticated: false},
			authNErr:  svcerr.ErrAuthentication,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:      "subscribe to channel with empty channel",
			clientKey: clientKey,
			chanID:    "",
			subtopic:  subTopic,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "subscribe to channel with empty clientKey",
			clientKey: "",
			chanID:    chanID,
			subtopic:  subTopic,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "subscribe to channel with empty clientKey and empty channel",
			clientKey: "",
			chanID:    "",
			subtopic:  subTopic,
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "subscribe to channel with invalid channel",
			clientKey: clientKey,
			chanID:    invalidID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			authZErr:  svcerr.ErrAuthorization,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:      "subscribe to channel with failed authentication",
			clientKey: clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Authenticated: false},
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:      "subscribe to channel with failed authorization",
			clientKey: clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: false},
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:      "subscribe to channel with valid clientKey prefixed with 'client_', chanID, subtopic",
			clientKey: "Client " + clientKey,
			chanID:    chanID,
			subtopic:  subTopic,
			authNRes:  &grpcClientsV1.AuthnRes{Id: clientID, Authenticated: true},
			authZRes:  &grpcChannelsV1.AuthzRes{Authorized: true},
			err:       nil,
		},
	}

	for _, tc := range cases {
		subConfig := messaging.SubscriberConfig{
			ID:       clientID,
			Topic:    "channels." + tc.chanID + "." + subTopic,
			ClientID: clientID,
			Handler:  c,
		}
		authReq := &grpcClientsV1.AuthnReq{ClientSecret: tc.clientKey}
		if strings.HasPrefix(tc.clientKey, "Client") {
			authReq.ClientSecret = strings.TrimPrefix(tc.clientKey, "Client ")
		}
		clientsCall := clients.On("Authenticate", mock.Anything, authReq).Return(tc.authNRes, tc.authNErr)
		channelsCall := channels.On("Authorize", mock.Anything, &grpcChannelsV1.AuthzReq{
			ClientType: policies.ClientType,
			ClientId:   tc.authNRes.GetId(),
			Type:       uint32(connections.Subscribe),
			ChannelId:  tc.chanID,
		}).Return(tc.authZRes, tc.authZErr)
		repocall := pubsub.On("Subscribe", mock.Anything, subConfig).Return(tc.subErr)
		err := svc.Subscribe(context.Background(), tc.clientKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
		clientsCall.Unset()
		channelsCall.Unset()
	}
}
