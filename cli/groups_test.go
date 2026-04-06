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

const (
	tagUpdateType = "tags"
	newTagsJson   = "[\"tag1\", \"tag2\"]"
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
				createCmd,
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
				createCmd,
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
				createCmd,
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
				createCmd,
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
				createCmd,
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
			sdkCall := sdkMock.On("CreateGroup", mock.Anything, mock.Anything, tc.args[2], tc.args[3]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &gp)
				assert.Nil(t, err)
				assert.Equal(t, tc.group, gp, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.group, gp))
			case usageLog:
				assert.True(t, strings.Contains(out, "cli groups create"), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
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
				delCmd,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete group with invalid args",
			args: []string{
				group.ID,
				delCmd,
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
				delCmd,
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
				delCmd,
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
			sdkCall := sdkMock.On("DeleteGroup", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.sdkErr)
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

func TestUpdategroupCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	groupCmd := cli.NewGroupsCmd()
	rootCmd := setFlags(groupCmd)

	newTagString := []string{"tag1", "tag2"}

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
				group.ID,
				updateCmd,
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
				group.ID,
				updateCmd,
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
				invalidID,
				updateCmd,
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
				group.ID,
				updateCmd,
				fmt.Sprintf("{\"id\":\"%s\",\"name\" : \"group1\"", group.ID),
				domainID,
				token,
			},
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update group tags successfully",
			args: []string{
				group.ID,
				updateCmd,
				tagUpdateType,
				newTagsJson,
				domainID,
				token,
			},
			group: smqsdk.Group{
				Name:     group.Name,
				ID:       group.ID,
				DomainID: group.DomainID,
				Status:   group.Status,
				Tags:     newTagString,
			},
			logType: entityLog,
		},
		{
			desc: "update group with invalid tags",
			args: []string{
				group.ID,
				updateCmd,
				tagUpdateType,
				"[\"tag1\", \"tag2\"",
				domainID,
				token,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
		},
		{
			desc: "update group tags with invalid group id",
			args: []string{
				invalidID,
				updateCmd,
				tagUpdateType,
				newTagsJson,
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
			var ch smqsdk.Group
			sdkCall := sdkMock.On("UpdateGroup", mock.Anything, mock.Anything, tc.args[3], tc.args[4]).Return(tc.group, tc.sdkErr)
			sdkCall1 := sdkMock.On("UpdateGroupTags", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
			sdkCall1.Unset()
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
				enableCmd,
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
				enableCmd,
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
				enableCmd,
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
				enableCmd,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableGroup", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				disableCmd,
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
				disableCmd,
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
				disableCmd,
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
				disableCmd,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableGroup", mock.Anything, tc.args[0], tc.args[2], tc.args[3]).Return(tc.group, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				createCmd,
				`{"role_name":"admin","optional_actions":["read","update"]}`,
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
				group.ID,
				rolesCmd,
				createCmd,
				`{"role_name":"admin","optional_actions":["read","update"}`,
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
			sdkCall := sdkMock.On("CreateGroupRole", mock.Anything, tc.args[0], tc.args[4], roleReq, tc.args[5]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				getCmd,
				all,
				domainID,
				token,
			},
			roles:   rolesPage,
			logType: entityLog,
		},
		{
			desc: "get group roles with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				getCmd,
				all,
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
			sdkCall := sdkMock.On("GroupRoles", mock.Anything, tc.args[0], tc.args[4], mock.Anything, tc.args[5]).Return(tc.roles, tc.sdkErr)
			if tc.args[3] != all {
				sdkCall = sdkMock.On("GroupRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4], tc.args[5]).Return(role, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				updateCmd,
				role.ID,
				"new_name",
				domainID,
				token,
			},
			role:    role,
			logType: entityLog,
		},
		{
			desc: "update group role name with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				updateCmd,
				role.ID,
				"new_name",
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
			sdkCall := sdkMock.On("UpdateGroupRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4], tc.args[5], tc.args[6]).Return(tc.role, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				delCmd,
				roleID,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete group role with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				delCmd,
				roleID,
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
			sdkCall := sdkMock.On("DeleteGroupRole", mock.Anything, tc.args[0], tc.args[3], tc.args[4], tc.args[5]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				actionsCmd,
				addCmd,
				roleID,
				`{"actions":["read","write"]}`,
				domainID,
				token,
			},
			actions: actions.Actions,
			logType: entityLog,
		},
		{
			desc: "add actions to role with invalid JSON",
			args: []string{
				group.ID,
				rolesCmd,
				actionsCmd,
				addCmd,
				roleID,
				`{"actions":["read","write"}`,
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
			sdkCall := sdkMock.On("AddGroupRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[6], tc.actions, tc.args[7]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				actionsCmd,
				listCmd,
				roleID,
				domainID,
				token,
			},
			actions: actions,
			logType: entityLog,
		},
		{
			desc: "list actions of role with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				actionsCmd,
				listCmd,
				roleID,
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
			sdkCall := sdkMock.On("GroupRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[5], tc.args[6]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				`{"actions":["read","write"]}`,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete all actions from role successfully",
			args: []string{
				group.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				all,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete actions from role with invalid JSON",
			args: []string{
				group.ID,
				rolesCmd,
				actionsCmd,
				delCmd,
				roleID,
				`{"actions":["read","write"}`,
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
			if tc.args[5] == all {
				sdkCall = sdkMock.On("RemoveAllGroupRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[6], tc.args[7]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveGroupRoleActions", mock.Anything, tc.args[0], tc.args[4], tc.args[6], actions.Actions, tc.args[7]).Return(tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				actionsCmd,
				availableActionsCmd,
				domainID,
				token,
			},
			actions: actions,
			logType: entityLog,
		},
		{
			desc: "list available actions with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				actionsCmd,
				availableActionsCmd,
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
			sdkCall := sdkMock.On("AvailableGroupRoleActions", mock.Anything, tc.args[4], tc.args[5]).Return(tc.actions, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				membersCmd,
				addCmd,
				roleID,
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`,
				domainID,
				token,
			},
			members: members.Members,
			logType: entityLog,
		},
		{
			desc: "add members to role with invalid JSON",
			args: []string{
				group.ID,
				rolesCmd,
				membersCmd,
				addCmd,
				roleID,
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}`,
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
			sdkCall := sdkMock.On("AddGroupRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.args[6], tc.members, tc.args[7]).Return(tc.members, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				membersCmd,
				listCmd,
				roleID,
				domainID,
				token,
			},
			members: membersPage,
			logType: entityLog,
		},
		{
			desc: "list members of role with invalid token",
			args: []string{
				group.ID,
				rolesCmd,
				membersCmd,
				listCmd,
				roleID,
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
			sdkCall := sdkMock.On("GroupRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.args[5], mock.Anything, tc.args[6]).Return(tc.members, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
				group.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"]}`,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete all members from role successfully",
			args: []string{
				group.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				all,
				domainID,
				token,
			},
			logType: okLog,
		},
		{
			desc: "delete members from role with invalid JSON",
			args: []string{
				group.ID,
				rolesCmd,
				membersCmd,
				delCmd,
				roleID,
				`{"members":["5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb", "5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb"}`,
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
			if tc.args[5] == all {
				sdkCall = sdkMock.On("RemoveAllGroupRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.args[6], tc.args[7]).Return(tc.sdkErr)
			} else {
				sdkCall = sdkMock.On("RemoveGroupRoleMembers", mock.Anything, tc.args[0], tc.args[4], tc.args[6], members.Members, tc.args[7]).Return(tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, tc.args...)

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
