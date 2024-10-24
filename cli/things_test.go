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
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	token              = "valid" + "domaintoken"
	domainID           = "domain-id"
	tokenWithoutDomain = "valid"
	relation           = "administrator"
	all                = "all"
)

var thing = sdk.Thing{
	ID:   testsutil.GenerateUUID(&testing.T{}),
	Name: "testthing",
	Credentials: sdk.ClientCredentials{
		Secret: "secret",
	},
	DomainID: testsutil.GenerateUUID(&testing.T{}),
	Status:   mgclients.EnabledStatus.String(),
}

func TestCreateThingsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingJson := "{\"name\":\"testthing\", \"metadata\":{\"key1\":\"value1\"}}"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var tg sdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         sdk.Thing
		logType       outputLog
	}{
		{
			desc: "create thing successfully with token",
			args: []string{
				thingJson,
				domainID,
				token,
			},
			thing:   thing,
			logType: entityLog,
		},
		{
			desc: "create thing without token",
			args: []string{
				thingJson,
				domainID,
			},
			logType: usageLog,
		},
		{
			desc: "create thing with invalid token",
			args: []string{
				thingJson,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "failed to create thing",
			args: []string{
				thingJson,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "create thing with invalid metadata",
			args: []string{
				"{\"name\":\"testthing\", \"metadata\":{\"key1\":value1}}",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.New("invalid character 'v' looking for beginning of value"), 306),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character 'v' looking for beginning of value")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateThing", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.thing, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.thing, tg))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestGetThingsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var tg sdk.Thing
	var page sdk.ThingsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         sdk.Thing
		page          sdk.ThingsPage
		logType       outputLog
	}{
		{
			desc: "get all things successfully",
			args: []string{
				all,
				domainID,
				token,
			},
			logType: entityLog,
			page: sdk.ThingsPage{
				Things: []sdk.Thing{thing},
			},
		},
		{
			desc: "get thing successfully with id",
			args: []string{
				thing.ID,
				domainID,
				token,
			},
			logType: entityLog,
			thing:   thing,
		},
		{
			desc: "get things with invalid token",
			args: []string{
				all,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			page:          sdk.ThingsPage{},
			logType:       errLog,
		},
		{
			desc: "get things with invalid args",
			args: []string{
				all,
				invalidToken,
				all,
				invalidToken,
				all,
				invalidToken,
				all,
				invalidToken,
			},
			logType: usageLog,
		},
		{
			desc: "get thing without token",
			args: []string{
				all,
				domainID,
			},
			logType: usageLog,
		},
		{
			desc: "get thing with invalid thing id",
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
			sdkCall := sdkMock.On("Things", mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
			sdkCall1 := sdkMock.On("Thing", mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			if tc.logType == entityLog {
				switch {
				case tc.args[1] == all:
					err := json.Unmarshal([]byte(out), &page)
					if err != nil {
						t.Fatalf("Failed to unmarshal JSON: %v", err)
					}
				default:
					err := json.Unmarshal([]byte(out), &tg)
					if err != nil {
						t.Fatalf("Failed to unmarshal JSON: %v", err)
					}
				}
			}

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			if tc.logType == entityLog {
				if tc.args[1] != all {
					assert.Equal(t, tc.thing, tg, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.thing, tg))
				} else {
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				}
			}

			sdkCall.Unset()
			sdkCall1.Unset()
		})
	}
}

func TestUpdateThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	tagUpdateType := "tags"
	secretUpdateType := "secret"
	newTagsJson := "[\"tag1\", \"tag2\"]"
	newTagString := []string{"tag1", "tag2"}
	newNameandMeta := "{\"name\": \"thingName\", \"metadata\": {\"role\": \"general\"}}"
	newSecret := "secret"

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         sdk.Thing
		logType       outputLog
	}{
		{
			desc: "update thing name and metadata successfully",
			args: []string{
				thing.ID,
				newNameandMeta,
				domainID,
				token,
			},
			thing: sdk.Thing{
				Name: "thingName",
				Metadata: map[string]interface{}{
					"metadata": map[string]interface{}{
						"role": "general",
					},
				},
				ID:       thing.ID,
				DomainID: thing.DomainID,
				Status:   thing.Status,
			},
			logType: entityLog,
		},
		{
			desc: "update thing name and metadata with invalid json",
			args: []string{
				thing.ID,
				"{\"name\": \"thingName\", \"metadata\": {\"role\": \"general\"}",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update thing name and metadata with invalid thing id",
			args: []string{
				invalidID,
				newNameandMeta,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing tags successfully",
			args: []string{
				tagUpdateType,
				thing.ID,
				newTagsJson,
				domainID,
				token,
			},
			thing: sdk.Thing{
				Name:     thing.Name,
				ID:       thing.ID,
				DomainID: thing.DomainID,
				Status:   thing.Status,
				Tags:     newTagString,
			},
			logType: entityLog,
		},
		{
			desc: "update thing with invalid tags",
			args: []string{
				tagUpdateType,
				thing.ID,
				"[\"tag1\", \"tag2\"",
				domainID,
				token,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
		},
		{
			desc: "update thing tags with invalid thing id",
			args: []string{
				tagUpdateType,
				invalidID,
				newTagsJson,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing secret successfully",
			args: []string{
				secretUpdateType,
				thing.ID,
				newSecret,
				domainID,
				token,
			},
			thing: sdk.Thing{
				Name:     thing.Name,
				ID:       thing.ID,
				DomainID: thing.DomainID,
				Status:   thing.Status,
				Credentials: sdk.ClientCredentials{
					Secret: newSecret,
				},
			},
			logType: entityLog,
		},
		{
			desc: "update thing with invalid secret",
			args: []string{
				secretUpdateType,
				thing.ID,
				"",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "update thing with invalid token",
			args: []string{
				secretUpdateType,
				thing.ID,
				newSecret,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing with invalid args",
			args: []string{
				secretUpdateType,
				thing.ID,
				newSecret,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var tg sdk.Thing
			sdkCall := sdkMock.On("UpdateThing", mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
			sdkCall1 := sdkMock.On("UpdateThingTags", mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
			sdkCall2 := sdkMock.On("UpdateThingSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)

			switch {
			case tc.args[0] == tagUpdateType:
				var th sdk.Thing
				th.Tags = []string{"tag1", "tag2"}
				th.ID = tc.args[1]

				sdkCall1 = sdkMock.On("UpdateThingTags", th, tc.args[3]).Return(tc.thing, tc.sdkErr)
			case tc.args[0] == secretUpdateType:
				var th sdk.Thing
				th.Credentials.Secret = tc.args[2]
				th.ID = tc.args[1]

				sdkCall2 = sdkMock.On("UpdateThingSecret", th, tc.args[2], tc.args[3]).Return(tc.thing, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.thing, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.thing, tg))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			sdkCall.Unset()
			sdkCall1.Unset()
			sdkCall2.Unset()
		})
	}
}

func TestDeleteThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete thing successfully",
			args: []string{
				thing.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				thing.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing id",
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
			desc: "delete thing with invalid args",
			args: []string{
				thing.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteThing", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkErr)
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

func TestEnableThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)
	var tg sdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         sdk.Thing
		logType       outputLog
	}{
		{
			desc: "enable thing successfully",
			args: []string{
				thing.ID,
				domainID,
				validToken,
			},
			sdkErr:  nil,
			thing:   thing,
			logType: entityLog,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				thing.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing ID",
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
			desc: "enable thing with invalid args",
			args: []string{
				thing.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableThing", tc.args[0], tc.args[1], tc.args[2]).Return(tc.thing, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.thing, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.thing, tg))
			}

			sdkCall.Unset()
		})
	}
}

func TestDisablethingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var tg sdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         sdk.Thing
		logType       outputLog
	}{
		{
			desc: "disable thing successfully",
			args: []string{
				thing.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			thing:   thing,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				thing.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing ID",
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
			desc: "disable thing with invalid args",
			args: []string{
				thing.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableThing", tc.args[0], tc.args[1], tc.args[2]).Return(tc.thing, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{disableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				if err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				assert.Equal(t, tc.thing, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.thing, tg))
			}

			sdkCall.Unset()
		})
	}
}

func TestUsersThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	page := sdk.UsersPage{}

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		page          sdk.UsersPage
		sdkErr        errors.SDKError
	}{
		{
			desc: "get thing's users successfully",
			args: []string{
				thing.ID,
				domainID,
				token,
			},
			page: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Users: []sdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "list thing users' with invalid args",
			args: []string{
				thing.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list thing users' with invalid domain",
			args: []string{
				thing.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users with invalid id",
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
			sdkCall := sdkMock.On("ListThingUsers", mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
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

func TestConnectThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "Connect thing to channel successfully",
			args: []string{
				thing.ID,
				channel.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "connect with invalid args",
			args: []string{
				thing.ID,
				channel.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "connect with invalid thing id",
			args: []string{
				invalidID,
				channel.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "connect with invalid channel id",
			args: []string{
				thing.ID,
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users' with invalid domain",
			args: []string{
				thing.ID,
				channel.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Connect", mock.Anything, tc.args[2], tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{connCmd}, tc.args...)...)

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

func TestDisconnectThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "Disconnect thing to channel successfully",
			args: []string{
				thing.ID,
				channel.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "Disconnect with invalid args",
			args: []string{
				thing.ID,
				channel.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "disconnect with invalid thing id",
			args: []string{
				invalidID,
				channel.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "disconnect with invalid channel id",
			args: []string{
				thing.ID,
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disconnect thing with invalid domain",
			args: []string{
				thing.ID,
				channel.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Disconnect", mock.Anything, tc.args[2], tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{disconnCmd}, tc.args...)...)

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

func TestListConnectionCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cp := sdk.ChannelsPage{}
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		page          sdk.ChannelsPage
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "list connections successfully",
			args: []string{
				thing.ID,
				domainID,
				token,
			},
			page: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []sdk.Channel{channel},
			},
			logType: entityLog,
		},
		{
			desc: "list connections with invalid args",
			args: []string{
				thing.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list connections with invalid thing ID",
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
			desc: "list connections with invalid token",
			args: []string{
				thing.ID,
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
			sdkCall := sdkMock.On("ChannelsByThing", tc.args[0], mock.Anything, tc.args[1], tc.args[2]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{connsCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &cp)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				assert.Equal(t, tc.page, cp, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, cp))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestShareThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "share thing successfully",
			args: []string{
				thing.ID,
				user.ID,
				relation,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "share thing with invalid user id",
			args: []string{
				thing.ID,
				invalidID,
				relation,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "share thing with invalid thing ID",
			args: []string{
				invalidID,
				user.ID,
				relation,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "share thing with invalid args",
			args: []string{
				thing.ID,
				user.ID,
				relation,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "share thing with invalid relation",
			args: []string{
				thing.ID,
				user.ID,
				"invalid",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ShareThing", tc.args[0], mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{shrCmd}, tc.args...)...)

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

func TestUnshareThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "unshare thing successfully",
			args: []string{
				thing.ID,
				user.ID,
				relation,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unshare thing with invalid thing ID",
			args: []string{
				invalidID,
				user.ID,
				relation,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "unshare thing with invalid args",
			args: []string{
				thing.ID,
				user.ID,
				relation,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unshare thing with invalid relation",
			args: []string{
				thing.ID,
				user.ID,
				"invalid",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UnshareThing", tc.args[0], mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{unshrCmd}, tc.args...)...)

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
