// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package lora_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/lora"
	"github.com/absmach/magistrala/lora/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	pubmocks "github.com/absmach/magistrala/pkg/messaging/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	thingID  = "thingID-1"
	chanID   = "chanID-1"
	devEUI   = "devEUI-1"
	appID    = "appID-1"
	thingID2 = "thingID-2"
	chanID2  = "chanID-2"
	devEUI2  = "devEUI-2"
	appID2   = "appID-2"
	msg      = `[{"bn":"msg-base-name","n":"temperature","v": 17},{"n":"humidity","v": 56}]`
	invalid  = "wrong"
)

var (
	pub                           *pubmocks.PubSub
	thingsRM, channelsRM, connsRM *mocks.RouteMapRepository
)

func newService() lora.Service {
	pub = new(pubmocks.PubSub)
	thingsRM = new(mocks.RouteMapRepository)
	channelsRM = new(mocks.RouteMapRepository)
	connsRM = new(mocks.RouteMapRepository)

	return lora.New(pub, thingsRM, channelsRM, connsRM)
}

func TestPublish(t *testing.T) {
	svc := newService()

	msgBase64 := base64.StdEncoding.EncodeToString([]byte(msg))

	cases := []struct {
		desc           string
		err            error
		msg            lora.Message
		getThingErr    error
		getChannelErr  error
		connectionsErr error
		publishErr     error
	}{
		{
			desc: "publish message with existing route-map and valid Data",
			err:  nil,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        devEUI,
				Data:          msgBase64,
			},
			getThingErr:    nil,
			getChannelErr:  nil,
			connectionsErr: nil,
			publishErr:     nil,
		},
		{
			desc: "publish message with existing route-map and invalid Data",
			err:  lora.ErrMalformedMessage,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        devEUI,
				Data:          "wrong",
			},
			getThingErr:    nil,
			getChannelErr:  nil,
			connectionsErr: nil,
			publishErr:     errors.New("Failed publishing"),
		},
		{
			desc: "publish message with non existing appID route-map",
			err:  lora.ErrNotFoundApp,
			msg: lora.Message{
				ApplicationID: "wrong",
				DevEUI:        devEUI,
			},
			getChannelErr: lora.ErrNotFoundApp,
		},
		{
			desc: "publish message with non existing devEUI route-map",
			err:  lora.ErrNotFoundDev,
			msg: lora.Message{
				ApplicationID: appID,
				DevEUI:        "wrong",
			},
			getThingErr: lora.ErrNotFoundDev,
		},
		{
			desc: "publish message with non existing connection route-map",
			err:  lora.ErrNotConnected,
			msg: lora.Message{
				ApplicationID: appID2,
				DevEUI:        devEUI2,
			},
			connectionsErr: lora.ErrNotConnected,
		},
		{
			desc: "publish message with wrong Object",
			err:  errors.New("json: unsupported type: chan int"),
			msg: lora.Message{
				ApplicationID: appID2,
				DevEUI:        devEUI2,
				Object:        make(chan int),
			},
		},
		{
			desc: "publish message with valid Object",
			err:  nil,
			msg: lora.Message{
				ApplicationID: appID2,
				DevEUI:        devEUI2,
				Object:        map[string]interface{}{"key": "value"},
			},
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Get", context.Background(), tc.msg.DevEUI).Return(tc.msg.DevEUI, tc.getThingErr)
		repoCall1 := channelsRM.On("Get", context.Background(), tc.msg.ApplicationID).Return(tc.msg.ApplicationID, tc.getChannelErr)
		repoCall2 := connsRM.On("Get", context.Background(), mock.Anything).Return("", tc.connectionsErr)
		repoCall3 := pub.On("Publish", context.Background(), tc.msg.ApplicationID, mock.Anything).Return(tc.publishErr)
		err := svc.Publish(context.Background(), &tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc   string
		err    error
		ChanID string
		AppID  string
	}{
		{
			desc:   "create channel with valid data",
			err:    nil,
			ChanID: chanID,
			AppID:  appID,
		},
		{
			desc:   "create channel with empty chanID",
			err:    lora.ErrNotFoundApp,
			ChanID: "",
			AppID:  appID,
		},
		{
			desc:   "create channel with empty appID",
			err:    lora.ErrNotFoundApp,
			ChanID: chanID,
			AppID:  "",
		},
	}

	for _, tc := range cases {
		repoCall := channelsRM.On("Save", context.Background(), tc.ChanID, tc.AppID).Return(tc.err)
		err := svc.CreateChannel(context.Background(), tc.ChanID, tc.AppID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestCreateThing(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		err     error
		ThingID string
		DevEUI  string
	}{
		{
			desc:    "create thing with valid data",
			err:     nil,
			ThingID: thingID,
			DevEUI:  devEUI,
		},
		{
			desc:    "create thing with empty thingID",
			err:     lora.ErrNotFoundDev,
			ThingID: "",
			DevEUI:  devEUI,
		},
		{
			desc:    "create thing with empty devEUI",
			err:     lora.ErrNotFoundDev,
			ThingID: thingID,
			DevEUI:  "",
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Save", context.Background(), tc.ThingID, tc.DevEUI).Return(tc.err)
		err := svc.CreateThing(context.Background(), tc.ThingID, tc.DevEUI)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestConnectThing(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc          string
		err           error
		channelID     string
		thingID       string
		getThingErr   error
		getChannelErr error
	}{
		{
			desc:          "connect thing with valid data",
			err:           nil,
			channelID:     chanID,
			thingID:       thingID,
			getThingErr:   nil,
			getChannelErr: nil,
		},
		{
			desc:        "connect thing with non existing thing",
			err:         lora.ErrNotFoundDev,
			channelID:   chanID,
			thingID:     invalid,
			getThingErr: lora.ErrNotFoundDev,
		},
		{
			desc:          "connect thing with non existing channel",
			err:           lora.ErrNotFoundApp,
			channelID:     invalid,
			thingID:       thingID,
			getChannelErr: lora.ErrNotFoundApp,
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Get", context.Background(), tc.thingID).Return(devEUI, tc.getThingErr)
		repoCall1 := channelsRM.On("Get", context.Background(), tc.channelID).Return(appID, tc.getChannelErr)
		repoCall2 := connsRM.On("Save", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
		err := svc.ConnectThing(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDisconnectThing(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc          string
		err           error
		channelID     string
		thingID       string
		getThingErr   error
		getChannelErr error
	}{
		{
			desc:          "disconnect thing with valid data",
			err:           nil,
			channelID:     chanID,
			thingID:       thingID,
			getThingErr:   nil,
			getChannelErr: nil,
		},
		{
			desc:        "disconnect thing with non existing thing ID",
			err:         lora.ErrNotFoundDev,
			channelID:   chanID,
			thingID:     invalid,
			getThingErr: lora.ErrNotFoundDev,
		},
		{
			desc:          "disconnect thing with non existing channel",
			err:           lora.ErrNotFoundApp,
			channelID:     invalid,
			thingID:       thingID,
			getChannelErr: lora.ErrNotFoundApp,
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Get", context.Background(), tc.thingID).Return(devEUI, tc.getThingErr)
		repoCall1 := channelsRM.On("Get", context.Background(), tc.channelID).Return(appID, tc.getChannelErr)
		repoCall2 := connsRM.On("Remove", context.Background(), mock.Anything).Return(tc.err)
		err := svc.DisconnectThing(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc   string
		err    error
		ChanID string
	}{
		{
			desc:   "remove channel with valid data",
			err:    nil,
			ChanID: chanID,
		},
		{
			desc:   "remove channel with non existing channel",
			err:    lora.ErrNotFoundApp,
			ChanID: invalid,
		},
		{
			desc:   "remove channel with empty channelID",
			err:    lora.ErrNotFoundApp,
			ChanID: "",
		},
	}

	for _, tc := range cases {
		repoCall := channelsRM.On("Remove", context.Background(), tc.ChanID).Return(tc.err)
		err := svc.RemoveChannel(context.Background(), tc.ChanID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		err     error
		ThingID string
	}{
		{
			desc:    "remove thing with valid data",
			err:     nil,
			ThingID: thingID,
		},
		{
			desc:    "remove thing with non existing thing",
			err:     lora.ErrNotFoundDev,
			ThingID: invalid,
		},
		{
			desc:    "remove thing with empty thingID",
			err:     lora.ErrNotFoundDev,
			ThingID: "",
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Remove", context.Background(), tc.ThingID).Return(tc.err)
		err := svc.RemoveThing(context.Background(), tc.ThingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc   string
		err    error
		ChanID string
		AppID  string
	}{
		{
			desc:   "update channel with valid data",
			err:    nil,
			ChanID: chanID,
			AppID:  appID,
		},
		{
			desc:   "update channel with non existing channel",
			err:    lora.ErrNotFoundApp,
			ChanID: invalid,
			AppID:  appID,
		},
		{
			desc:   "update channel with empty channelID",
			err:    lora.ErrNotFoundApp,
			ChanID: "",
			AppID:  appID,
		},
		{
			desc:   "update channel with empty appID",
			err:    lora.ErrNotFoundApp,
			ChanID: chanID,
			AppID:  "",
		},
		{
			desc:   "update channel with non existing appID",
			err:    lora.ErrNotFoundApp,
			ChanID: chanID,
			AppID:  invalid,
		},
	}

	for _, tc := range cases {
		repoCall := channelsRM.On("Save", context.Background(), tc.ChanID, tc.AppID).Return(tc.err)
		err := svc.UpdateChannel(context.Background(), tc.ChanID, tc.AppID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		err     error
		ThingID string
		DevEUI  string
	}{
		{
			desc:    "update thing with valid data",
			err:     nil,
			ThingID: thingID,
			DevEUI:  devEUI,
		},
		{
			desc:    "update thing with non existing thing",
			err:     lora.ErrNotFoundDev,
			ThingID: invalid,
			DevEUI:  devEUI,
		},
		{
			desc:    "update thing with empty thingID",
			err:     lora.ErrNotFoundDev,
			ThingID: "",
			DevEUI:  devEUI,
		},
		{
			desc:    "update thing with empty devEUI",
			err:     lora.ErrNotFoundDev,
			ThingID: thingID,
			DevEUI:  "",
		},
		{
			desc:    "update thing with non existing devEUI",
			err:     lora.ErrNotFoundDev,
			ThingID: thingID,
			DevEUI:  invalid,
		},
	}

	for _, tc := range cases {
		repoCall := thingsRM.On("Save", context.Background(), tc.ThingID, tc.DevEUI).Return(tc.err)
		err := svc.UpdateThing(context.Background(), tc.ThingID, tc.DevEUI)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
