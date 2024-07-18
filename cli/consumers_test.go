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

var subscription = mgsdk.Subscription{
	ID:      testsutil.GenerateUUID(&testing.T{}),
	OwnerID: user.ID,
	Topic:   "topic",
	Contact: "identity@example.com",
}

func TestCreateSubscriptionCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	subCmd := cli.NewSubscriptionCmd()
	rootCmd := setFlags(subCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
		response      string
		id            string
	}{
		{
			desc: "create subscription successfully",
			args: []string{
				subscription.Topic,
				subscription.Contact,
				validToken,
			},
			id:       user.ID,
			response: fmt.Sprintf("\ncreated: %s\n\n", user.ID),
			logType:  createLog,
		},
		{
			desc: "create subscription with invalid args",
			args: []string{
				subscription.Topic,
				subscription.Contact,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create subscription with invalid token",
			args: []string{
				subscription.Topic,
				subscription.Contact,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateSubscription", tc.args[0], tc.args[1], tc.args[2]).Return(tc.id, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case createLog:
				assert.Equal(t, tc.response, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.response, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestGetSubscriptionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	subCmd := cli.NewSubscriptionCmd()
	rootCmd := setFlags(subCmd)

	var sub mgsdk.Subscription
	var page mgsdk.SubscriptionPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          mgsdk.SubscriptionPage
		subscription  mgsdk.Subscription
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get all subscriptions successfully",
			args: []string{
				all,
				token,
			},
			page: mgsdk.SubscriptionPage{
				Subscriptions: []mgsdk.Subscription{subscription},
			},
			logType: entityLog,
		},
		{
			desc: "get subscription with id",
			args: []string{
				subscription.ID,
				token,
			},
			logType:      entityLog,
			subscription: subscription,
		},
		{
			desc: "get subscriptions with invalid args",
			args: []string{
				all,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all subscriptions with invalid token",
			args: []string{
				all,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get subscription without domain token",
			args: []string{
				subscription.ID,
				tokenWithoutDomain,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get subscription with invalid id",
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
			sdkCall := sdkMock.On("ViewSubscription", tc.args[0], tc.args[1]).Return(tc.subscription, tc.sdkErr)
			sdkCall1 := sdkMock.On("ListSubscriptions", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				if tc.args[1] == all {
					err := json.Unmarshal([]byte(out), &page)
					assert.Nil(t, err)
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				} else {
					err := json.Unmarshal([]byte(out), &sub)
					assert.Nil(t, err)
					assert.Equal(t, tc.subscription, sub, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.subscription, sub))
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

func TestRemoveSubscriptionCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	subCmd := cli.NewSubscriptionCmd()
	rootCmd := setFlags(subCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "remove subscription successfully",
			args: []string{
				subscription.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "remove subscription with invalid args",
			args: []string{
				subscription.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "remove subscription with invalid subscription id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "remove subscription with invalid token",
			args: []string{
				subscription.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteSubscription", tc.args[0], tc.args[1]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{rmCmd}, tc.args...)...)

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
