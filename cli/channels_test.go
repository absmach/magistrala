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
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
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
				channelJson,
				token,
			},
			channel: channel,
			logType: entityLog,
		},
		{
			desc: "create channel with invalid args",
			args: []string{
				channelJson,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create channel with invalid json",
			args: []string{
				"{\"name\":\"testchannel\", \"metadata\":{\"key1\":\"value1\"}",
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
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "create channel without domain token",
			args: []string{
				channelJson,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateChannel", mock.Anything, tc.args[1]).Return(tc.channel, tc.sdkErr)
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
				token,
			},
			logType: entityLog,
			channel: channel,
		},
		{
			desc: "get channels with invalid args",
			args: []string{
				all,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all channels with invalid token",
			args: []string{
				all,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get channel without domain token",
			args: []string{
				channel.ID,
				tokenWithoutDomain,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get channel with invalid id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Channel", tc.args[0], tc.args[1]).Return(tc.channel, tc.sdkErr)
			sdkCall1 := sdkMock.On("Channels", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)

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
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete channel with invalid args",
			args: []string{
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete channel with invalid channel id",
			args: []string{
				invalidID,
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
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteChannel", tc.args[0], tc.args[1]).Return(tc.sdkErr)
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
			sdkCall := sdkMock.On("UpdateChannel", mock.Anything, tc.args[2]).Return(tc.channel, tc.sdkErr)
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

func TestListConnectionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelCmd)

	var tp mgsdk.ThingsPage
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		page          mgsdk.ThingsPage
	}{
		{
			desc: "list connections successfully",
			args: []string{
				channel.ID,
				token,
			},
			page: mgsdk.ThingsPage{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Things: []mgsdk.Thing{thing},
			},
			logType: entityLog,
		},
		{
			desc: "list connections with invalid args",
			args: []string{
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list connections with invalid channel id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ThingsByChannel", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{connsCmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &tp)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				assert.Equal(t, tc.page, tp, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, tp))
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
				validToken,
			},
			channel: channel,
			logType: entityLog,
		},
		{
			desc: "delete channel with invalid token",
			args: []string{
				channel.ID,
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
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableChannel", tc.args[0], tc.args[1]).Return(tc.channel, tc.sdkErr)
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
				validToken,
			},
			logType: entityLog,
			channel: channel,
		},
		{
			desc: "disable channel with invalid token",
			args: []string{
				channel.ID,
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
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable thing with invalid args",
			args: []string{
				channel.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableChannel", tc.args[0], tc.args[1]).Return(tc.channel, tc.sdkErr)
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

func TestUsersChannelCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	page := mgsdk.UsersPage{}

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		page          mgsdk.UsersPage
		sdkErr        errors.SDKError
	}{
		{
			desc: "get channel's users successfully",
			args: []string{
				channel.ID,
				token,
			},
			page: mgsdk.UsersPage{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Users: []mgsdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "list channel users with invalid args",
			args: []string{
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list channel users without domain token",
			args: []string{
				channel.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list channel users with invalid id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListChannelUsers", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{usrCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &page)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestListGroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	var gp mgsdk.GroupsPage
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		page          mgsdk.GroupsPage
	}{
		{
			desc: "list groups successfully",
			args: []string{
				channel.ID,
				token,
			},
			page: mgsdk.GroupsPage{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Groups: []mgsdk.Group{group},
			},
			logType: entityLog,
		},
		{
			desc: "list groups with invalid args",
			args: []string{
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list groups with invalid channel id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListChannelUserGroups", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{grpCmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &gp)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				assert.Equal(t, tc.page, gp, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, gp))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestAssignUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	userIds := fmt.Sprintf("[\"%s\"]", user.ID)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "assign user successfully",
			args: []string{
				relation,
				userIds,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "assign user with invalid args",
			args: []string{
				relation,
				userIds,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "assign user with invalid json",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"", user.ID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "assign user with invalid channel id",
			args: []string{
				relation,
				userIds,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "assign user with invalid user id",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"]", invalidID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddUserToChannel", tc.args[2], mock.Anything, tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{assignCmd, usrCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestAssignGroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	grpIds := fmt.Sprintf("[\"%s\"]", group.ID)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "assign group successfully",
			args: []string{
				grpIds,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "assign group with invalid args",
			args: []string{
				grpIds,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "assign group with invalid json",
			args: []string{
				fmt.Sprintf("[\"%s\"", group.ID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "assign group with invalid channel id",
			args: []string{
				grpIds,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "assign group with invalid user id",
			args: []string{
				fmt.Sprintf("[\"%s\"]", invalidID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddUserGroupToChannel", tc.args[1], mock.Anything, tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{assignCmd, grpCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestUnassignUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	userIds := fmt.Sprintf("[\"%s\"]", user.ID)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "unassign user successfully",
			args: []string{
				relation,
				userIds,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unassign user with invalid args",
			args: []string{
				relation,
				userIds,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unassign user with invalid json",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"", user.ID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "unassign user with invalid channel id",
			args: []string{
				relation,
				userIds,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "unassign user with invalid user id",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"]", invalidID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RemoveUserFromChannel", tc.args[2], mock.Anything, tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{unassignCmd, usrCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestUnassignGroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCmd := cli.NewChannelsCmd()
	rootCmd := setFlags(channelsCmd)

	grpIds := fmt.Sprintf("[\"%s\"]", group.ID)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "unassign group successfully",
			args: []string{
				unassignCmd,
				grpCmd,
				grpIds,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unassign group with invalid args",
			args: []string{
				grpIds,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unassign group with invalid json",
			args: []string{
				fmt.Sprintf("[\"%s\"", group.ID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "unassign group with invalid channel id",
			args: []string{
				grpIds,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "unassign group with invalid user id",
			args: []string{
				fmt.Sprintf("[\"%s\"]", invalidID),
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RemoveUserGroupFromChannel", tc.args[1], mock.Anything, tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{unassignCmd, grpCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}
