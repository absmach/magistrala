// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/absmach/magistrala/cli"
	// "github.com/absmach/magistrala/internal/testsutil".
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

var token = "validToken"

func TestCreateUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	createCommand := "create"
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

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
				token,
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
		var usr mgsdk.User
		out := executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)

		switch tc.logType {
		case errLog:
			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
		case usageLog:
			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
		}
		assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))

		sdkCall.Unset()
	}
}

// func TestGetUsersCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	getCommand := "get"
// 	all := "all"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		page          mgsdk.UsersPage
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "get users successfully",
// 			args: []string{
// 				getCommand,
// 				all,
// 				,
// 			},
// 			sdkerr: nil,
// 			page: mgsdk.UsersPage{
// 				Users: []mgsdk.User{user},
// 			},
// 			logType: entityLog,
// 		},
// 		{
// 			desc: "get user successfully with id",
// 			args: []string{
// 				getCommand,
// 				"id",
// 				,
// 			},
// 			sdkerr:  nil,
// 			user:    user,
// 			logType: entityLog,
// 		},
// 		{
// 			desc: "get users successfully with offset and limit",
// 			args: []string{
// 				getCommand,
// 				all,
// 				,
// 				"--offset=2",
// 				"--limit=5",
// 			},
// 			sdkerr: nil,
// 			page: mgsdk.UsersPage{
// 				Users: []mgsdk.User{user},
// 			},
// 			logType: entityLog,
// 		},
// 		{
// 			desc: "get users with invalid token",
// 			args: []string{
// 				getCommand,
// 				all,
// 				invalidToken,
// 			},
// 			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
// 			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
// 			page:          mgsdk.UsersPage{},
// 			logType:       errLog,
// 		},
// 		{
// 			desc: "invalid args for get users command",
// 			args: []string{
// 				getCommand,
// 				all,
// 				invalidToken,
// 				all,
// 				invalidToken,
// 				all,
// 				invalidToken,
// 				all,
// 				invalidToken,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var page mgsdk.UsersPage
// 		var usr mgsdk.User
// 		sdkCall := sdkMock.On("Users", mock.Anything, mock.Anything).Return(tc.page, tc.sdkerr)
// 		sdkCall1 := sdkMock.On("User", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)

