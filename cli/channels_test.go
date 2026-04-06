// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
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
				createCmd,
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
				createCmd,
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
				createCmd,
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
				createCmd,
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
			var sdkCall *mock.Call
			if len(tc.args) >= 4 {
				sdkCall = sdkMock.On("CreateChannel", mock.Anything, mock.Anything, tc.args[2], tc.args[3]).Return(tc.channel, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &cp)
				assert.Nil(t, err)
				assert.Equal(t, tc.channel, cp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.channel, cp))
			case usageLog:
				assert.True(t, strings.Contains(out, "cli channels create"), fmt.Sprintf("%s invalid usage: expected to contain create usage, got: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			if sdkCall != nil {
				sdkCall.Unset()
			}
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
				getCmd,
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
				getCmd,
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
				getCmd,
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
				getCmd,
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
				getCmd,
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
			sdkCall := sdkMock.On("Channel", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.channel, tc.sdkErr)
			sdkCall1 := sdkMock.On("Channels", mock.Anything, mock.Anything, tc.args[2], tc.args[3]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				if tc.args[0] == all {
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
				delCmd,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete channel with invalid args",
			args: []string{
				channel.ID,
				delCmd,
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
				delCmd,
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
				delCmd,
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
			sdkCall := sdkMock.On("DeleteChannel", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				updateCmd,
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
				updateCmd,
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
				updateCmd,
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
				updateCmd,
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
			sdkCall := sdkMock.On("UpdateChannel", mock.Anything, mock.Anything, tc.args[3], tc.args[4]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				enableCmd,
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
				enableCmd,
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
				enableCmd,
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
				enableCmd,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableChannel", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				disableCmd,
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
				disableCmd,
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
				disableCmd,
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
				disableCmd,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableChannel", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.channel, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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

func TestChannelUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	var mp mgsdk.EntityMembersPage

	memberRole := mgsdk.MemberRoles{
		MemberID: testsutil.GenerateUUID(t),
		Roles:    []mgsdk.MemberRole{},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		usersPage     mgsdk.EntityMembersPage
		logType       outputLog
	}{
		{
			desc: "list channel users successfully",
			args: []string{
				channel.ID,
				usersCmd,
				domainID,
				validToken,
			},
			usersPage: mgsdk.EntityMembersPage{
				Members: []mgsdk.MemberRoles{memberRole},
			},
			logType: entityLog,
		},
		{
			desc: "list channel users with invalid token",
			args: []string{
				channel.ID,
				usersCmd,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list channel users with invalid channel id",
			args: []string{
				invalidID,
				usersCmd,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list channel users with invalid args",
			args: []string{
				channel.ID,
				usersCmd,
				domainID,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListChannelMembers", mock.Anything, tc.args[0], tc.args[2], mock.Anything, tc.args[3]).Return(tc.usersPage, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &mp)
				if err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				assert.Equal(t, tc.usersPage, mp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.usersPage, mp))
			}

			sdkCall.Unset()
		})
	}
}
