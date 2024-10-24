// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	chmocks "github.com/absmach/magistrala/channels/mocks"
	climocks "github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/mocks"
	"github.com/absmach/magistrala/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	chanID     = "1"
	invalidID  = "invalidID"
	invalidKey = "invalidKey"
	id         = "1"
	clientKey   = "client_key"
	subTopic   = "subtopic"
	protocol   = "ws"
)

var msg = messaging.Message{
	Channel:   chanID,
	Publisher: id,
	Subtopic:  "",
	Protocol:  protocol,
	Payload:   []byte(`[{"n":"current","t":-5,"v":1.2}]`),
}

func newService() (ws.Service, *mocks.PubSub, *climocks.ClientsServiceClient, *chmocks.ChannelsServiceClient) {
	pubsub := new(mocks.PubSub)
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)

	return ws.New(clients, channels, pubsub), pubsub, clients, channels
}

func TestSubscribe(t *testing.T) {
	svc, pubsub, _, _ := newService()

	c := ws.NewClient(nil)

	cases := []struct {
		desc     string
		clientKey string
		chanID   string
		subtopic string
		err      error
	}{
		{
			desc:     "subscribe to channel with valid clientKey, chanID, subtopic",
			clientKey: clientKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      nil,
		},
		{
			desc:     "subscribe again to channel with valid clientKey, chanID, subtopic",
			clientKey: clientKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      nil,
		},
		{
			desc:     "subscribe to channel with subscribe set to fail",
			clientKey: clientKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with invalid chanID and invalid clientKey",
			clientKey: invalidKey,
			chanID:   invalidID,
			subtopic: subTopic,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with empty channel",
			clientKey: clientKey,
			chanID:   "",
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "subscribe to channel with empty clientKey",
			clientKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "subscribe to channel with empty clientKey and empty channel",
			clientKey: "",
			chanID:   "",
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		clientID := testsutil.GenerateUUID(t)
		subConfig := messaging.SubscriberConfig{
			ID:      clientID,
			Topic:   "channels." + tc.chanID + "." + subTopic,
			Handler: c,
		}
		repocall := pubsub.On("Subscribe", mock.Anything, subConfig).Return(tc.err)
		err := svc.Subscribe(context.Background(), tc.clientKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}
