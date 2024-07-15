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
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	token              = "valid" + "domaintoken"
	tokenWithoutDomain = "valid"
	relation           = "administrator"
	all                = "all"
)

var thing = mgsdk.Thing{
	ID:       testsutil.GenerateUUID(&testing.T{}),
	Name:     "testthing",
	DomainID: testsutil.GenerateUUID(&testing.T{}),
	Status:   mgclients.EnabledStatus.String(),
}

var channel = mgsdk.Channel{
	ID:       testsutil.GenerateUUID(&testing.T{}),
	Name:     "testchannel",
	DomainID: thing.DomainID,
}

func TestCreateThingsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	createCommand := "create"
	thingJson := "{\"name\":\"testthing\", \"metadata\":{\"key1\":\"value1\"}}"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var tg mgsdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         mgsdk.Thing
		logType       outputLog
	}{
		{
			desc: "create thing successfully with token",
			args: []string{
				createCommand,
				thingJson,
				token,
			},
			thing:   thing,
			logType: entityLog,
		},
		{
			desc: "create thing without token",
			args: []string{
				createCommand,
				thingJson,
			},
			logType: usageLog,
		},
		{
			desc: "create thing with invalid token",
			args: []string{
				createCommand,
				thingJson,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "failed to create thing",
			args: []string{
				createCommand,
				thingJson,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "create thing without domain token",
			args: []string{
				createCommand,
				thingJson,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "create thing with invalid metdata",
			args: []string{
				createCommand,
				"{\"name\":\"testthing\", \"metadata\":{\"key1\":value1}}",
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.New("invalid character 'v' looking for beginning of value"), 306),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character 'v' looking for beginning of value")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("CreateThing", mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestGetThingsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	getCommand := "get"

	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var tg mgsdk.Thing
	var page mgsdk.ThingsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         mgsdk.Thing
		page          mgsdk.ThingsPage
		logType       outputLog
	}{
		{
			desc: "get things successfully",
			args: []string{
				getCommand,
				all,
				token,
			},
			logType: entityLog,
			page: mgsdk.ThingsPage{
				Things: []mgsdk.Thing{thing},
			},
		},
		{
			desc: "get things successfully with id",
			args: []string{
				getCommand,
				thing.ID,
				token,
			},
			logType: entityLog,
			thing:   thing,
		},
		{
			desc: "get things with invalid token",
			args: []string{
				getCommand,
				all,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			page:          mgsdk.ThingsPage{},
			logType:       errLog,
		},
		{
			desc: "get thing without domain token",
			args: []string{
				getCommand,
				all,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "get things with invalid args",
			args: []string{
				getCommand,
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
				getCommand,
				all,
			},
			logType: usageLog,
		},
		{
			desc: "get thing with invalid thing id",
			args: []string{
				getCommand,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("Things", mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
		sdkCall1 := sdkMock.On("Thing", mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)

		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestUpdateThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	updateCommand := "update"
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
		thing         mgsdk.Thing
		logType       outputLog
	}{
		{
			desc: "update thing name and metadata successfully",
			args: []string{
				updateCommand,
				thing.ID,
				newNameandMeta,
				token,
			},
			thing: mgsdk.Thing{
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
				updateCommand,
				thing.ID,
				"{\"name\": \"thingName\", \"metadata\": {\"role\": \"general\"}",
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update thing name and metadata with invalid thing id",
			args: []string{
				updateCommand,
				invalidID,
				newNameandMeta,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing tags successfully",
			args: []string{
				updateCommand,
				tagUpdateType,
				thing.ID,
				newTagsJson,
				token,
			},
			thing: mgsdk.Thing{
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
				updateCommand,
				tagUpdateType,
				thing.ID,
				"[\"tag1\", \"tag2\"",
				token,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
		},
		{
			desc: "update thing tags with invalid thing id",
			args: []string{
				updateCommand,
				tagUpdateType,
				invalidID,
				newTagsJson,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing secret successfully",
			args: []string{
				updateCommand,
				secretUpdateType,
				thing.ID,
				newSecret,
				token,
			},
			thing: mgsdk.Thing{
				Name:     thing.Name,
				ID:       thing.ID,
				DomainID: thing.DomainID,
				Status:   thing.Status,
				Credentials: sdk.Credentials{
					Secret: newSecret,
				},
			},
			logType: entityLog,
		},
		{
			desc: "update thing with invalid secret",
			args: []string{
				updateCommand,
				secretUpdateType,
				thing.ID,
				"",
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "update thing with invalid token",
			args: []string{
				updateCommand,
				secretUpdateType,
				thing.ID,
				newSecret,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update thing with invalid args",
			args: []string{
				updateCommand,
				secretUpdateType,
				thing.ID,
				newSecret,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		var tg mgsdk.Thing
		sdkCall := sdkMock.On("UpdateThing", mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
		sdkCall1 := sdkMock.On("UpdateThingTags", mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)
		sdkCall2 := sdkMock.On("UpdateThingSecret", mock.Anything, mock.Anything, mock.Anything).Return(tc.thing, tc.sdkErr)

		switch {
		case tc.args[1] == tagUpdateType:
			var th mgsdk.Thing
			th.Tags = []string{"tag1", "tag2"}
			th.ID = tc.args[2]

			sdkCall1 = sdkMock.On("UpdateThingTags", th, tc.args[4]).Return(tc.thing, tc.sdkErr)
		case tc.args[1] == secretUpdateType:
			var th mgsdk.Thing
			th.Credentials.Secret = tc.args[3]
			th.ID = tc.args[2]

			sdkCall2 = sdkMock.On("UpdateThingSecret", th, tc.args[3], tc.args[4]).Return(tc.thing, tc.sdkErr)
		}
		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestDeleteThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	deleteCommand := "delete"
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
				deleteCommand,
				thing.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				deleteCommand,
				thing.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing id",
			args: []string{
				deleteCommand,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid args",
			args: []string{
				deleteCommand,
				thing.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("DeleteThing", mock.Anything, mock.Anything).Return(tc.sdkErr)
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
	}
}

func TestEnableThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	enableCommand := "enable"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)
	var tg mgsdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         mgsdk.Thing
		logType       outputLog
	}{
		{
			desc: "enable thing successfully",
			args: []string{
				enableCommand,
				thing.ID,
				validToken,
			},
			sdkErr:  nil,
			thing:   thing,
			logType: entityLog,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				enableCommand,
				thing.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing ID",
			args: []string{
				enableCommand,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "enable thing with invalid args",
			args: []string{
				enableCommand,
				thing.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("EnableThing", tc.args[1], tc.args[2]).Return(tc.thing, tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestDisablethingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	disableCommand := "disable"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	var usr mgsdk.Thing

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		thing         mgsdk.Thing
		logType       outputLog
	}{
		{
			desc: "disable thing successfully",
			args: []string{
				disableCommand,
				thing.ID,
				validToken,
			},
			logType: entityLog,
			thing:   thing,
		},
		{
			desc: "delete thing with invalid token",
			args: []string{
				disableCommand,
				thing.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete thing with invalid thing ID",
			args: []string{
				disableCommand,
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
				disableCommand,
				thing.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("DisableThing", tc.args[1], tc.args[2]).Return(tc.thing, tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &usr)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			assert.Equal(t, tc.thing, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.thing, usr))
		}

		sdkCall.Unset()
	}
}

func TestUsersthingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCommand := "users"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

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
			desc: "get thing's users successfully",
			args: []string{
				usersCommand,
				thing.ID,
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
			desc: "list thing users' with invalid args",
			args: []string{
				usersCommand,
				thing.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list thing users' without domain token",
			args: []string{
				usersCommand,
				thing.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users' with invalid thing ID",
			args: []string{
				usersCommand,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users' without domain token",
			args: []string{
				usersCommand,
				thing.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("ListThingUsers", mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestConnectThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	connectCommand := "connect"
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
				connectCommand,
				thing.ID,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "connect with invalid args",
			args: []string{
				connectCommand,
				thing.ID,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "connect with invalid thing id",
			args: []string{
				connectCommand,
				invalidID,
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "connect with invalid channel id",
			args: []string{
				connectCommand,
				thing.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users' without domain token",
			args: []string{
				connectCommand,
				thing.ID,
				channel.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("Connect", mock.Anything, mock.Anything).Return(tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case okLog:
			assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		}
		sdkCall.Unset()
	}
}

func TestDisconnectThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	disconnectCommand := "disconnect"
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
				disconnectCommand,
				thing.ID,
				channel.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "Disconnect with invalid args",
			args: []string{
				disconnectCommand,
				thing.ID,
				channel.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "disconnect with invalid thing id",
			args: []string{
				disconnectCommand,
				invalidID,
				channel.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "disconnect with invalid channel id",
			args: []string{
				disconnectCommand,
				thing.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list thing users' without domain token",
			args: []string{
				disconnectCommand,
				thing.ID,
				channel.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("Disconnect", mock.Anything, mock.Anything).Return(tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case okLog:
			assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		}
		sdkCall.Unset()
	}
}

func TestConnectionCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	connectionsCmd := "connections"
	thingsCmd := cli.NewThingsCmd()
	rootCmd := setFlags(thingsCmd)

	cp := mgsdk.ChannelsPage{}
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		page          mgsdk.ChannelsPage
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "list connections successfully",
			args: []string{
				connectionsCmd,
				thing.ID,
				token,
			},
			page: mgsdk.ChannelsPage{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Channels: []mgsdk.Channel{channel},
			},
			logType: entityLog,
		},
		{
			desc: "list connections with invalid args",
			args: []string{
				connectionsCmd,
				thing.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list connections with invalid thing ID",
			args: []string{
				connectionsCmd,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list connections with invalid token",
			args: []string{
				connectionsCmd,
				thing.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		sdkCall := sdkMock.On("ChannelsByThing", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

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
	}
}

func TestShareThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	shareCmd := "share"
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
				shareCmd,
				thing.ID,
				user.ID,
				relation,
				token,
			},
			logType: okLog,
		},
		{
			desc: "share thing with invalid user id",
			args: []string{
				shareCmd,
				thing.ID,
				invalidID,
				relation,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "share thing with invalid thing ID",
			args: []string{
				shareCmd,
				invalidID,
				user.ID,
				relation,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "share thing with invalid args",
			args: []string{
				shareCmd,
				thing.ID,
				user.ID,
				relation,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "share thing with invalid relation",
			args: []string{
				shareCmd,
				thing.ID,
				user.ID,
				"invalid",
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		sdkCall := sdkMock.On("ShareThing", tc.args[1], mock.Anything, tc.args[4]).Return(tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case okLog:
			assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		}
		sdkCall.Unset()
	}
}

func TestUnshareThingCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	unshareCmd := "unshare"
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
				unshareCmd,
				thing.ID,
				user.ID,
				relation,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unshare thing with invalid thing ID",
			args: []string{
				unshareCmd,
				invalidID,
				user.ID,
				relation,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "unshare thing with invalid args",
			args: []string{
				unshareCmd,
				thing.ID,
				user.ID,
				relation,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unshare thing with invalid relation",
			args: []string{
				unshareCmd,
				thing.ID,
				user.ID,
				"invalid",
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest)),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		sdkCall := sdkMock.On("UnshareThing", tc.args[1], mock.Anything, tc.args[4]).Return(tc.sdkErr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case okLog:
			assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		}
		sdkCall.Unset()
	}
}
