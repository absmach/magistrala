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
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	domain = smqsdk.Domain{
		ID:    testsutil.GenerateUUID(&testing.T{}),
		Name:  "Test domain",
		Route: "route",
	}
	roleID = testsutil.GenerateUUID(&testing.T{})
)

func TestCreateDomainsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainCmd)

	var dom smqsdk.Domain

	cases := []struct {
		desc          string
		args          []string
		domain        smqsdk.Domain
		errLogMessage string
		sdkErr        errors.SDKError
		logType       outputLog
	}{
		{
			desc: "create domain successfully",
			args: []string{
				createCmd,
				dom.Name,
				dom.Route,
				validToken,
			},
			logType: entityLog,
			domain:  domain,
		},
		{
			desc: "create domain with invalid args",
			args: []string{
				createCmd,
				dom.Name,
				dom.Route,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create domain with invalid token",
			args: []string{
				createCmd,
				dom.Name,
				dom.Route,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateDomain", mock.Anything, mock.Anything, mock.Anything).Return(tc.domain, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &dom)
				assert.Nil(t, err)
				assert.Equal(t, tc.domain, dom, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.domain, dom))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.True(t, strings.Contains(out, "cli domains create"), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
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

	var dom smqsdk.Domain
	var page smqsdk.DomainsPage

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		page          smqsdk.DomainsPage
		domain        smqsdk.Domain
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "get all domains successfully",
			args: []string{
				all,
				getCmd,
				validToken,
			},
			page: smqsdk.DomainsPage{
				Domains: []smqsdk.Domain{domain},
			},
			logType: entityLog,
		},
		{
			desc: "get domain with id",
			args: []string{
				domain.ID,
				getCmd,
				validToken,
			},
			logType: entityLog,
			domain:  domain,
		},
		{
			desc: "get domains with invalid args",
			args: []string{
				all,
				getCmd,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get all domains with invalid token",
			args: []string{
				all,
				getCmd,
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
				getCmd,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Domain", mock.Anything, tc.args[0], tc.args[2]).Return(tc.domain, tc.sdkErr)
			sdkCall1 := sdkMock.On("Domains", mock.Anything, mock.Anything, tc.args[2]).Return(tc.page, tc.sdkErr)

			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				if tc.args[0] == all {
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

func TestUpdateDomainCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	newDomainJson := "{\"name\" : \"New domain\"}"
	cases := []struct {
		desc          string
		args          []string
		domain        smqsdk.Domain
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "update domain successfully",
			args: []string{
				domain.ID,
				updateCmd,
				newDomainJson,
				token,
			},
			domain: smqsdk.Domain{
				Name: "New domain",
				ID:   domain.ID,
			},
			logType: entityLog,
		},
		{
			desc: "update domain with invalid args",
			args: []string{
				domain.ID,
				updateCmd,
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
				updateCmd,
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
				updateCmd,
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
			var dom smqsdk.Domain
			sdkCall := sdkMock.On("UpdateDomain", mock.Anything, mock.Anything, tc.args[3]).Return(tc.domain, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				enableCmd,
				validToken,
			},
			logType: entityLog,
		},
		{
			desc: "enable domain with invalid token",
			args: []string{
				domain.ID,
				enableCmd,
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
				enableCmd,
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
				enableCmd,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableDomain", mock.Anything, tc.args[0], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				disableCmd,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "disable domain with invalid token",
			args: []string{
				domain.ID,
				disableCmd,
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
				disableCmd,
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
				disableCmd,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableDomain", mock.Anything, tc.args[0], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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

func TestFreezeDomainCmd(t *testing.T) {
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
			desc: "freeze domain successfully",
			args: []string{
				domain.ID,
				freezeCmd,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "freeze domain with invalid token",
			args: []string{
				domain.ID,
				freezeCmd,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "freeze domain with invalid id",
			args: []string{
				invalidID,
				freezeCmd,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "freeze domain with invalid args",
			args: []string{
				domain.ID,
				freezeCmd,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("FreezeDomain", mock.Anything, tc.args[0], tc.args[2]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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

func TestCreateDomainRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	roleReq := smqsdk.RoleReq{
		RoleName:        "admin",
		OptionalActions: []string{"read", "update"},
	}
	roleReqJson, err := json.Marshal(roleReq)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %v", err))

	role := smqsdk.Role{
		ID:   roleID,
		Name: "admin",
	}

	cases := []struct {
		desc          string
		args          []string
		roleReq       smqsdk.RoleReq
		role          smqsdk.Role
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "create role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				createCmd,
				string(roleReqJson),
				token,
			},
			role:    role,
			roleReq: roleReq,
			logType: entityLog,
		},
		{
			desc: "create role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				createCmd,
				string(roleReqJson),
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				createCmd,
				string(roleReqJson),
				invalidToken,
			},
			roleReq:       roleReq,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateDomainRole", mock.Anything, tc.args[0], tc.roleReq, tc.args[4]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var resp smqsdk.Role
				err := json.Unmarshal([]byte(out), &resp)
				assert.Nil(t, err)
				assert.Equal(t, tc.role, resp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.roleReq, role))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestGetDomainRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	role := smqsdk.Role{
		ID:   roleID,
		Name: "admin",
	}

	cases := []struct {
		desc          string
		args          []string
		role          smqsdk.Role
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "get role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				getCmd,
				roleID,
				token,
			},
			role:    role,
			logType: entityLog,
		},
		{
			desc: "get role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				getCmd,
				roleID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				getCmd,
				roleID,
				invalidToken,
			},
			role:          role,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DomainRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var role smqsdk.Role
				err := json.Unmarshal([]byte(out), &role)
				assert.Nil(t, err)
				assert.Equal(t, tc.role, role, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.role, role))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestUpdateDomainRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	newRoleName := "new_name"
	role := smqsdk.Role{
		ID:   roleID,
		Name: newRoleName,
	}

	cases := []struct {
		desc          string
		args          []string
		role          smqsdk.Role
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "update role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				updateCmd,
				roleID,
				newRoleName,
				token,
			},
			role:    role,
			logType: entityLog,
		},
		{
			desc: "update role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				updateCmd,
				roleID,
				newRoleName,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "update role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				updateCmd,
				roleID,
				newRoleName,
				invalidToken,
			},
			role:          role,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdateDomainRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4], tc.args[5]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var role smqsdk.Role
				err := json.Unmarshal([]byte(out), &role)
				assert.Nil(t, err)
				assert.Equal(t, tc.role, role, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.role, role))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestDeleteDomainRoleCmd(t *testing.T) {
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
			desc: "delete role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				delCmd,
				roleID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				delCmd,
				roleID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				delCmd,
				roleID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteDomainRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4]).Return(tc.sdkErr)
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
		})
	}
}

func TestAddDomainRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		actions       []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "add actions to role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				addCmd,
				roleID,
				`{"actions":["read","write"]}`,
				token,
			},
			actions: []string{"read", "write"},
			logType: entityLog,
		},
		{
			desc: "add actions to role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				addCmd,
				roleID,
				`{"actions":["read","write"]}`,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "add actions to role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				addCmd,
				roleID,
				`{"actions":["read","write"]}`,
				invalidToken,
			},
			actions:       []string{"read", "write"},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddDomainRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.actions, tc.args[6]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var actions []string
				err := json.Unmarshal([]byte(out), &actions)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, actions, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, actions))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestListDomainRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		actions       []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "list actions of role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				listCmd,
				roleID,
				token,
			},
			actions: []string{"read", "write"},
			logType: entityLog,
		},
		{
			desc: "list actions of role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				listCmd,
				roleID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list actions of role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				listCmd,
				roleID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DomainRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[5]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var actions []string
				err := json.Unmarshal([]byte(out), &actions)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, actions, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, actions))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestDeleteDomainRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		actions       []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete actions from role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				`{"actions":["read","write"]}`,
				token,
			},
			actions: []string{"read", "write"},
			logType: okLog,
		},
		{
			desc: "delete all actions from role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				all,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete actions from role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				`{"actions":["read","write"]}`,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete actions from role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				`{"actions":["read","write"]}`,
				invalidToken,
			},
			actions:       []string{"read", "write"},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if tc.args[5] == all {
				sdkCall = sdkMock.On("RemoveAllDomainRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[6]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveDomainRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.actions, tc.args[6]).Return(tc.sdkErr)
			}
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
		})
	}
}

func TestAvailableDomainRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	cases := []struct {
		desc          string
		args          []string
		actions       []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "list available actions successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				availableActionsCmd,
				token,
			},
			actions: []string{"read", "write", "update"},
			logType: entityLog,
		},
		{
			desc: "list available actions with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				availableActionsCmd,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list available actions with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				actionsCmd,
				availableActionsCmd,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AvailableDomainRoleActions", mock.Anything, tc.args[4]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var actions []string
				err := json.Unmarshal([]byte(out), &actions)
				assert.Nil(t, err)
				assert.Equal(t, tc.actions, actions, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.actions, actions))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestAddDomainRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	members := []string{"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}
	membersJson := `{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`

	cases := []struct {
		desc          string
		args          []string
		members       []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "add members to role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				addCmd,
				roleID,
				membersJson,
				token,
			},
			members: members,
			logType: entityLog,
		},
		{
			desc: "add members to role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				addCmd,
				roleID,
				membersJson,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "add members to role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				addCmd,
				roleID,
				membersJson,
				invalidToken,
			},
			members:       members,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("AddDomainRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.members, tc.args[6]).Return(tc.members, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var members []string
				err := json.Unmarshal([]byte(out), &members)
				assert.Nil(t, err)
				assert.Equal(t, tc.members, members, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.members, members))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestListDomainRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	page := smqsdk.RoleMembersPage{
		Total:   1,
		Offset:  0,
		Limit:   10,
		Members: []string{"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"},
	}

	cases := []struct {
		desc          string
		args          []string
		page          smqsdk.RoleMembersPage
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "list members of role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				listCmd,
				roleID,
				token,
			},
			page:    page,
			logType: entityLog,
		},
		{
			desc: "list members of role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				listCmd,
				roleID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list members of role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				listCmd,
				roleID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DomainRoleMembers", mock.Anything, tc.args[0], tc.args[4], mock.Anything, tc.args[5]).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var page smqsdk.RoleMembersPage
				err := json.Unmarshal([]byte(out), &page)
				assert.Nil(t, err)
				assert.Equal(t, tc.page, page, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, page))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestDeleteDomainRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCmd := cli.NewDomainsCmd()
	rootCmd := setFlags(domainsCmd)

	members := []string{"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}
	membersJson := `{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`

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
				domain.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				membersJson,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete all members from role successfully",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				all,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete members from role with invalid args",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				membersJson,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete members from role with invalid token",
			args: []string{
				domain.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				membersJson,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if tc.args[5] == all {
				sdkCall = sdkMock.On("RemoveAllDomainRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.args[6]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveDomainRoleMembers", mock.Anything, tc.args[0], tc.args[4], members, tc.args[6]).Return(tc.sdkErr)
			}
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
		})
	}
}
