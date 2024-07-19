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

var journal = mgsdk.Journal{
	ID: testsutil.GenerateUUID(&testing.T{}),
}

func TestGetJournalCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewJournalCmd()
	rootCmd := setFlags(invCmd)

	var page mgsdk.JournalsPage
	entityType := "entity_type"
	entityId := journal.ID

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          mgsdk.JournalsPage
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get journal with journal id",
			args: []string{
				entityType,
				entityId,
				token,
			},
			logType: entityLog,
			page: mgsdk.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []mgsdk.Journal{journal},
			},
		},
		{
			desc: "get journal with invalid args",
			args: []string{
				entityType,
				entityId,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get journal with invalid token",
			args: []string{
				entityType,
				entityId,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Journal", tc.args[0], tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &page)
				assert.Nil(t, err)
				assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}
