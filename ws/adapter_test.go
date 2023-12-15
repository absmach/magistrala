// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/mocks"
	"github.com/absmach/magistrala/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func newService() (ws.Service, *mocks.PubSub, *authmocks.Service) {
	pubsub := new(mocks.PubSub)
	auth := new(authmocks.Service)

	return ws.New(auth, pubsub), pubsub, auth
}

func TestSubscribe(t *testing.T) {
	svc, pubsub, auth := newService()

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
			thingKey: authmocks.InvalidValue,
			chanID:   authmocks.InvalidValue,
			subtopic: subTopic,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		thingID := testsutil.GenerateUUID(t)
		subConfig := messaging.SubscriberConfig{
			ID:      thingID,
			Topic:   "channels." + chanID + "." + subTopic,
			Handler: c,
		}
		repocall := pubsub.On("Subscribe", mock.Anything, subConfig).Return(tc.err)
		repocall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true, Id: thingID}, nil)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall1.Parent.AssertCalled(t, "Authorize", mock.Anything, mock.Anything)
		repocall.Unset()
		repocall1.Unset()
	}
}
