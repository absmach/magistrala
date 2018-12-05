//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	channel      = sdk.Channel{ID: "1", Name: "test"}
	emptyChannel = sdk.Channel{}
)

func TestCreateChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     error
		empty   bool
	}{
		{
			desc:    "create new channel",
			channel: channel,
			token:   token,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create new channel with empty token",
			channel: channel,
			token:   "",
			err:     sdk.ErrUnauthorized,
			empty:   true,
		},
		{
			desc:    "create new channel with invalid token",
			channel: channel,
			token:   wrongValue,
			err:     sdk.ErrUnauthorized,
			empty:   true,
		},
		{
			desc:    "create new empty channel",
			channel: emptyChannel,
			token:   token,
			err:     nil,
			empty:   false,
		},
	}

	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.empty, loc == "", fmt.Sprintf("%s: expected empty result location, got: %s", tc.desc, loc))
	}
}

func TestChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		chanID   string
		token    string
		err      error
		response sdk.Channel
	}{
		{
			desc:     "get existing channel",
			chanID:   id,
			token:    token,
			err:      nil,
			response: channel,
		},
		{
			desc:     "get non-existent channel",
			chanID:   "43",
			token:    token,
			err:      sdk.ErrNotFound,
			response: sdk.Channel{},
		},
		{
			desc:     "get channel with invalid token",
			chanID:   id,
			token:    "",
			err:      sdk.ErrUnauthorized,
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
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}
	var channels []sdk.Channel
	mainfluxSDK := sdk.NewSDK(sdkConf)
	for i := 1; i < 101; i++ {
		ch := sdk.Channel{ID: strconv.Itoa(i), Name: "test"}
		mainfluxSDK.CreateChannel(ch, token)
		channels = append(channels, ch)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		err      error
		response []sdk.Channel
	}{
		{
			desc:     "get a list of channels",
			token:    token,
			offset:   0,
			limit:    5,
			err:      nil,
			response: channels[0:5],
		},
		{
			desc:     "get a list of channels with invalid token",
			token:    wrongValue,
			offset:   0,
			limit:    5,
			err:      sdk.ErrUnauthorized,
			response: nil,
		},
		{
			desc:     "get a list of channels with empty token",
			token:    "",
			offset:   0,
			limit:    5,
			err:      sdk.ErrUnauthorized,
			response: nil,
		},
		{
			desc:     "get a list of channels with zero limit",
			token:    token,
			offset:   0,
			limit:    0,
			err:      sdk.ErrInvalidArgs,
			response: nil,
		},
		{
			desc:     "get a list of channels with limit greater than max",
			token:    token,
			offset:   0,
			limit:    110,
			err:      sdk.ErrInvalidArgs,
			response: nil,
		},
		{
			desc:     "get a list of channels with offset greater than max",
			token:    token,
			offset:   110,
			limit:    5,
			err:      nil,
			response: nil,
		},
		{
			desc:     "get a list of channels with invalid args (zero limit) and invalid token",
			token:    wrongValue,
			offset:   0,
			limit:    0,
			err:      sdk.ErrInvalidArgs,
			response: nil,
		},
	}
	for _, tc := range cases {
		respChs, err := mainfluxSDK.Channels(tc.token, tc.offset, tc.limit)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respChs, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, respChs))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     error
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
			err:     sdk.ErrNotFound,
		},
		{
			desc:    "update channel with invalid id",
			channel: sdk.Channel{ID: "", Name: "test2"},
			token:   token,
			err:     sdk.ErrInvalidArgs,
		},
		{
			desc:    "update channel with invalid token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   wrongValue,
			err:     sdk.ErrUnauthorized,
		},
		{
			desc:    "update channel with empty token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   "",
			err:     sdk.ErrUnauthorized,
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteChannel(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(channel, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		chanID string
		token  string
		err    error
	}{
		{
			desc:   "delete channel with invalid token",
			chanID: id,
			token:  wrongValue,
			err:    sdk.ErrUnauthorized,
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
			err:    sdk.ErrInvalidArgs,
		},
		{
			desc:   "delete channel with empty token",
			chanID: id,
			token:  "",
			err:    sdk.ErrUnauthorized,
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
