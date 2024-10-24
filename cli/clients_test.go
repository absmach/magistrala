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
	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
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

var client = sdk.Client{
	ID:   testsutil.GenerateUUID(&testing.T{}),
	Name: "testclient",
	Credentials: sdk.ClientCredentials{
		Secret: "secret",
	},
	DomainID: testsutil.GenerateUUID(&testing.T{}),
	Status:   clients.EnabledStatus.String(),
}

func TestCreateClientsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientJson := "{\"name\":\"testclient\", \"metadata\":{\"key1\":\"value1\"}}"
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	var tg sdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        sdk.Client
		logType       outputLog
	}{
		{
			desc: "create client successfully with token",
			args: []string{
				clientJson,
				domainID,
				token,
			},
			client:  client,
			logType: entityLog,
		},
		{
			desc: "create client without token",
			args: []string{
				clientJson,
				domainID,
			},
			logType: usageLog,
		},
		{
			desc: "create client with invalid token",
			args: []string{
				clientJson,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "failed to create client",
			args: []string{
				clientJson,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "create client with invalid metadata",
			args: []string{
				"{\"name\":\"testclient\", \"metadata\":{\"key1\":value1}}",
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
			sdkCall := sdkMock.On("CreateClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.client, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.client, tg))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestGetClientssCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	var tg sdk.Client
	var page sdk.ClientsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        sdk.Client
		page          sdk.ClientsPage
		logType       outputLog
	}{
		{
			desc: "get all clients successfully",
			args: []string{
				all,
				domainID,
				token,
			},
			logType: entityLog,
			page: sdk.ClientsPage{
				Clients: []sdk.Client{client},
			},
		},
		{
			desc: "get client successfully with id",
			args: []string{
				client.ID,
				domainID,
				token,
			},
			logType: entityLog,
			client:  client,
		},
		{
			desc: "get clients with invalid token",
			args: []string{
				all,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			page:          sdk.ClientsPage{},
			logType:       errLog,
		},
		{
			desc: "get clients with invalid args",
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
			desc: "get client without token",
			args: []string{
				all,
				domainID,
			},
			logType: usageLog,
		},
		{
			desc: "get client with invalid client id",
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
			sdkCall := sdkMock.On("Clients", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
			sdkCall1 := sdkMock.On("Client", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)

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
					assert.Equal(t, tc.client, tg, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.client, tg))
				} else {
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				}
			}

			sdkCall.Unset()
			sdkCall1.Unset()
		})
	}
}

func TestUpdateClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	tagUpdateType := "tags"
	secretUpdateType := "secret"
	newTagsJson := "[\"tag1\", \"tag2\"]"
	newTagString := []string{"tag1", "tag2"}
	newNameandMeta := "{\"name\": \"clientName\", \"metadata\": {\"role\": \"general\"}}"
	newSecret := "secret"

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        sdk.Client
		logType       outputLog
	}{
		{
			desc: "update client name and metadata successfully",
			args: []string{
				client.ID,
				newNameandMeta,
				domainID,
				token,
			},
			client: sdk.Client{
				Name: "clientName",
				Metadata: map[string]interface{}{
					"metadata": map[string]interface{}{
						"role": "general",
					},
				},
				ID:       client.ID,
				DomainID: client.DomainID,
				Status:   client.Status,
			},
			logType: entityLog,
		},
		{
			desc: "update client name and metadata with invalid json",
			args: []string{
				client.ID,
				"{\"name\": \"clientName\", \"metadata\": {\"role\": \"general\"}",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update client name and metadata with invalid client id",
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
			desc: "update client tags successfully",
			args: []string{
				tagUpdateType,
				client.ID,
				newTagsJson,
				domainID,
				token,
			},
			client: sdk.Client{
				Name:     client.Name,
				ID:       client.ID,
				DomainID: client.DomainID,
				Status:   client.Status,
				Tags:     newTagString,
			},
			logType: entityLog,
		},
		{
			desc: "update client with invalid tags",
			args: []string{
				tagUpdateType,
				client.ID,
				"[\"tag1\", \"tag2\"",
				domainID,
				token,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
		},
		{
			desc: "update client tags with invalid client id",
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
			desc: "update client secret successfully",
			args: []string{
				secretUpdateType,
				client.ID,
				newSecret,
				domainID,
				token,
			},
			client: sdk.Client{
				Name:     client.Name,
				ID:       client.ID,
				DomainID: client.DomainID,
				Status:   client.Status,
				Credentials: sdk.ClientCredentials{
					Secret: newSecret,
				},
			},
			logType: entityLog,
		},
		{
			desc: "update client  with invalid secret",
			args: []string{
				secretUpdateType,
				client.ID,
				"",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest)),
			logType:       errLog,
		},
		{
			desc: "update client  with invalid token",
			args: []string{
				secretUpdateType,
				client.ID,
				newSecret,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update client  with invalid args",
			args: []string{
				secretUpdateType,
				client.ID,
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
			var tg sdk.Client
			sdkCall := sdkMock.On("UpdateClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)
			sdkCall1 := sdkMock.On("UpdateClientTags", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)
			sdkCall2 := sdkMock.On("UpdateClientSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)

			switch {
			case tc.args[0] == tagUpdateType:
				var th sdk.Client
				th.Tags = []string{"tag1", "tag2"}
				th.ID = tc.args[1]

				sdkCall1 = sdkMock.On("UpdateClientTags", th, tc.args[3]).Return(tc.client, tc.sdkErr)
			case tc.args[0] == secretUpdateType:
				var th sdk.Client
				th.Credentials.Secret = tc.args[2]
				th.ID = tc.args[1]

				sdkCall2 = sdkMock.On("UpdateClientSecret", th, tc.args[2], tc.args[3]).Return(tc.client, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.client, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.client, tg))
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

func TestDeleteClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientdCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientdCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete client successfully",
			args: []string{
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete client with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete client with invalid client id",
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
			desc: "delete client with invalid args",
			args: []string{
				client.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteClient", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkErr)
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

func TestEnableClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)
	var tg sdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        sdk.Client
		logType       outputLog
	}{
		{
			desc: "enable client successfully",
			args: []string{
				client.ID,
				domainID,
				validToken,
			},
			sdkErr:  nil,
			client:  client,
			logType: entityLog,
		},
		{
			desc: "delete client with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete client with invalid client ID",
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
			desc: "enable client with invalid args",
			args: []string{
				client.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableClient", tc.args[0], tc.args[1], tc.args[2]).Return(tc.client, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &tg)
				assert.Nil(t, err)
				assert.Equal(t, tc.client, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.client, tg))
			}

			sdkCall.Unset()
		})
	}
}

func TestDisableclientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	var tg sdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        sdk.Client
		logType       outputLog
	}{
		{
			desc: "disable client successfully",
			args: []string{
				client.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			client:  client,
		},
		{
			desc: "delete client with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete client with invalid client ID",
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
			desc: "disable client with invalid args",
			args: []string{
				client.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableClient", tc.args[0], tc.args[1], tc.args[2]).Return(tc.client, tc.sdkErr)
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
				assert.Equal(t, tc.client, tg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.client, tg))
			}

			sdkCall.Unset()
		})
	}
}

func TestUsersClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

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
			desc: "get client's users successfully",
			args: []string{
				client.ID,
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
			desc: "list client users' with invalid args",
			args: []string{
				client.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list client users' with invalid domain",
			args: []string{
				client.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list client users with invalid id",
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
			sdkCall := sdkMock.On("ListClientUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
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

func TestConnectClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "Connect client to channel successfully",
			args: []string{
				client.ID,
				channel.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "connect with invalid args",
			args: []string{
				client.ID,
				channel.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "connect with invalid client id",
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
				client.ID,
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list client users' with invalid domain",
			args: []string{
				client.ID,
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

func TestDisconnectClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "Disconnect client to channel successfully",
			args: []string{
				client.ID,
				channel.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "Disconnect with invalid args",
			args: []string{
				client.ID,
				channel.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "disconnect with invalid client id",
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
				client.ID,
				invalidID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disconnect client with invalid domain",
			args: []string{
				client.ID,
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
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

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
				client.ID,
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
				client.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list connections with invalid client ID",
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
				client.ID,
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
			sdkCall := sdkMock.On("ChannelsByClient", tc.args[0], mock.Anything, tc.args[1], tc.args[2]).Return(tc.page, tc.sdkErr)
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

func TestShareClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "share client successfully",
			args: []string{
				client.ID,
				user.ID,
				relation,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "share client with invalid user id",
			args: []string{
				client.ID,
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
			desc: "share client with invalid client ID",
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
			desc: "share client with invalid args",
			args: []string{
				client.ID,
				user.ID,
				relation,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "share client with invalid relation",
			args: []string{
				client.ID,
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
			sdkCall := sdkMock.On("ShareClient", tc.args[0], mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
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

func TestUnshareClientCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "unshare client successfully",
			args: []string{
				client.ID,
				user.ID,
				relation,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unshare client with invalid client ID",
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
			desc: "unshare client with invalid args",
			args: []string{
				client.ID,
				user.ID,
				relation,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unshare client with invalid relation",
			args: []string{
				client.ID,
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
			sdkCall := sdkMock.On("UnshareClient", tc.args[0], mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
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
