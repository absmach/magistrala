// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/cli"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	sdkmocks "github.com/absmach/supermq/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	token    = "valid" + "domaintoken"
	domainID = "domain-id"
	relation = "administrator"
	all      = "all"
	conntype = `["publish","subscribe"]`
)

var client = smqsdk.Client{
	ID:   testsutil.GenerateUUID(&testing.T{}),
	Name: "testclient",
	Credentials: smqsdk.ClientCredentials{
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

	var tg smqsdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        smqsdk.Client
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

	var tg smqsdk.Client
	var page smqsdk.ClientsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        smqsdk.Client
		page          smqsdk.ClientsPage
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
			page: smqsdk.ClientsPage{
				Clients: []smqsdk.Client{client},
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
			page:          smqsdk.ClientsPage{},
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
		client        smqsdk.Client
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
			client: smqsdk.Client{
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
			client: smqsdk.Client{
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
			client: smqsdk.Client{
				Name:     client.Name,
				ID:       client.ID,
				DomainID: client.DomainID,
				Status:   client.Status,
				Credentials: smqsdk.ClientCredentials{
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
			var tg smqsdk.Client
			sdkCall := sdkMock.On("UpdateClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)
			sdkCall1 := sdkMock.On("UpdateClientTags", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)
			sdkCall2 := sdkMock.On("UpdateClientSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.sdkErr)

			switch {
			case tc.args[0] == tagUpdateType:
				var th smqsdk.Client
				th.Tags = []string{"tag1", "tag2"}
				th.ID = tc.args[1]

				sdkCall1 = sdkMock.On("UpdateClientTags", th, tc.args[3]).Return(tc.client, tc.sdkErr)
			case tc.args[0] == secretUpdateType:
				var th smqsdk.Client
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
	var tg smqsdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        smqsdk.Client
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

	var tg smqsdk.Client

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		client        smqsdk.Client
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
				conntype,
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
				conntype,
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
				conntype,
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
				conntype,
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
				conntype,
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
			sdkCall := sdkMock.On("Connect", mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
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
				conntype,
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
				conntype,
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
				conntype,
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
				conntype,
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
				conntype,
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
			sdkCall := sdkMock.On("Disconnect", mock.Anything, tc.args[3], tc.args[4]).Return(tc.sdkErr)
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

func TestCreateClientRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	roleReq := smqsdk.RoleReq{
		RoleName:        "admin",
		OptionalActions: []string{"read", "update"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		role          smqsdk.Role
		logType       outputLog
	}{
		{
			desc: "create client role successfully",
			args: []string{
				`{"role_name":"admin","optional_actions":["read","update"]}`,
				client.ID,
				domainID,
				token,
			},
			role: smqsdk.Role{
				ID:              testsutil.GenerateUUID(&testing.T{}),
				Name:            "admin",
				OptionalActions: []string{"read", "update"},
			},
			logType: entityLog,
		},
		{
			desc: "create client role with invalid JSON",
			args: []string{
				`{"role_name":"admin","optional_actions":["read","update"}`,
				client.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("invalid character '}' after array element")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character '}' after array element")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateClientRole", tc.args[1], tc.args[2], roleReq, tc.args[3]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "create"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var role smqsdk.Role
				err := json.Unmarshal([]byte(out), &role)
				assert.Nil(t, err)
				assert.Equal(t, tc.role, role, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.role, role))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestGetClientRolesCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	role := smqsdk.Role{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		Name:            "admin",
		OptionalActions: []string{"read", "update"},
	}
	rolesPage := smqsdk.RolesPage{
		Total:  1,
		Offset: 0,
		Limit:  10,
		Roles:  []smqsdk.Role{role},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		roles         smqsdk.RolesPage
		logType       outputLog
	}{
		{
			desc: "get all client roles successfully",
			args: []string{
				all,
				client.ID,
				domainID,
				token,
			},
			roles:   rolesPage,
			logType: entityLog,
		},
		{
			desc: "get client roles with invalid token",
			args: []string{
				all,
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ClientRoles", tc.args[1], tc.args[2], mock.Anything, tc.args[3]).Return(tc.roles, tc.sdkErr)
			if tc.args[0] != all {
				sdkCall = sdkMock.On("ClientRole", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(role, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, append([]string{"roles", "get"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var roles smqsdk.RolesPage
				err := json.Unmarshal([]byte(out), &roles)
				assert.Nil(t, err)
				assert.Equal(t, tc.roles, roles, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.roles, roles))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestUpdateClientRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	role := smqsdk.Role{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		Name:            "new_name",
		OptionalActions: []string{"read", "update"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		role          smqsdk.Role
		logType       outputLog
	}{
		{
			desc: "update client role name successfully",
			args: []string{
				"new_name",
				role.ID,
				client.ID,
				domainID,
				token,
			},
			role:    role,
			logType: entityLog,
		},
		{
			desc: "update client role name with invalid token",
			args: []string{
				"new_name",
				role.ID,
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdateClientRole", tc.args[2], tc.args[1], tc.args[0], tc.args[3], tc.args[4]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "update"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var role smqsdk.Role
				err := json.Unmarshal([]byte(out), &role)
				assert.Nil(t, err)
				assert.Equal(t, tc.role, role, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.role, role))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestDeleteClientRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete client role successfully",
			args: []string{
				roleID,
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete client role with invalid token",
			args: []string{
				roleID,
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteClientRole", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "delete"}, tc.args...)...)

			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestAddClientRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	actions := struct {
		Actions []string `json:"actions"`
	}{
		Actions: []string{"read", "write"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		actions       []string
		logType       outputLog
	}{
		{
			desc: "add actions to role successfully",
			args: []string{
				`{"actions":["read","write"]}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			actions: actions.Actions,
			logType: entityLog,
		},
		{
			desc: "add actions to role with invalid JSON",
			args: []string{
				`{"actions":["read","write"}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("invalid character '}' after array element")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character '}' after array element")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddClientRoleActions", tc.args[2], tc.args[1], tc.args[3], tc.actions, tc.args[4]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "actions", "add"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var acts []string
				err := json.Unmarshal([]byte(out), &acts)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, acts, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, acts))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestListClientRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	actions := []string{"read", "write"}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		actions       []string
		logType       outputLog
	}{
		{
			desc: "list actions of role successfully",
			args: []string{
				roleID,
				client.ID,
				domainID,
				token,
			},
			actions: actions,
			logType: entityLog,
		},
		{
			desc: "list actions of role with invalid token",
			args: []string{
				roleID,
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ClientRoleActions", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "actions", "list"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var acts []string
				err := json.Unmarshal([]byte(out), &acts)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, acts, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, acts))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestDeleteClientRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	actions := struct {
		Actions []string `json:"actions"`
	}{
		Actions: []string{"read", "write"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete actions from role successfully",
			args: []string{
				`{"actions":["read","write"]}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete all actions from role successfully",
			args: []string{
				all,
				roleID,
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete actions from role with invalid JSON",
			args: []string{
				`{"actions":["read","write"}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("invalid character '}' after array element")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character '}' after array element")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if tc.args[0] == all {
				sdkCall = sdkMock.On("RemoveAllClientRoleActions", tc.args[2], tc.args[1], tc.args[3], tc.args[4]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveClientRoleActions", tc.args[2], tc.args[1], tc.args[3], actions.Actions, tc.args[4]).Return(tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, append([]string{"roles", "actions", "delete"}, tc.args...)...)

			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestAvailableClientRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	actions := []string{"read", "write", "update"}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		actions       []string
		logType       outputLog
	}{
		{
			desc: "list available actions successfully",
			args: []string{
				domainID,
				token,
			},
			actions: actions,
			logType: entityLog,
		},
		{
			desc: "list available actions with invalid token",
			args: []string{
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AvailableClientRoleActions", tc.args[0], tc.args[1]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "actions", "available-actions"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var acts []string
				err := json.Unmarshal([]byte(out), &acts)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, acts, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, acts))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestAddClientRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	members := struct {
		Members []string `json:"members"`
	}{
		Members: []string{"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		members       []string
		logType       outputLog
	}{
		{
			desc: "add members to role successfully",
			args: []string{
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			members: members.Members,
			logType: entityLog,
		},
		{
			desc: "add members to role with invalid JSON",
			args: []string{
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("invalid character '}' after array element")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character '}' after array element")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddClientRoleMembers", tc.args[2], tc.args[1], tc.args[3], tc.members, tc.args[4]).Return(tc.members, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "members", "add"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var members []string
				err := json.Unmarshal([]byte(out), &members)
				assert.Nil(t, err)
				assert.Equal(t, tc.members, members, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.members, members))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestListClientRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	membersPage := smqsdk.RoleMembersPage{
		Total:  1,
		Offset: 0,
		Limit:  10,
		Members: []string{
			"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb",
		},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		members       smqsdk.RoleMembersPage
		logType       outputLog
	}{
		{
			desc: "list members of role successfully",
			args: []string{
				roleID,
				client.ID,
				domainID,
				token,
			},
			members: membersPage,
			logType: entityLog,
		},
		{
			desc: "list members of role with invalid token",
			args: []string{
				roleID,
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ClientRoleMembers", tc.args[1], tc.args[0], tc.args[2], mock.Anything, tc.args[3]).Return(tc.members, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"roles", "members", "list"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				var members smqsdk.RoleMembersPage
				err := json.Unmarshal([]byte(out), &members)
				assert.Nil(t, err)
				assert.Equal(t, tc.members, members, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.members, members))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestDeleteClientRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	clientsCmd := cli.NewClientsCmd()
	rootCmd := setFlags(clientsCmd)

	members := struct {
		Members []string `json:"members"`
	}{
		Members: []string{"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete members from role successfully",
			args: []string{
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete all members from role successfully",
			args: []string{
				all,
				roleID,
				client.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete members from role with invalid JSON",
			args: []string{
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}`,
				roleID,
				client.ID,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("invalid character '}' after array element")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("invalid character '}' after array element")),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if tc.args[0] == all {
				sdkCall = sdkMock.On("RemoveAllClientRoleMembers", tc.args[2], tc.args[1], tc.args[3], tc.args[4]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveClientRoleMembers", tc.args[2], tc.args[1], tc.args[3], members.Members, tc.args[4]).Return(tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, append([]string{"roles", "members", "delete"}, tc.args...)...)

			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}

			sdkCall.Unset()
		})
	}
}
