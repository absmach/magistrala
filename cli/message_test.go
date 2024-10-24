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
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendMesageCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	messageCmd := cli.NewMessagesCmd()
	rootCmd := setFlags(messageCmd)

	message := "[{\"bn\":\"Dev1\",\"n\":\"temp\",\"v\":20}, {\"n\":\"hum\",\"v\":40}, {\"bn\":\"Dev2\", \"n\":\"temp\",\"v\":20}, {\"n\":\"hum\",\"v\":40}]"

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "send message successfully",
			args: []string{
				channel.ID,
				message,
				client.Credentials.Secret,
			},
			logType: okLog,
		},
		{
			desc: "send message with invalid args",
			args: []string{
				channel.ID,
				message,
				client.Credentials.Secret,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "send message with invalid client secret",
			args: []string{
				channel.ID,
				message,
				"invalid_secret",
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(svcerr.ErrAuthorization, svcerr.ErrNotFound)), http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(svcerr.ErrAuthorization, svcerr.ErrNotFound)), http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("SendMessage", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{sendCmd}, tc.args...)...)

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

func TestReadMesageCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	messageCmd := cli.NewMessagesCmd()
	rootCmd := setFlags(messageCmd)

	var mp mgsdk.MessagesPage
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
		page          mgsdk.MessagesPage
	}{
		{
			desc: "read message successfully",
			args: []string{
				channel.ID,
				domainID,
				validToken,
			},
			page: mgsdk.MessagesPage{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Messages: []senml.Message{
					{
						Channel: channel.ID,
					},
				},
			},
			logType: entityLog,
		},
		{
			desc: "read message with invalid args",
			args: []string{
				channel.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "read message with invalid token",
			args: []string{
				channel.ID,
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
			sdkCall := sdkMock.On("ReadMessages", mock.Anything, tc.args[0], tc.args[1], tc.args[2]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{readCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &mp)
				assert.Nil(t, err)
				assert.Equal(t, tc.page, mp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, mp))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}
