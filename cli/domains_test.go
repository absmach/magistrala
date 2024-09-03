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

var domain = mgsdk.Domain{
	ID:    testsutil.GenerateUUID(&testing.T{}),
	Name:  "Test domain",
	Alias: "alias",
}

func TestCreateDomainsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainCmd)

	var dom mgsdk.Domain

	cases := []struct {
		desc          string
		args          []string
		domain        mgsdk.Domain
		errLogMessage string
		sdkErr        errors.SDKError
		logType       outputLog
	}{
		{
			desc: "create domain successfully",
			args: []string{
				dom.Name,
				dom.Alias,
				validToken,
			},
			logType: entityLog,
			domain:  domain,
		},
		{
			desc: "create domain with invalid args",
			args: []string{
				dom.Name,
				dom.Alias,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create domain with invalid token",
			args: []string{
				dom.Name,
				dom.Alias,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateDomain", mock.Anything, mock.Anything).Return(tc.domain, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &dom)
				assert.Nil(t, err)
				assert.Equal(t, tc.domain, dom, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.domain, dom))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestGetDomainsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	all := "all"
	domainCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainCmd)

	var dom mgsdk.Domain
	var page mgsdk.DomainsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          mgsdk.DomainsPage
		domain        mgsdk.Domain
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get all domains successfully",
			args: []string{
				all,
				validToken,
			},
			page: mgsdk.DomainsPage{
				Domains: []mgsdk.Domain{domain},
			},
			logType: entityLog,
		},
		{
			desc: "get domain with id",
			args: []string{
				domain.ID,
				validToken,
			},
			logType: entityLog,
			domain:  domain,
		},
		{
			desc: "get domains with invalid args",
			args: []string{
				all,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all domains with invalid token",
			args: []string{
				all,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get domain with invalid id",
			args: []string{
				invalidID,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Domain", tc.args[0], tc.args[1]).Return(tc.domain, tc.sdkErr)
			sdkCall1 := sdkMock.On("Domains", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				if tc.args[1] == all {
					err := json.Unmarshal([]byte(out), &page)
					assert.Nil(t, err)
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				} else {
					err := json.Unmarshal([]byte(out), &dom)
					assert.Nil(t, err)
					assert.Equal(t, tc.domain, dom, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.domain, dom))
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

func TestListDomainUsers(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

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
			desc: "list domain users successfully",
			args: []string{
				domain.ID,
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
			desc: "list domain users with invalid args",
			args: []string{
				domain.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list domain users without domain token",
			args: []string{
				domain.ID,
				tokenWithoutDomain,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "list domain users with invalid id",
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
			sdkCall := sdkMock.On("ListDomainUsers", mock.Anything, tc.args[1]).Return(tc.page, tc.sdkErr)
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

func TestUpdateDomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	newDomainJson := "{\"name\" : \"New domain\"}"
	cases := []struct {
		desc          string
		args          []string
		domain        mgsdk.Domain
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "update domain successfully",
			args: []string{
				domain.ID,
				newDomainJson,
				token,
			},
			domain: mgsdk.Domain{
				Name: "New domain",
				ID:   domain.ID,
			},
			logType: entityLog,
		},
		{
			desc: "update domain with invalid args",
			args: []string{
				domain.ID,
				newDomainJson,
				token,
				extraArg,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "update domain with invalid id",
			args: []string{
				invalidID,
				newDomainJson,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update domain with invalid json syntax",
			args: []string{
				domain.ID,
				"{\"name\" : \"New domain\"",
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var dom mgsdk.Domain
			sdkCall := sdkMock.On("UpdateDomain", mock.Anything, tc.args[2]).Return(tc.domain, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &dom)
				assert.Nil(t, err)
				assert.Equal(t, tc.domain, dom, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.domain, dom))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestEnableDomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "enable domain successfully",
			args: []string{
				domain.ID,
				validToken,
			},
			logType: entityLog,
		},
		{
			desc: "enable domain with invalid token",
			args: []string{
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "enable domain with invalid domain id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "enable domain with invalid args",
			args: []string{
				domain.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableDomain", tc.args[0], tc.args[1]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestDisableDomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "disable domain successfully",
			args: []string{
				domain.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "disable domain with invalid token",
			args: []string{
				domain.ID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable domain with invalid id",
			args: []string{
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable domain with invalid args",
			args: []string{
				domain.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableDomain", tc.args[0], tc.args[1]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{disableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestAssignUserToDomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	userIds := fmt.Sprintf("[\"%s\"]", user.ID)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "assign user successfully",
			args: []string{
				relation,
				userIds,
				domain.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "assign user with invalid args",
			args: []string{
				relation,
				userIds,
				domain.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "assign user with invalid json",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"", user.ID),
				domain.ID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "assign user with invalid domain id",
			args: []string{
				relation,
				userIds,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "assign user with invalid user id",
			args: []string{
				relation,
				fmt.Sprintf("[\"%s\"]", invalidID),
				domain.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddUserToDomain", tc.args[2], mock.Anything, tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{assignCmd, usrCmd}, tc.args...)...)
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

func TestUnassignUserTodomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
	}{
		{
			desc: "unassign user successfully",
			args: []string{
				user.ID,
				domain.ID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "unassign user with invalid args",
			args: []string{
				user.ID,
				domain.ID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "unassign user with invalid domain id",
			args: []string{
				user.ID,
				invalidID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "unassign user with invalid user id",
			args: []string{
				invalidID,
				domain.ID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAddPolicies, http.StatusBadRequest)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RemoveUserFromDomain", tc.args[1], tc.args[0], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{unassignCmd, usrCmd}, tc.args...)...)
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
