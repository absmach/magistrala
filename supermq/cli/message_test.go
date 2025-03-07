// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/absmach/supermq/cli"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	sdkmocks "github.com/absmach/supermq/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
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
