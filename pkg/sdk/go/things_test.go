// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/senml+json"
	email       = "user@example.com"
	otherEmail  = "other_user@example.com"
	token       = "token"
	otherToken  = "other_token"
	wrongValue  = "wrong_value"
	badID       = "999"
	emptyValue  = ""

	keyPrefix = "123e4567-e89b-12d3-a456-"
)

var (
	metadata   = map[string]interface{}{"meta": "data"}
	metadata2  = map[string]interface{}{"meta": "data2"}
	thing      = sdk.Thing{ID: "001", Name: "test_device", Metadata: metadata}
	emptyThing = sdk.Thing{}
)

func newThingsService(tokens map[string]string) things.Service {
	policies := []mocks.MockSubjectSet{{Object: "users", Relation: "member"}}
	auth := mocks.NewAuthService(tokens, map[string][]mocks.MockSubjectSet{email: policies})
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, thingsRepo, channelsRepo, chanCache, thingCache, idProvider)
}

func newThingsServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc)
	return httptest.NewServer(mux)
}

func TestCreateThing(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		thing    sdk.Thing
		token    string
		err      error
		location string
	}{
		{
			desc:     "create new thing",
			thing:    thing,
			token:    token,
			err:      nil,
			location: "001",
		},
		{
			desc:     "create new empty thing",
			thing:    emptyThing,
			token:    token,
			err:      nil,
			location: "002",
		},
		{
			desc:     "create new thing with empty token",
			thing:    thing,
			token:    "",
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: "",
		},
		{
			desc:     "create new thing with invalid token",
			thing:    thing,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			location: "",
		},
	}
	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateThing(tc.thing, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.location, loc, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, loc))
	}
}

func TestCreateThings(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	things := []sdk.Thing{
		sdk.Thing{ID: "001", Name: "1", Key: "1"},
		sdk.Thing{ID: "002", Name: "2", Key: "2"},
	}

	cases := []struct {
		desc   string
		things []sdk.Thing
		token  string
		err    error
		res    []sdk.Thing
	}{
		{
			desc:   "create new things",
			things: things,
			token:  token,
			err:    nil,
			res:    things,
		},
		{
			desc:   "create new things with empty things",
			things: []sdk.Thing{},
			token:  token,
			err:    createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			res:    []sdk.Thing{},
		},
		{
			desc:   "create new thing with empty token",
			things: things,
			token:  "",
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:    []sdk.Thing{},
		},
		{
			desc:   "create new thing with invalid token",
			things: things,
			token:  wrongValue,
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:    []sdk.Thing{},
		},
	}
	for _, tc := range cases {
		res, err := mainfluxSDK.CreateThings(tc.things, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))

		for idx := range tc.res {
			assert.Equal(t, tc.res[idx].ID, res[idx].ID, fmt.Sprintf("%s: expected response ID %s got %s", tc.desc, tc.res[idx].ID, res[idx].ID))
		}
	}
}

func TestThing(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thing.Key = fmt.Sprintf("%s%012d", keyPrefix, 2)

	cases := []struct {
		desc     string
		thID     string
		token    string
		err      error
		response sdk.Thing
	}{
		{
			desc:     "get existing thing",
			thID:     id,
			token:    token,
			err:      nil,
			response: thing,
		},
		{
			desc:     "get non-existent thing",
			thID:     "43",
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusForbidden),
			response: sdk.Thing{},
		},
		{
			desc:     "get thing with invalid token",
			thID:     id,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Thing{},
		},
	}

	for _, tc := range cases {
		respTh, err := mainfluxSDK.Thing(tc.thID, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respTh, fmt.Sprintf("%s: expected response thing %s, got %s", tc.desc, tc.response, respTh))
	}
}

func TestThings(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	var things []sdk.Thing

	mainfluxSDK := sdk.NewSDK(sdkConf)
	for i := 1; i < 101; i++ {
		th := sdk.Thing{ID: fmt.Sprintf("%03d", i), Name: "test_device", Metadata: metadata}
		_, err := mainfluxSDK.CreateThing(th, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th.Key = fmt.Sprintf("%s%012d", keyPrefix, 2*i)
		things = append(things, th)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		err      error
		response []sdk.Thing
		name     string
	}{
		{
			desc:     "get a list of things",
			token:    token,
			offset:   0,
			limit:    5,
			err:      nil,
			response: things[0:5],
		},
		{
			desc:     "get a list of things with invalid token",
			token:    wrongValue,
			offset:   0,
			limit:    5,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things with empty token",
			token:    "",
			offset:   0,
			limit:    5,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things with zero limit",
			token:    token,
			offset:   0,
			limit:    0,
			err:      nil,
			response: things[0:10],
		},
		{
			desc:     "get a list of things with limit greater than max",
			token:    token,
			offset:   0,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things with offset greater than max",
			token:    token,
			offset:   110,
			limit:    5,
			err:      nil,
			response: []sdk.Thing{},
		},
	}
	for _, tc := range cases {
		page, err := mainfluxSDK.Things(tc.token, tc.offset, tc.limit, tc.name)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Things))
	}
}

func TestThingsByChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	ch := sdk.Channel{Name: "test_channel"}
	cid, err := mainfluxSDK.CreateChannel(ch, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var n = 10
	var thsDiscoNum = 1
	var things []sdk.Thing
	for i := 1; i < n+1; i++ {
		th := sdk.Thing{
			ID:       fmt.Sprintf("%03d", i),
			Name:     "test_device",
			Metadata: metadata,
			Key:      fmt.Sprintf("%s%012d", keyPrefix, 2*i+1),
		}
		tid, err := mainfluxSDK.CreateThing(th, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		things = append(things, th)

		// Don't connect last Thing
		if i == n+1-thsDiscoNum {
			break
		}

		// Don't connect last 2 Channels
		conIDs := sdk.ConnectionIDs{
			ChannelIDs: []string{cid},
			ThingIDs:   []string{tid},
		}
		err = mainfluxSDK.Connect(conIDs, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc         string
		channel      string
		token        string
		offset       uint64
		limit        uint64
		disconnected bool
		err          error
		response     []sdk.Thing
	}{
		{
			desc:     "get a list of things by channel",
			channel:  cid,
			token:    token,
			offset:   0,
			limit:    5,
			err:      nil,
			response: things[0:5],
		},
		{
			desc:     "get a list of things by channel with invalid token",
			channel:  cid,
			token:    wrongValue,
			offset:   0,
			limit:    5,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with empty token",
			channel:  cid,
			token:    "",
			offset:   0,
			limit:    5,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with zero limit",
			channel:  cid,
			token:    token,
			offset:   0,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with limit greater than max",
			channel:  cid,
			token:    token,
			offset:   0,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of things by channel with offset greater than max",
			channel:  cid,
			token:    token,
			offset:   110,
			limit:    5,
			err:      nil,
			response: []sdk.Thing{},
		},
		{
			desc:     "get a list of things by channel with invalid args (zero limit) and invalid token",
			channel:  cid,
			token:    wrongValue,
			offset:   0,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:         "get a list of not connected things by channel",
			channel:      cid,
			token:        token,
			offset:       0,
			limit:        100,
			disconnected: true,
			err:          nil,
			response:     []sdk.Thing{things[n-thsDiscoNum]},
		},
	}
	for _, tc := range cases {
		page, err := mainfluxSDK.ThingsByChannel(tc.token, tc.channel, tc.offset, tc.limit, tc.disconnected)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Things))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thing.Name = "test2"

	cases := []struct {
		desc  string
		thing sdk.Thing
		token string
		err   error
	}{
		{
			desc: "update existing thing",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: token,
			err:   nil,
		},
		{
			desc: "update non-existing thing",
			thing: sdk.Thing{
				ID:       "0",
				Name:     "test_device",
				Metadata: metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusForbidden),
		},
		{
			desc: "update channel with invalid id",
			thing: sdk.Thing{
				ID:       "",
				Name:     "test_device",
				Metadata: metadata,
			},
			token: token,
			err:   createError(sdk.ErrFailedUpdate, http.StatusBadRequest),
		},
		{
			desc: "update channel with invalid token",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: wrongValue,
			err:   createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc: "update channel with empty token",
			thing: sdk.Thing{
				ID:       id,
				Name:     "test_app",
				Metadata: metadata2,
			},
			token: "",
			err:   createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateThing(tc.thing, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteThing(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		token   string
		err     error
	}{
		{
			desc:    "delete thing with invalid token",
			thingID: id,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete non-existing thing",
			thingID: "2",
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusForbidden),
		},
		{
			desc:    "delete thing with invalid id",
			thingID: "",
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "delete thing with empty token",
			thingID: id,
			token:   "",
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete existing thing",
			thingID: id,
			token:   token,
			err:     nil,
		},
		{
			desc:    "delete deleted thing",
			thingID: id,
			token:   token,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteThing(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestConnectThing(t *testing.T) {
	svc := newThingsService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	thingID, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(channel, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		token   string
		err     error
	}{
		{
			desc:    "connect existing thing to existing channel",
			thingID: thingID,
			chanID:  chanID1,
			token:   token,
			err:     nil,
		},

		{
			desc:    "connect existing thing to non-existing channel",
			thingID: thingID,
			chanID:  "9",
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect non-existing thing to existing channel",
			thingID: "9",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect existing thing to channel with invalid ID",
			thingID: thingID,
			chanID:  "",
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},
		{
			desc:    "connect thing with invalid ID to existing channel",
			thingID: "",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},

		{
			desc:    "connect existing thing to existing channel with invalid token",
			thingID: thingID,
			chanID:  chanID1,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect existing thing to existing channel with empty token",
			thingID: thingID,
			chanID:  chanID1,
			token:   "",
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect thing from owner to channel of other user",
			thingID: thingID,
			chanID:  chanID2,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		conIDs := sdk.ConnectionIDs{
			ChannelIDs: []string{tc.chanID},
			ThingIDs:   []string{tc.thingID},
		}
		err := mainfluxSDK.Connect(conIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newThingsService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	thingID, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(channel, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		token   string
		err     error
	}{
		{
			desc:    "connect existing things to existing channels",
			thingID: thingID,
			chanID:  chanID1,
			token:   token,
			err:     nil,
		},

		{
			desc:    "connect existing things to non-existing channels",
			thingID: thingID,
			chanID:  badID,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect non-existing things to existing channels",
			thingID: badID,
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
		{
			desc:    "connect existing things to channels with invalid ID",
			thingID: thingID,
			chanID:  emptyValue,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},
		{
			desc:    "connect things with invalid ID to existing channels",
			thingID: emptyValue,
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusBadRequest),
		},

		{
			desc:    "connect existing things to existing channels with invalid token",
			thingID: thingID,
			chanID:  chanID1,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect existing things to existing channels with empty token",
			thingID: thingID,
			chanID:  chanID1,
			token:   emptyValue,
			err:     createError(sdk.ErrFailedConnect, http.StatusUnauthorized),
		},
		{
			desc:    "connect things from owner to channels of other user",
			thingID: thingID,
			chanID:  chanID2,
			token:   token,
			err:     createError(sdk.ErrFailedConnect, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		connIDs := sdk.ConnectionIDs{
			[]string{tc.thingID},
			[]string{tc.chanID},
		}

		err := mainfluxSDK.Connect(connIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDisconnectThing(t *testing.T) {
	svc := newThingsService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})

	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	thingID, err := mainfluxSDK.CreateThing(thing, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID1, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	conIDs := sdk.ConnectionIDs{
		ChannelIDs: []string{chanID1},
		ThingIDs:   []string{thingID},
	}
	err = mainfluxSDK.Connect(conIDs, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chanID2, err := mainfluxSDK.CreateChannel(channel, otherToken)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		token   string
		err     error
	}{
		{
			desc:    "disconnect connected thing from channel",
			thingID: thingID,
			chanID:  chanID1,
			token:   token,
			err:     nil,
		},
		{
			desc:    "disconnect existing thing from non-existing channel",
			thingID: thingID,
			chanID:  "9",
			token:   token,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
		{
			desc:    "disconnect non-existing thing from existing channel",
			thingID: "9",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
		{
			desc:    "disconnect existing thing from channel with invalid ID",
			thingID: thingID,
			chanID:  "",
			token:   token,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusBadRequest),
		},
		{
			desc:    "disconnect thing with invalid ID from existing channel",
			thingID: "",
			chanID:  chanID1,
			token:   token,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusBadRequest),
		},
		{
			desc:    "disconnect existing thing from existing channel with invalid token",
			thingID: thingID,
			chanID:  chanID1,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusUnauthorized),
		},
		{
			desc:    "disconnect existing thing from existing channel with empty token",
			thingID: thingID,
			chanID:  chanID1,
			token:   "",
			err:     createError(sdk.ErrFailedDisconnect, http.StatusUnauthorized),
		},
		{
			desc:    "disconnect owner's thing from someone elses channel",
			thingID: thingID,
			chanID:  chanID2,
			token:   token,
			err:     createError(sdk.ErrFailedDisconnect, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DisconnectThing(tc.thingID, tc.chanID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