// 		out := ""
// 		switch {
// 		case tc.args[1] == all:
// 			sdkCall = sdkMock.On("Users", mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
// 			out = executeCommand(t, rootCmd, &page, tc.logType, tc.args...)
// 		default:
// 			out = executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)
// 		}

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		if tc.args[1] != all {
// 			assert.Equal(t, tc.user, usr, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.user, usr))
// 		} else {
// 			assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
// 		}

// 		sdkCall.Unset()
// 		sdkCall1.Unset()
// 	}
// }

// func TestIssueTokenCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	tokenCommand := "token"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	token := mgsdk.Token{
// 		AccessToken:  ,
// 		RefreshToken: ,
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		token         mgsdk.Token
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "issue token successfully without domain id",
// 			args: []string{
// 				tokenCommand,
// 				"john.doe@example.com",
// 				"12345678",
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			token:   token,
// 		},
// 		{
// 			desc: "issue token successfully with domain id",
// 			args: []string{
// 				tokenCommand,
// 				"john.doe@example.com",
// 				"12345678",
// 				testsutil.GenerateUUID(t),
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			token:   token,
// 		},
// 		{
// 			desc: " failed to issue token successfully with authentication error",
// 			args: []string{
// 				tokenCommand,
// 				"john.doe@example.com",
// 				"wrong-password",
// 			},
// 			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
// 			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
// 			logType:       errLog,
// 			token:         mgsdk.Token{},
// 		},
// 		{
// 			desc: "invalid args for issue token command",
// 			args: []string{
// 				tokenCommand,
// 				"john.doe@example.com",
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var tkn mgsdk.Token
// 		sdkCall := sdkMock.On("CreateToken", mock.Anything).Return(tc.token, tc.sdkerr)
// 		switch len(tc.args) {
// 		case 3:
// 			lg := mgsdk.Login{
// 				Identity: tc.args[1],
// 				Secret:   tc.args[2],
// 			}
// 			sdkCall = sdkMock.On("CreateToken", lg).Return(tc.token, tc.sdkerr)
// 		case 4:
// 			lg := mgsdk.Login{
// 				Identity: tc.args[1],
// 				Secret:   tc.args[2],
// 				DomainID: tc.args[3],
// 			}
// 			sdkCall = sdkMock.On("CreateToken", lg).Return(tc.token, tc.sdkerr)
// 		}
// 		out := executeCommand(t, rootCmd, &tkn, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}
// 		assert.Equal(t, tc.token, tkn, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.token, tkn))

// 		sdkCall.Unset()
// 	}
// }

// func TestRefreshIssueTokenCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	tokenCommand := "refreshtoken"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	token := mgsdk.Token{
// 		AccessToken:  ,
// 		RefreshToken: ,
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		token         mgsdk.Token
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "issue refresh token successfully without domain id",
// 			args: []string{
// 				tokenCommand,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			token:   token,
// 		},
// 		{
// 			desc: "issue refresh token successfully with domain id",
// 			args: []string{
// 				tokenCommand,
// 				,
// 				testsutil.GenerateUUID(t),
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			token:   token,
// 		},
// 		{
// 			desc: "invalid args for issue refresh token",
// 			args: []string{
// 				tokenCommand,
// 				,
// 				testsutil.GenerateUUID(t),
// 				"extra-arg",
// 			},
// 			logType: usageLog,
// 		},
// 		{
// 			desc: "failed to issue token successfully",
// 			args: []string{
// 				tokenCommand,
// 				invalidToken,
// 			},
// 			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
// 			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
// 			logType:       errLog,
// 			token:         mgsdk.Token{},
// 		},
// 	}

// 	for _, tc := range cases {
// 		var tkn mgsdk.Token
// 		sdkCall := sdkMock.On("RefreshToken", mock.Anything, mock.Anything).Return(tc.token, tc.sdkerr)
// 		switch len(tc.args) {
// 		case 2:
// 			lg := mgsdk.Login{}
// 			sdkCall = sdkMock.On("RefreshToken", lg, tc.args[1]).Return(tc.token, tc.sdkerr)
// 		case 3:
// 			lg := mgsdk.Login{
// 				DomainID: tc.args[2],
// 			}
// 			sdkCall = sdkMock.On("RefreshToken", lg, tc.args[0]).Return(tc.token, tc.sdkerr)
// 		}
// 		out := executeCommand(t, rootCmd, &tkn, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}
// 		assert.Equal(t, tc.token, tkn, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.token, tkn))
// 		sdkCall.Unset()
// 	}
// }

// func TestUpdateUserCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	updateCommand := "update"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "update user tags successfully",
// 			args: []string{
// 				updateCommand,
// 				"tags",
// 				user.ID,
// 				"[\"tag1\", \"tag2\"]",
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "update user identity successfully",
// 			args: []string{
// 				updateCommand,
// 				"identity",
// 				user.ID,
// 				"newidentity@example.com",
// 				,
// 			},
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "update user successfully",
// 			args: []string{
// 				updateCommand,
// 				user.ID,
// 				"{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}}",
// 				,
// 			},
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "update user role successfully",
// 			args: []string{
// 				updateCommand,
// 				"role",
// 				user.ID,
// 				"administrator",
// 				,
// 			},
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "update user with invalid args",
// 			args: []string{
// 				updateCommand,
// 				"role",
// 				user.ID,
// 				"administrator",
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var usr mgsdk.User
// 		sdkCall := sdkMock.On("UpdateUser", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
// 		sdkCall1 := sdkMock.On("UpdateUserTags", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
// 		sdkCall2 := sdkMock.On("UpdateUserIdentity", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
// 		sdkCall3 := sdkMock.On("UpdateUserRole", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
// 		switch {
// 		case tc.args[1] == "tags":
// 			var u mgsdk.User
// 			u.Tags = []string{"tag1", "tag2"}
// 			u.ID = tc.args[2]

// 			sdkCall1 = sdkMock.On("UpdateUserTags", u, tc.args[4]).Return(tc.user, tc.sdkerr)
// 		case tc.args[1] == "identity":
// 			var u mgsdk.User
// 			u.Credentials.Identity = tc.args[3]
// 			u.ID = tc.args[2]

// 			sdkCall2 = sdkMock.On("UpdateUserIdentity", u, tc.args[4]).Return(tc.user, tc.sdkerr)
// 		case tc.args[1] == "role" && len(tc.args) == 5:
// 			sdkCall3 = sdkMock.On("UpdateUserRole", mgsdk.User{
// 				Role: tc.args[3],
// 			}, tc.args[4]).Return(tc.user, tc.sdkerr)
// 		case tc.args[1] == user.ID:
// 			sdkCall = sdkMock.On("UpdateUser", mgsdk.User{
// 				Name: "new name",
// 				Metadata: mgsdk.Metadata{
// 					"key": "value",
// 				},
// 			}, tc.args[3]).Return(tc.user, tc.sdkerr)
// 		}
// 		out := executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))

// 		sdkCall.Unset()
// 		sdkCall1.Unset()
// 		sdkCall2.Unset()
// 		sdkCall3.Unset()
// 	}
// }

// func TestGetUserProfileCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	profileCommand := "profile"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "get user profile successfully",
// 			args: []string{
// 				profileCommand,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 		},
// 		{
// 			desc: "get user profile with invalid args",
// 			args: []string{
// 				profileCommand,
// 				,
// 				"extra-arg",
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var usr mgsdk.User
// 		sdkCall := sdkMock.On("UserProfile", tc.args[1]).Return(tc.user, tc.sdkerr)

// 		out := executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}
// 		assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
// 		sdkCall.Unset()
// 	}
// }

// func TestResetPasswordRequestCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "request password reset successfully",
// 			args: []string{
// 				"resetpasswordrequest",
// 				"example@mail.com",
// 			},
// 			sdkerr:  nil,
// 			logType: okLog,
// 		},
// 		{
// 			desc: "request password reset with invalid args",
// 			args: []string{
// 				"resetpasswordrequest",
// 				"example@mail.com",
// 				"extra-arg",
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		sdkCall := sdkMock.On("ResetPasswordRequest", tc.args[1]).Return(tc.sdkerr)
// 		out := executeCommand(t, rootCmd, nil, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}
// 		sdkCall.Unset()
// 	}
// }

// func TestResetPasswordCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)
// 	validRequestToken :=

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "reset password successfully",
// 			args: []string{
// 				"resetpassword",
// 				"new-password",
// 				"new-password",
// 				validRequestToken,
// 			},
// 			sdkerr:  nil,
// 			logType: okLog,
// 		},
// 		{
// 			desc: "reset password with invalid args",
// 			args: []string{
// 				"resetpassword",
// 				"new-password",
// 				"new-password",
// 				validRequestToken,
// 				"extra-arg",
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		sdkCall := sdkMock.On("ResetPassword", tc.args[1], tc.args[2], tc.args[3]).Return(tc.sdkerr)
// 		out := executeCommand(t, rootCmd, nil, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		sdkCall.Unset()
// 	}
// }

// func TestUpdatePasswordCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "reset password successfully",
// 			args: []string{
// 				"password",
// 				"old-password",
// 				"new-password",
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "reset password with invalid args",
// 			args: []string{
// 				"password",
// 				"old-password",
// 				"new-password",
// 				,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: usageLog,
// 			user:    user,
// 		},
// 	}

// 	for _, tc := range cases {
// 		sdkCall := sdkMock.On("UpdatePassword", tc.args[1], tc.args[2], tc.args[3]).Return(tc.user, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &mgsdk.User{}, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		sdkCall.Unset()
// 	}
// }

// func TestEnableUserCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	enableCommand := "enable"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "enable user successfully",
// 			args: []string{
// 				enableCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			user:    user,
// 			logType: entityLog,
// 		},
// 		{
// 			desc: "enable user with invalid args",
// 			args: []string{
// 				enableCommand,
// 				user.ID,
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var usr mgsdk.User
// 		sdkCall := sdkMock.On("EnableUser", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
// 		sdkCall.Unset()
// 	}
// }

// func TestDisableUserCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	disableCommand := "disable"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		user          mgsdk.User
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "disable user successfully",
// 			args: []string{
// 				disableCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			user:    user,
// 		},
// 		{
// 			desc: "disable user with invalid args",
// 			args: []string{
// 				disableCommand,
// 				user.ID,
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var usr mgsdk.User
// 		sdkCall := sdkMock.On("DisableUser", tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &usr, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
// 		sdkCall.Unset()
// 	}
// }

// func TestListUserChannelsCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	channelsCommand := "channels"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)
// 	ch := mgsdk.Channel{
// 		ID:   testsutil.GenerateUUID(t),
// 		Name: "testchannel",
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		channel       mgsdk.Channel
// 		page          mgsdk.ChannelsPage
// 		output        bool
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "list user channels successfully",
// 			args: []string{
// 				channelsCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			page: mgsdk.ChannelsPage{
// 				Channels: []mgsdk.Channel{ch},
// 			},
// 		},
// 		{
// 			desc: "list user channels successfully with flags",
// 			args: []string{
// 				channelsCommand,
// 				user.ID,
// 				,
// 				"--offset=0",
// 				"--limit=5",
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			page: mgsdk.ChannelsPage{
// 				Channels: []mgsdk.Channel{ch},
// 			},
// 		},
// 		{
// 			desc: "list user channels with invalid args",
// 			args: []string{
// 				channelsCommand,
// 				user.ID,
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var pg mgsdk.ChannelsPage
// 		sdkCall := sdkMock.On("ListUserChannels", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &pg, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
// 		sdkCall.Unset()
// 	}
// }

// func TestListUserThingsCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	thingsCommand := "things"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)
// 	th := mgsdk.Thing{
// 		ID:   testsutil.GenerateUUID(t),
// 		Name: "testthing",
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		thing         mgsdk.Thing
// 		page          mgsdk.ThingsPage
// 		logType       outputLog
// 	}{
// 		{
// 			desc: "list user things successfully",
// 			args: []string{
// 				thingsCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			page: mgsdk.ThingsPage{
// 				Things: []mgsdk.Thing{th},
// 			},
// 		},
// 		{
// 			desc: "list user things with invalid args",
// 			args: []string{
// 				thingsCommand,
// 				user.ID,
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var pg mgsdk.ThingsPage
// 		sdkCall := sdkMock.On("ListUserThings", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &pg, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
// 		sdkCall.Unset()
// 	}
// }

// func TestListUserDomainsCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	domainsCommand := "domains"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)
// 	d := mgsdk.Domain{
// 		ID:   testsutil.GenerateUUID(t),
// 		Name: "testdomain",
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		logType       outputLog
// 		page          mgsdk.DomainsPage
// 	}{
// 		{
// 			desc: "list user domains successfully",
// 			args: []string{
// 				domainsCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			page: mgsdk.DomainsPage{
// 				Domains: []mgsdk.Domain{d},
// 			},
// 		},
// 		{
// 			desc: "list user domains with invalid args",
// 			args: []string{
// 				domainsCommand,
// 				user.ID,
// 				,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var pg mgsdk.DomainsPage
// 		sdkCall := sdkMock.On("ListUserDomains", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &pg, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
// 		sdkCall.Unset()
// 	}
// }

// func TestListUserGroupsCmd(t *testing.T) {
// 	sdkMock := new(sdkmocks.SDK)
// 	cli.SetSDK(sdkMock)
// 	domainsCommand := "groups"
// 	usersCmd := cli.NewUsersCmd()
// 	rootCmd := setFlags(usersCmd)
// 	g := mgsdk.Group{
// 		ID:   testsutil.GenerateUUID(t),
// 		Name: "testgroup",
// 	}

// 	cases := []struct {
// 		desc          string
// 		args          []string
// 		sdkerr        errors.SDKError
// 		errLogMessage string
// 		logType       outputLog
// 		page          mgsdk.GroupsPage
// 	}{
// 		{
// 			desc: "list user groups successfully",
// 			args: []string{
// 				domainsCommand,
// 				user.ID,
// 				,
// 			},
// 			sdkerr:  nil,
// 			logType: entityLog,
// 			page: mgsdk.GroupsPage{
// 				Groups: []mgsdk.Group{g},
// 			},
// 		},
// 		{
// 			desc: "list user groups with invalid args",
// 			args: []string{
// 				domainsCommand,
// 				user.ID,
// 				,
// 			},
// 			logType: usageLog,
// 		},
// 	}

// 	for _, tc := range cases {
// 		var pg mgsdk.GroupsPage
// 		sdkCall := sdkMock.On("ListUserGroups", tc.args[1], mock.Anything, tc.args[2]).Return(tc.page, tc.sdkerr)
// 		out := executeCommand(t, rootCmd, &pg, tc.logType, tc.args...)

// 		switch tc.logType {
// 		case errLog:
// 			assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
// 		case usageLog:
// 			assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
// 		}

// 		assert.Equal(t, tc.page, pg, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.page, pg))
// 		sdkCall.Unset()
// 	}
// }
