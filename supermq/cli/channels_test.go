// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/absmach/supermq/cli"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	mgsdk "github.com/absmach/supermq/pkg/sdk"
	sdkmocks "github.com/absmach/supermq/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var channel = mgsdk.Channel{
	ID:   testsutil.GenerateUUID(&testing.T{}),
	Name: "testchannel",
}

func TestCreateChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelJson := "{\"name\":\"testchannel\", \"metadata\":{\"key1\":\"value1\"}}"
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)

	cp := mgsdk.Channel{}
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		channel       mgsdk.Channel
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "create channel successfully",
			args: []string{
				channelJson,
				domainID,
				token,
			},
			channel: channel,
			logType: entityLog,
		},
		{
			desc: "create channel with invalid args",
			args: []string{
				channelJson,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create channel with invalid json",
			args: []string{
				"{\"name\":\"testchannel\", \"metadata\":{\"key1\":\"value1\"}",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "create channel with invalid token",
			args: []string{
				channelJson,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateChannel", mock.Anything, tc.args[1], tc.args[2]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &cp)
				assert.Nil(t, err)
				assert.Equal(t, tc.channel, cp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.channel, cp))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestGetChannelsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)

	var ch mgsdk.Channel
	var page mgsdk.ChannelsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          mgsdk.ChannelsPage
		channel       mgsdk.Channel
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get all channels successfully",
			args: []string{
				all,
				domainID,
				token,
			},
			page: mgsdk.ChannelsPage{
				Channels: []mgsdk.Channel{channel},
			},
			logType: entityLog,
		},
		{
			desc: "get channel with id",
			args: []string{
				channel.ID,
				domainID,
				token,
			},
			logType: entityLog,
			channel: channel,
		},
		{
			desc: "get channels with invalid args",
			args: []string{
				all,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all channels with invalid token",
			args: []string{
				all,
				domainID,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get channel with invalid id",
			args: []string{
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Channel", tc.args[0], tc.args[1], tc.args[2]).Return(tc.channel, tc.sdkErr)
			sdkCall1 := sdkMock.On("Channels", mock.Anything, tc.args[1], tc.args[2]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				if tc.args[1] == all {
					err := json.Unmarshal([]byte(out), &page)
					assert.Nil(t, err)
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				} else {
					err := json.Unmarshal([]byte(out), &ch)
					assert.Nil(t, err)
					assert.Equal(t, tc.channel, ch, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.channel, ch))
				}
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
			sdkCall1.Unset()
		})
	}
}

func TestDeleteChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "delete channel successfully",
			args: []string{
				channel.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete channel with invalid args",
			args: []string{
				channel.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete channel with invalid channel id",
			args: []string{
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete channel with invalid token",
			args: []string{
				channel.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteChannel", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{delCmd}, tc.args...)...)

			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestUpdateChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)

	newChannelJson := "{\"name\" : \"channel1\"}"
	cases := []struct {
		desc          string
		args          []string
		channel       mgsdk.Channel
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "update channel successfully",
			args: []string{
				channel.ID,
				newChannelJson,
				domainID,
				token,
			},
			channel: mgsdk.Channel{
				Name: "newchannel1",
				ID:   channel.ID,
			},
			logType: entityLog,
		},
		{
			desc: "update channel with invalid args",
			args: []string{
				channel.ID,
				newChannelJson,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "update channel with invalid channel id",
			args: []string{
				invalidID,
				newChannelJson,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update channel with invalid json syntax",
			args: []string{
				channel.ID,
				"{\"name\" : \"channel1\"",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var ch mgsdk.Channel
			sdkCall := sdkMock.On("UpdateChannel", mock.Anything, tc.args[2], tc.args[3]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				assert.Nil(t, err)
				assert.Equal(t, tc.channel, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.channel, ch))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestEnableChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)
	var ch mgsdk.Channel

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		channel       mgsdk.Channel
		logType       outputLog
	}{
		{
			desc: "enable channel successfully",
			args: []string{
				channel.ID,
				domainID,
				validToken,
			},
			channel: channel,
			logType: entityLog,
		},
		{
			desc: "delete channel with invalid token",
			args: []string{
				channel.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete channel with invalid channel ID",
			args: []string{
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "enable channel with invalid args",
			args: []string{
				channel.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableChannel", tc.args[0], tc.args[1], tc.args[2]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				assert.Nil(t, err)
				assert.Equal(t, tc.channel, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.channel, ch))
			}

			sdkCall.Unset()
		})
	}
}

func TestDisableChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	var ch mgsdk.Channel

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		channel       mgsdk.Channel
		logType       outputLog
	}{
		{
			desc: "disable channel successfully",
			args: []string{
				channel.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			channel: channel,
		},
		{
			desc: "disable channel with invalid token",
			args: []string{
				channel.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable channel with invalid id",
			args: []string{
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable client with invalid args",
			args: []string{
				channel.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableChannel", tc.args[0], tc.args[1], tc.args[2]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{disableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				if err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				assert.Equal(t, tc.channel, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.channel, ch))
			}

			sdkCall.Unset()
		})
	}
}
