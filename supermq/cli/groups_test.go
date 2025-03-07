// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/absmach/supermq/cli"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	sdkmocks "github.com/absmach/supermq/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var group = smqsdk.Group{
	ID:   testsutil.GenerateUUID(&testing.T{}),
	Name: "testgroup",
}

func TestCreateGroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupJson := "{\"name\":\"testgroup\", \"metadata\":{\"key1\":\"value1\"}}"
	groupCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupCmd)

	gp := smqsdk.Group{}
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		group         smqsdk.Group
		sdkErr        errors.SDKError
		errLogMessage string
	}{
		{
			desc: "create group successfully",
			args: []string{
				groupJson,
				domainID,
				token,
			},
			group:   group,
			logType: entityLog,
		},
		{
			desc: "create group with invalid args",
			args: []string{
				groupJson,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "create group with invalid json",
			args: []string{
				"{\"name\":\"testgroup\", \"metadata\":{\"key1\":\"value1\"}",
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "create group with invalid token",
			args: []string{
				groupJson,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "create group with invalid domain",
			args: []string{
				groupJson,
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrDomainAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateGroup", mock.Anything, tc.args[1], tc.args[2]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &gp)
				assert.Nil(t, err)
				assert.Equal(t, tc.group, gp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.group, gp))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestDeletegroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		logType       outputLog
		errLogMessage string
	}{
		{
			desc: "delete group successfully",
			args: []string{
				group.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete group with invalid args",
			args: []string{
				group.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete group with invalid id",
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
			desc: "delete group with invalid token",
			args: []string{
				group.ID,
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
			sdkCall := sdkMock.On("DeleteGroup", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkErr)
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

func TestUpdategroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupCmd)

	newGroupJson := fmt.Sprintf("{\"id\":\"%s\",\"name\" : \"newgroup\"}", group.ID)
	cases := []struct {
		desc          string
		args          []string
		group         smqsdk.Group
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "update group successfully",
			args: []string{
				newGroupJson,
				domainID,
				token,
			},
			group: smqsdk.Group{
				Name: "newgroup1",
				ID:   group.ID,
			},
			logType: entityLog,
		},
		{
			desc: "update group with invalid args",
			args: []string{
				newGroupJson,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "update group with invalid group id",
			args: []string{
				fmt.Sprintf("{\"id\":\"%s\",\"name\" : \"group1\"}", invalidID),
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "update group with invalid json syntax",
			args: []string{
				fmt.Sprintf("{\"id\":\"%s\",\"name\" : \"group1\"", group.ID),
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var ch smqsdk.Group
			sdkCall := sdkMock.On("UpdateGroup", mock.Anything, tc.args[1], tc.args[2]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				assert.Nil(t, err)
				assert.Equal(t, tc.group, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.group, ch))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestEnablegroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupCmd)
	var ch smqsdk.Group

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		group         smqsdk.Group
		logType       outputLog
	}{
		{
			desc: "enable group successfully",
			args: []string{
				group.ID,
				domainID,
				validToken,
			},
			group:   group,
			logType: entityLog,
		},
		{
			desc: "delete group with invalid token",
			args: []string{
				group.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "delete group with invalid group ID",
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
			desc: "enable group with invalid args",
			args: []string{
				group.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableGroup", tc.args[0], tc.args[1], tc.args[2]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				assert.Nil(t, err)
				assert.Equal(t, tc.group, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.group, ch))
			}

			sdkCall.Unset()
		})
	}
}

func TestDisablegroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

	var ch smqsdk.Group

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		group         smqsdk.Group
		logType       outputLog
	}{
		{
			desc: "disable group successfully",
			args: []string{
				group.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			group:   group,
		},
		{
			desc: "disable group with invalid token",
			args: []string{
				group.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
		{
			desc: "disable group with invalid id",
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
			desc: "disable group with invalid args",
			args: []string{
				group.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableGroup", tc.args[0], tc.args[1], tc.args[2]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{disableCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &ch)
				if err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				assert.Equal(t, tc.group, ch, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.group, ch))
			}

			sdkCall.Unset()
		})
	}
}

func TestCreateGroupRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
			desc: "create group role successfully",
			args: []string{
				`{"role_name":"admin","optional_actions":["read","update"]}`,
				group.ID,
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
			desc: "create group role with invalid JSON",
			args: []string{
				`{"role_name":"admin","optional_actions":["read","update"}`,
				group.ID,
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
			sdkCall := sdkMock.On("CreateGroupRole", tc.args[1], tc.args[2], roleReq, tc.args[3]).Return(tc.role, tc.sdkErr)
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

func TestGetGroupRolesCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
			desc: "get all group roles successfully",
			args: []string{
				all,
				group.ID,
				domainID,
				token,
			},
			roles:   rolesPage,
			logType: entityLog,
		},
		{
			desc: "get group roles with invalid token",
			args: []string{
				all,
				group.ID,
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
			sdkCall := sdkMock.On("GroupRoles", tc.args[1], tc.args[2], mock.Anything, tc.args[3]).Return(tc.roles, tc.sdkErr)
			if tc.args[0] != all {
				sdkCall = sdkMock.On("GroupRole", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(role, tc.sdkErr)
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

func TestUpdateGroupRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
			desc: "update group role name successfully",
			args: []string{
				"new_name",
				role.ID,
				group.ID,
				domainID,
				token,
			},
			role:    role,
			logType: entityLog,
		},
		{
			desc: "update group role name with invalid token",
			args: []string{
				"new_name",
				role.ID,
				group.ID,
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
			sdkCall := sdkMock.On("UpdateGroupRole", tc.args[2], tc.args[1], tc.args[0], tc.args[3], tc.args[4]).Return(tc.role, tc.sdkErr)
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

func TestDeleteGroupRoleCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete group role successfully",
			args: []string{
				roleID,
				group.ID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete group role with invalid token",
			args: []string{
				roleID,
				group.ID,
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
			sdkCall := sdkMock.On("DeleteGroupRole", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(tc.sdkErr)
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

func TestAddGroupRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
			sdkCall := sdkMock.On("AddGroupRoleActions", tc.args[2], tc.args[1], tc.args[3], tc.actions, tc.args[4]).Return(tc.actions, tc.sdkErr)
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

func TestListGroupRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
			sdkCall := sdkMock.On("GroupRoleActions", tc.args[1], tc.args[0], tc.args[2], tc.args[3]).Return(tc.actions, tc.sdkErr)
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

func TestDeleteGroupRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
				group.ID,
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
				sdkCall = sdkMock.On("RemoveAllGroupRoleActions", tc.args[2], tc.args[1], tc.args[3], tc.args[4]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveGroupRoleActions", tc.args[2], tc.args[1], tc.args[3], actions.Actions, tc.args[4]).Return(tc.sdkErr)
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

func TestAvailableGroupRoleActionsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
			sdkCall := sdkMock.On("AvailableGroupRoleActions", tc.args[0], tc.args[1]).Return(tc.actions, tc.sdkErr)
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

func TestAddGroupRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
			sdkCall := sdkMock.On("AddGroupRoleMembers", tc.args[2], tc.args[1], tc.args[3], tc.members, tc.args[4]).Return(tc.members, tc.sdkErr)
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

func TestListGroupRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
			sdkCall := sdkMock.On("GroupRoleMembers", tc.args[1], tc.args[0], tc.args[2], mock.Anything, tc.args[3]).Return(tc.members, tc.sdkErr)
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

func TestDeleteGroupRoleMembersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupsCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupsCmd)

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
				group.ID,
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
				group.ID,
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
				group.ID,
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
				sdkCall = sdkMock.On("RemoveAllGroupRoleMembers", tc.args[2], tc.args[1], tc.args[3], tc.args[4]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveGroupRoleMembers", tc.args[2], tc.args[1], tc.args[3], members.Members, tc.args[4]).Return(tc.sdkErr)
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
