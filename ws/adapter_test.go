// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	authmocks "github.com/mainflux/mainflux/auth/mocks"
	"github.com/mainflux/mainflux/internal/testsutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/mocks"
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

func newService() (ws.Service, mocks.MockPubSub, *authmocks.Service) {
	pubsub := mocks.NewPubSub()
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
			err:      errors.Wrap(ws.ErrFailedSubscription, ws.ErrFailedSubscription),
		},
		{
			desc:     "subscribe to channel with invalid chanID and invalid thingKey",
			thingKey: authmocks.InvalidValue,
			chanID:   authmocks.InvalidValue,
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty channel",
			thingKey: thingKey,
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey",
			thingKey: "",
			chanID:   chanID,
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
		{
			desc:     "subscribe to channel with empty thingKey and empty channel",
			thingKey: "",
			chanID:   "",
			subtopic: subTopic,
			fail:     false,
			err:      ws.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		pubsub.SetFail(tc.fail)
		repocall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true, Id: testsutil.GenerateUUID(t)}, nil)
		err := svc.Subscribe(context.Background(), tc.thingKey, tc.chanID, tc.subtopic, c)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}
