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

var invitation = mgsdk.Invitation{
	InvitedBy: testsutil.GenerateUUID(&testing.T{}),
	UserID:    user.ID,
	DomainID:  domain.ID,
}

func TestSendUserInvitationCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewInvitationsCmd()
	rootCmd := setFlags(invCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "send invitation successfully",
			args: []string{
				user.ID,
				domain.ID,
				relation,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "send invitation with invalid args",
			args: []string{
				user.ID,
				domain.ID,
				relation,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "send invitation with invalid token",
			args: []string{
				user.ID,
				domain.ID,
				relation,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("SendInvitation", mock.Anything, mock.Anything).Return(tc.sdkErr)
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

func TestGetInvitationCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewInvitationsCmd()
	rootCmd := setFlags(invCmd)

	var inv mgsdk.Invitation
	var page mgsdk.InvitationPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          mgsdk.InvitationPage
		inv           mgsdk.Invitation
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get all invitations successfully",
			args: []string{
				all,
				token,
			},
			page: mgsdk.InvitationPage{
				Total:       1,
				Offset:      0,
				Limit:       10,
				Invitations: []mgsdk.Invitation{invitation},
			},
			logType: entityLog,
		},
		{
			desc: "get invitation with user id",
			args: []string{
				user.ID,
				domain.ID,
				token,
			},
			logType: entityLog,
			inv:     invitation,
		},
		{
			desc: "get invitation with invalid args",
			args: []string{
				all,
				token,
				extraArg,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all invitations with invalid token",
			args: []string{
				all,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get invitation with invalid token",
			args: []string{
				user.ID,
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Invitation", tc.args[0], tc.args[1], mock.Anything).Return(tc.inv, tc.sdkErr)
			sdkCall1 := sdkMock.On("Invitations", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				if tc.args[0] == all {
					err := json.Unmarshal([]byte(out), &page)
					assert.Nil(t, err)
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				} else {
					err := json.Unmarshal([]byte(out), &inv)
					assert.Nil(t, err)
					assert.Equal(t, tc.inv, inv, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.inv, inv))
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

func TestAcceptInvitationCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewInvitationsCmd()
	rootCmd := setFlags(invCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "accept invitation successfully",
			args: []string{
				domain.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "accept invitation with invalid args",
			args: []string{
				domain.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "accept invitation with invalid token",
			args: []string{
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AcceptInvitation", mock.Anything, mock.Anything).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{acceptCmd}, tc.args...)...)
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

func TestRejectInvitationCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewInvitationsCmd()
	rootCmd := setFlags(invCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "reject invitation successfully",
			args: []string{
				domain.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "reject invitation with invalid args",
			args: []string{
				domain.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "reject invitation with invalid token",
			args: []string{
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RejectInvitation", mock.Anything, mock.Anything).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{rejectCmd}, tc.args...)...)
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

func TestDeleteInvitationCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	invCmd := cli.NewInvitationsCmd()
	rootCmd := setFlags(invCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "delete invitation successfully",
			args: []string{
				user.ID,
				domain.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "delete invitation with invalid args",
			args: []string{
				user.ID,
				domain.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete invitation with invalid token",
			args: []string{
				user.ID,
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteInvitation", mock.Anything, mock.Anything, mock.Anything).Return(tc.sdkErr)
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
