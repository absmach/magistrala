// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

var (
	ch1          = sdk.Channel{Name: "test1"}
	ch2          = sdk.Channel{ID: "fe6b4e92-cc98-425e-b0aa-000000000001", Name: "test1"}
	ch3          = sdk.Channel{ID: "fe6b4e92-cc98-425e-b0aa-000000000002", Name: "test2"}
	chPrefix     = "fe6b4e92-cc98-425e-b0aa-"
	emptyChannel = sdk.Channel{}
)

func TestCreateChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	chWrongExtID := sdk.Channel{ID: "b0aa-000000000001", Name: "1", Metadata: metadata}

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     errors.SDKError
		empty   bool
	}{
		{
			desc:    "create new channel",
			channel: ch1,
			token:   token,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create new channel with empty token",
			channel: ch1,
			token:   "",
			err:     errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create new channel with invalid token",
			channel: ch1,
			token:   wrongValue,
			err:     errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create new empty channel",
			channel: emptyChannel,
			token:   token,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create a new channel with external UUID",
			channel: ch2,
			token:   token,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create a new channel with wrong external UUID",
			channel: chWrongExtID,
			token:   token,
			err:     errors.NewSDKErrorWithStatus(apiutil.ErrInvalidIDFormat, http.StatusBadRequest),
			empty:   true,
		},
	}

	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.empty, loc == "", fmt.Sprintf("%s: expected empty result location, got: %s", tc.desc, loc))
	}
}

func TestCreateChannels(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	channels := []sdk.Channel{
		ch2,
		ch3,
	}

	cases := []struct {
		desc     string
		channels []sdk.Channel
		token    string
		err      errors.SDKError
		res      []sdk.Channel
	}{
		{
			desc:     "create new channels",
			channels: channels,
			token:    token,
			err:      nil,
			res:      channels,
		},
		{
			desc:     "create new channels with empty channels",
			channels: []sdk.Channel{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrEmptyList, http.StatusBadRequest),
			res:      []sdk.Channel{},
		},
		{
			desc:     "create new channels with empty token",
			channels: channels,
			token:    "",
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			res:      []sdk.Channel{},
		},
		{
			desc:     "create new channels with invalid token",
			channels: channels,
			token:    wrongValue,
			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			res:      []sdk.Channel{},
		},
	}
	for _, tc := range cases {
		res, err := mainfluxSDK.CreateChannels(tc.channels, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))

		for idx := range tc.res {
			assert.Equal(t, tc.res[idx].ID, res[idx].ID, fmt.Sprintf("%s: expected response ID %s got %s", tc.desc, tc.res[idx].ID, res[idx].ID))
		}
	}
}

func TestChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: adminEmail})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(ch2, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		chanID   string
		token    string
		err      errors.SDKError
		response sdk.Channel
	}{
		{
			desc:     "get existing channel",
			chanID:   id,
			token:    token,
			err:      nil,
			response: ch2,
		},
		{
			desc:     "get non-existent channel",
			chanID:   "43",
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
			response: sdk.Channel{},
		},
		{
			desc:     "get channel with invalid token",
			chanID:   id,
			token:    "",
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			response: sdk.Channel{},
		},
	}

	for _, tc := range cases {
		respCh, err := mainfluxSDK.Channel(tc.chanID, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respCh, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, respCh))
	}
}

func TestChannels(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	var channels []sdk.Channel
	mainfluxSDK := sdk.NewSDK(sdkConf)
	for i := 1; i < 101; i++ {
		id := fmt.Sprintf("%s%012d", chPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		ch := sdk.Channel{ID: id, Name: name}
		_, err := mainfluxSDK.CreateChannel(ch, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		channels = append(channels, ch)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		name     string
		err      errors.SDKError
		response []sdk.Channel
		metadata map[string]interface{}
	}{
		{
			desc:     "get a list of channels",
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: channels[0:limit],
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with invalid token",
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels without limit, default 10",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with offset greater than max",
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Channel{},
			metadata: make(map[string]interface{}),
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Name:     tc.name,
			Total:    total,
			Offset:   uint64(tc.offset),
			Limit:    uint64(tc.limit),
			Metadata: tc.metadata,
		}

		page, err := mainfluxSDK.Channels(filter, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Channels, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Channels))
	}
}

func TestChannelsByThing(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	th := sdk.Thing{Name: "test_device"}
	tid, err := mainfluxSDK.CreateThing(th, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var n = 100
	var chsDiscoNum = 1
	var channels []sdk.Channel
	for i := 1; i < n+1; i++ {
		id := fmt.Sprintf("%s%012d", chPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		ch := sdk.Channel{ID: id, Name: name}
		cid, err := mainfluxSDK.CreateChannel(ch, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		channels = append(channels, ch)

		// Don't connect last Channel
		if i == n+1-chsDiscoNum {
			break
		}

		conIDs := sdk.ConnectionIDs{
			ChannelIDs: []string{cid},
			ThingIDs:   []string{tid},
		}
		err = mainfluxSDK.Connect(conIDs, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc         string
		thing        string
		token        string
		offset       uint64
		limit        uint64
		disconnected bool
		err          errors.SDKError
		response     []sdk.Channel
	}{
		{
			desc:     "get a list of channels by thing",
			thing:    tid,
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: channels[0:limit],
		},
		{
			desc:     "get a list of channels by thing with invalid token",
			thing:    tid,
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of channels by thing with empty token",
			thing:    tid,
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of channels by thing with zero limit",
			thing:    tid,
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of channels by thing with limit greater than max",
			thing:    tid,
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of channels by thing with offset greater than max",
			thing:    tid,
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Channel{},
		},
		{
			desc:     "get a list of channels by thing with invalid args (zero limit) and invalid token",
			thing:    tid,
			token:    wrongValue,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:         "get a list of not connected channels by thing",
			thing:        tid,
			token:        token,
			offset:       offset,
			limit:        100,
			disconnected: true,
			err:          nil,
			response:     []sdk.Channel{channels[n-chsDiscoNum]},
		},
	}

	for _, tc := range cases {
		pm := sdk.PageMetadata{
			Offset:       tc.offset,
			Limit:        tc.limit,
			Disconnected: tc.disconnected,
		}
		page, err := mainfluxSDK.ChannelsByThing(tc.thing, pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Channels, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Channels))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: adminEmail})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(ch2, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     errors.SDKError
	}{
		{
			desc:    "update existing channel",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   token,
			err:     nil,
		},
		{
			desc:    "update non-existing channel",
			channel: sdk.Channel{ID: "0", Name: "test2"},
			token:   token,
			err:     errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:    "update channel with invalid id",
			channel: sdk.Channel{ID: "", Name: "test2"},
			token:   token,
			err:     errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:    "update channel with invalid token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   wrongValue,
			err:     errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "update channel with empty token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   "",
			err:     errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: adminEmail})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(ch2, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		chanID string
		token  string
		err    errors.SDKError
	}{
		{
			desc:   "delete channel with invalid token",
			chanID: id,
			token:  wrongValue,
			err:    errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "delete non-existing channel",
			chanID: "2",
			token:  token,
			err:    nil,
		},
		{
			desc:   "delete channel with invalid id",
			chanID: "",
			token:  token,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:   "delete channel with empty token",
			chanID: id,
			token:  "",
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:   "delete existing channel",
			chanID: id,
			token:  token,
			err:    nil,
		},
		{
			desc:   "delete deleted channel",
			chanID: id,
			token:  token,
			err:    nil,
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteChannel(tc.chanID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
