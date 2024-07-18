// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/pkg/errors"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHealthCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	healthCmd := cli.NewHealthCmd()
	rootCmd := setFlags(healthCmd)
	service := "users"

	var health mgsdk.HealthInfo
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		health        mgsdk.HealthInfo
		sdkErr        errors.SDKError
	}{
		{
			desc: "Check health successfully",
			args: []string{
				service,
			},
			logType: entityLog,
			health: mgsdk.HealthInfo{
				Status:      "pass",
				Description: "users service",
			},
		},
		{
			desc: "Check health with invalid args",
			args: []string{
				service,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "Check health with invalid service",
			args: []string{
				"invalid",
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.New("unsupported protocol scheme"), 306),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(errors.New("unsupported protocol scheme"), 306)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Health", mock.Anything).Return(tc.health, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &health)
				assert.Nil(t, err)
				assert.Equal(t, tc.health, health, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.health, health))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.True(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}
