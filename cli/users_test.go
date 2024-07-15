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
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var user = mgsdk.User{
	Name: "testuser",
	Credentials: mgsdk.Credentials{
		Secret:   "testpassword",
		Identity: "identity@example.com",
	},
	Status: mgclients.EnabledStatus.String(),
}

var (
	validToken   = "valid"
	invalidToken = ""
	invalidID    = "invalidID"
	extraArg     = "extra-arg"
)

func TestCreateUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	createCommand := "create"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "create user successfully with token",
			args: []string{
				createCommand,
				user.Name,
				user.Credentials.Identity,
				user.Credentials.Secret,
				validToken,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "create user successfully without token",
			args: []string{
				createCommand,
				user.Name,
				user.Credentials.Identity,
				user.Credentials.Secret,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "failed to create user",
			args: []string{
				createCommand,
				user.Name,
				user.Credentials.Identity,
				user.Credentials.Secret,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
		{
			desc:    "create user with invalid args",
			args:    []string{createCommand, user.Name, user.Credentials.Identity},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("CreateUser", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
		if len(tc.args) == 3 {
			sdkUser := mgsdk.User{
				Name: tc.args[0],
				Credentials: mgsdk.Credentials{
					Identity: tc.args[1],
					Secret:   tc.args[2],
				},
			}
			sdkCall = sdkMock.On("CreateUser", mock.Anything, sdkUser).Return(tc.user, tc.sdkerr)
		}
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case entityLog:
			err := json.Unmarshal([]byte(out), &usr)
			assert.Nil(t, err)
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}

		sdkCall.Unset()
	}
}

func TestGetUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	getCommand := "get"
	all := "all"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var page mgsdk.UsersPage
	var usr mgsdk.User
	out := ""
	userID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		page          mgsdk.UsersPage
		logType       outputLog
	}{
		{
			desc: "get users successfully",
			args: []string{
				getCommand,
				all,
				validToken,
			},
			sdkerr: nil,
			page: mgsdk.UsersPage{
				Users: []mgsdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "get user successfully with id",
			args: []string{
				getCommand,
				userID,
				validToken,
			},
			sdkerr:  nil,
			user:    user,
			logType: entityLog,
		},
		{
			desc: "get user with invalid id",
			args: []string{
				getCommand,
				invalidID,
				validToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest).Error()),
			user:          mgsdk.User{},
			logType:       errLog,
		},
		{
			desc: "get users successfully with offset and limit",
			args: []string{
				getCommand,
				all,
				validToken,
				"--offset=2",
				"--limit=5",
			},
			sdkerr: nil,
			page: mgsdk.UsersPage{
				Users: []mgsdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "get users with invalid token",
			args: []string{
				getCommand,
				all,
				invalidToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			page:          mgsdk.UsersPage{},
			logType:       errLog,
		},
		{
			desc: "get users with invalid args",
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
			desc: "get user with failed get operation",
			args: []string{
				getCommand,
				userID,
				validToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusInternalServerError),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusInternalServerError).Error()),
			user:          mgsdk.User{},
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("Users", mock.Anything, mock.Anything).Return(tc.page, tc.sdkerr)
		sdkCall1 := sdkMock.On("User", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)

		out = executeCommand(t, rootCmd, tc.args...)

		if tc.logType == entityLog {
			switch {
			case tc.args[1] == all:
				err := json.Unmarshal([]byte(out), &page)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
			default:
				err := json.Unmarshal([]byte(out), &usr)
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
				assert.Equal(t, tc.user, usr, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.user, usr))
			} else {
				assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
			}
		}

		sdkCall.Unset()
		sdkCall1.Unset()
	}
}

func TestIssueTokenCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	tokenCommand := "token"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var tkn mgsdk.Token
	domainID := testsutil.GenerateUUID(t)
	invalidPassword := ""

	token := mgsdk.Token{
		AccessToken:  testsutil.GenerateUUID(t),
		RefreshToken: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		token         mgsdk.Token
		logType       outputLog
	}{
		{
			desc: "issue token successfully without domain id",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
				user.Credentials.Secret,
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue token successfully with domain id",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
				user.Credentials.Secret,
				domainID,
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue token with failed authentication",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
				invalidPassword,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
			token:         mgsdk.Token{},
		},
		{
			desc: "issue token with invalid args",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("CreateToken", mock.Anything).Return(tc.token, tc.sdkerr)
		switch len(tc.args) {
		case 3:
			lg := mgsdk.Login{
				Identity: tc.args[1],
				Secret:   tc.args[2],
			}
			sdkCall = sdkMock.On("CreateToken", lg).Return(tc.token, tc.sdkerr)
		case 4:
			lg := mgsdk.Login{
				Identity: tc.args[1],
				Secret:   tc.args[2],
				DomainID: tc.args[3],
			}
			sdkCall = sdkMock.On("CreateToken", lg).Return(tc.token, tc.sdkerr)
		}
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case entityLog:
			err := json.Unmarshal([]byte(out), &tkn)
			assert.Nil(t, err)
			assert.Equal(t, tc.token, tkn, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.token, tkn))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}

		sdkCall.Unset()
	}
}

func TestRefreshIssueTokenCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	tokenCommand := "refreshtoken"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var tkn mgsdk.Token
	domainID := testsutil.GenerateUUID(t)
	invalidIdentity := "invalidIdentity"

	token := mgsdk.Token{
		AccessToken:  testsutil.GenerateUUID(t),
		RefreshToken: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		token         mgsdk.Token
		logType       outputLog
	}{
		{
			desc: "issue refresh token successfully without domain id",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue refresh token successfully with domain id",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
				domainID,
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue refresh token with invalid args",
			args: []string{
				tokenCommand,
				user.Credentials.Identity,
				domainID,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "issue refresh token with invalid identity",
			args: []string{
				tokenCommand,
				invalidIdentity,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
			token:         mgsdk.Token{},
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("RefreshToken", mock.Anything, mock.Anything).Return(tc.token, tc.sdkerr)
		switch len(tc.args) {
		case 2:
			lg := mgsdk.Login{
				Identity: tc.args[1],
			}
			sdkCall = sdkMock.On("RefreshToken", lg).Return(tc.token, tc.sdkerr)
		case 3:
			lg := mgsdk.Login{
				Identity: tc.args[1],
				DomainID: tc.args[2],
			}
			sdkCall = sdkMock.On("RefreshToken", lg).Return(tc.token, tc.sdkerr)
		}
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case entityLog:
			err := json.Unmarshal([]byte(out), &tkn)
			assert.Nil(t, err)
			assert.Equal(t, tc.token, tkn, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.token, tkn))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}

		sdkCall.Unset()
	}
}

func TestUpdateUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	updateCommand := "update"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	userID := testsutil.GenerateUUID(t)

	tagUpdateType := "tags"
	identityUpdateType := "identity"
	roleUpdateType := "role"
	newIdentity := "newidentity@example.com"
	newRole := "administrator"
	newTagsJSON := "[\"tag1\", \"tag2\"]"
	newNameMetadataJSON := "{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}}"

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "update user tags successfully",
			args: []string{
				updateCommand,
				tagUpdateType,
				userID,
				newTagsJSON,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user identity successfully",
			args: []string{
				updateCommand,
				identityUpdateType,
				userID,
				newIdentity,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user successfully",
			args: []string{
				updateCommand,
				userID,
				newNameMetadataJSON,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user role successfully",
			args: []string{
				updateCommand,
				roleUpdateType,
				userID,
				newRole,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user with invalid args",
			args: []string{
				updateCommand,
				roleUpdateType,
				userID,
				newRole,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("UpdateUser", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
		sdkCall1 := sdkMock.On("UpdateUserTags", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
		sdkCall2 := sdkMock.On("UpdateUserIdentity", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
		sdkCall3 := sdkMock.On("UpdateUserRole", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
		switch {
		case tc.args[1] == tagUpdateType:
			var u mgsdk.User
			u.Tags = []string{"tag1", "tag2"}
			u.ID = tc.args[2]

			sdkCall1 = sdkMock.On("UpdateUserTags", u, tc.args[4]).Return(tc.user, tc.sdkerr)
		case tc.args[1] == identityUpdateType:
			var u mgsdk.User
			u.Credentials.Identity = tc.args[3]
			u.ID = tc.args[2]

			sdkCall2 = sdkMock.On("UpdateUserIdentity", u, tc.args[4]).Return(tc.user, tc.sdkerr)
		case tc.args[1] == roleUpdateType && len(tc.args) == 5:
			sdkCall3 = sdkMock.On("UpdateUserRole", mgsdk.User{
				Role: tc.args[3],
			}, tc.args[4]).Return(tc.user, tc.sdkerr)
		case tc.args[1] == userID:
			sdkCall = sdkMock.On("UpdateUser", mgsdk.User{
				Name: "new name",
				Metadata: mgsdk.Metadata{
					"key": "value",
				},
			}, tc.args[3]).Return(tc.user, tc.sdkerr)
		}
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case entityLog:
			err := json.Unmarshal([]byte(out), &usr)
			assert.Nil(t, err)
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}

		sdkCall.Unset()
		sdkCall1.Unset()
		sdkCall2.Unset()
		sdkCall3.Unset()
	}
}

func TestGetUserProfileCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	profileCommand := "profile"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "get user profile successfully",
			args: []string{
				profileCommand,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
		},
		{
			desc: "get user profile with invalid args",
			args: []string{
				profileCommand,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("UserProfile", tc.args[1]).Return(tc.user, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &usr)
			assert.Nil(t, err)
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
		}
		sdkCall.Unset()
	}
}

func TestResetPasswordRequestCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	resetPasswordCommand := "resetpasswordrequest"
	exampleEmail := "example@mail.com"

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "request password reset successfully",
			args: []string{
				resetPasswordCommand,
				exampleEmail,
			},
			sdkerr:  nil,
			logType: okLog,
		},
		{
			desc: "request password reset with invalid args",
			args: []string{
				resetPasswordCommand,
				exampleEmail,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("ResetPasswordRequest", tc.args[1]).Return(tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}
		sdkCall.Unset()
	}
}

func TestResetPasswordCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	resetPasswordCommand := "resetpassword"
	newPassword := "new-password"

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "reset password successfully",
			args: []string{
				resetPasswordCommand,
				newPassword,
				newPassword,
				validToken,
			},
			sdkerr:  nil,
			logType: okLog,
		},
		{
			desc: "reset password with invalid args",
			args: []string{
				resetPasswordCommand,
				newPassword,
				newPassword,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("ResetPassword", tc.args[1], tc.args[2], tc.args[3]).Return(tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}

		sdkCall.Unset()
	}
}

func TestUpdatePasswordCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	passwordCommand := "password"
	oldPassword := "old-password"
	newPassword := "new-password"

	var usr mgsdk.User
	var err error

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "update password successfully",
			args: []string{
				passwordCommand,
				oldPassword,
				newPassword,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "reset password with invalid args",
			args: []string{
				passwordCommand,
				oldPassword,
				newPassword,
				validToken,
				extraArg,
			},
			sdkerr:  nil,
			logType: usageLog,
			user:    user,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("UpdatePassword", tc.args[1], tc.args[2], tc.args[3]).Return(tc.user, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err = json.Unmarshal([]byte(out), &usr)
			assert.Nil(t, err)
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s user mismatch: expected %+v got %+v", tc.desc, tc.user, usr))
		}

		sdkCall.Unset()
	}
}

func TestEnableUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	enableCommand := "enable"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	var usr mgsdk.User

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "enable user successfully",
			args: []string{
				enableCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			user:    user,
			logType: entityLog,
		},
		{
			desc: "enable user with invalid args",
			args: []string{
				enableCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("EnableUser", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &usr)
			assert.Nil(t, err)
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
		}

		sdkCall.Unset()
	}
}

func TestDisableUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	disableCommand := "disable"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "disable user successfully",
			args: []string{
				disableCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "disable user with invalid args",
			args: []string{
				disableCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("DisableUser", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)
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
			assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
		}

		sdkCall.Unset()
	}
}

func TestDeleteUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	deleteCommand := "delete"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete user successfully",
			args: []string{
				deleteCommand,
				user.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "delete user with invalid token",
			args: []string{
				deleteCommand,
				user.ID,
				invalidToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with invalid user ID",
			args: []string{
				deleteCommand,
				invalidID,
				validToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with failed to delete",
			args: []string{
				deleteCommand,
				user.ID,
				validToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with invalid args",
			args: []string{
				deleteCommand,
				user.ID,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("DeleteUser", mock.Anything, mock.Anything).Return(tc.sdkerr)
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

func TestListUserChannelsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	channelsCommand := "channels"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	ch := mgsdk.Channel{
		ID:   testsutil.GenerateUUID(t),
		Name: "testchannel",
	}

	var pg mgsdk.ChannelsPage

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		channel       mgsdk.Channel
		page          mgsdk.ChannelsPage
		output        bool
		logType       outputLog
	}{
		{
			desc: "list user channels successfully",
			args: []string{
				channelsCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.ChannelsPage{
				Channels: []mgsdk.Channel{ch},
			},
		},
		{
			desc: "list user channels successfully with flags",
			args: []string{
				channelsCommand,
				user.ID,
				validToken,
				"--offset=0",
				"--limit=5",
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.ChannelsPage{
				Channels: []mgsdk.Channel{ch},
			},
		},
		{
			desc: "list user channels with invalid args",
			args: []string{
				channelsCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("ListUserChannels", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &pg)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
		}

		sdkCall.Unset()
	}
}

func TestListUserThingsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	thingsCommand := "things"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	th := mgsdk.Thing{
		ID:   testsutil.GenerateUUID(t),
		Name: "testthing",
	}

	var pg mgsdk.ThingsPage

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		thing         mgsdk.Thing
		page          mgsdk.ThingsPage
		logType       outputLog
	}{
		{
			desc: "list user things successfully",
			args: []string{
				thingsCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.ThingsPage{
				Things: []mgsdk.Thing{th},
			},
		},
		{
			desc: "list user things with invalid args",
			args: []string{
				thingsCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		sdkCall := sdkMock.On("ListUserThings", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &pg)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
		}

		sdkCall.Unset()
	}
}

func TestListUserDomainsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCommand := "domains"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	d := mgsdk.Domain{
		ID:   testsutil.GenerateUUID(t),
		Name: "testdomain",
	}

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		logType       outputLog
		page          mgsdk.DomainsPage
	}{
		{
			desc: "list user domains successfully",
			args: []string{
				domainsCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.DomainsPage{
				Domains: []mgsdk.Domain{d},
			},
		},
		{
			desc: "list user domains with invalid args",
			args: []string{
				domainsCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		var pg mgsdk.DomainsPage
		sdkCall := sdkMock.On("ListUserDomains", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &pg)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
		}

		sdkCall.Unset()
	}
}

func TestListUserGroupsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	domainsCommand := "groups"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	g := mgsdk.Group{
		ID:   testsutil.GenerateUUID(t),
		Name: "testgroup",
	}

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		logType       outputLog
		page          mgsdk.GroupsPage
	}{
		{
			desc: "list user groups successfully",
			args: []string{
				domainsCommand,
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.GroupsPage{
				Groups: []mgsdk.Group{g},
			},
		},
		{
			desc: "list user groups with invalid args",
			args: []string{
				domainsCommand,
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		var pg mgsdk.GroupsPage
		sdkCall := sdkMock.On("ListUserGroups", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
		out := executeCommand(t, rootCmd, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		case entityLog:
			err := json.Unmarshal([]byte(out), &pg)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
		}

		sdkCall.Unset()
	}
}
