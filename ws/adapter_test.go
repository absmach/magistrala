// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/testsutil"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/mocks"
	thmocks "github.com/absmach/magistrala/things/mocks"
	"github.com/absmach/magistrala/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	chanID     = "1"
	invalidID  = "invalidID"
	invalidKey = "invalidKey"
	id         = "1"
	thingKey   = "thing_key"
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

func newService() (ws.Service, *mocks.PubSub, *thmocks.ThingsServiceClient) {
	pubsub := new(mocks.PubSub)
	things := new(thmocks.ThingsServiceClient)

	return ws.New(things, pubsub), pubsub, things
}

func TestSubscribe(t *testing.T) {
	svc, pubsub, things := newService()

	c := ws.NewClient(nil)

	cases := []struct {
		desc     string
		thingKey string
		chanID   string
		subtopic string
		err      error
	}{
		{
			desc:     "subscribe to channel with valid thingKey, chanID, subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      nil,
		},
		{
			desc:     "subscribe again to channel with valid thingKey, chanID, subtopic",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      nil,
		},
		{
			desc:     "subscribe to channel with subscribe set to fail",
			thingKey: thingKey,
			chanID:   chanID,
			subtopic: subTopic,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with invalid chanID and invalid thingKey",
			thingKey: invalidKey,
			chanID:   invalidID,
			subtopic: subTopic,
			err:      ws.ErrFailedSubscription,
		},
		{
			desc:     "subscribe to channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "subscribe to channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "subscribe to channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		thingID := testsutil.GenerateUUID(t)
		subConfig := messaging.SubscriberConfig{
			ID:      thingID,
			Topic:   "channels." + tc.chanID + "." + subTopic,
			Handler: c,
		}
		repocall := pubsub.On("Subscribe", mock.Anything, subConfig).Return(tc.err)
		repocall1 := things.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.ThingsAuthzRes{Authorized: true, Id: thingID}, nil)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall1.Parent.AssertCalled(t, "Authorize", mock.Anything, mock.Anything)
		repocall.Unset()
		repocall1.Unset()
	}
}
