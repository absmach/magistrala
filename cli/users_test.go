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
	"github.com/absmach/magistrala/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var user = mgsdk.User{
	ID:        testsutil.GenerateUUID(&testing.T{}),
	FirstName: "testuserfirstname",
	LastName:  "testuserfirstname",
	Credentials: mgsdk.Credentials{
		Secret:   "testpassword",
		Username: "testusername",
	},
	Status: users.EnabledStatus.String(),
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
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Secret,
				user.Credentials.Username,
				validToken,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "create user successfully without token",
			args: []string{
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Secret,
				user.Credentials.Username,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "failed to create user",
			args: []string{
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Secret,
				user.Credentials.Username,
				validToken,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
		{
			desc:    "create user with invalid args",
			args:    []string{user.FirstName, user.Credentials.Username},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateUser", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
			if len(tc.args) == 4 {
				sdkUser := mgsdk.User{
					FirstName: tc.args[0],
					LastName:  tc.args[1],
					Email:     tc.args[2],
					Credentials: mgsdk.Credentials{
						Secret: tc.args[3],
					},
				}
				sdkCall = sdkMock.On("CreateUser", mock.Anything, sdkUser).Return(tc.user, tc.sdkerr)
			}
			out := executeCommand(t, rootCmd, append([]string{createCmd}, tc.args...)...)

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
		})
	}
}

func TestGetUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Users", mock.Anything, mock.Anything).Return(tc.page, tc.sdkerr)
			sdkCall1 := sdkMock.On("User", tc.args[0], tc.args[1]).Return(tc.user, tc.sdkerr)

			out = executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)

			if tc.logType == entityLog {
				switch {
				case tc.args[0] == all:
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
				if tc.args[0] != all {
					assert.Equal(t, tc.user, usr, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.user, usr))
				} else {
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				}
			}

			sdkCall.Unset()
			sdkCall1.Unset()
		})
	}
}

func TestIssueTokenCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var tkn mgsdk.Token
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
			desc: "issue token successfully",
			args: []string{
				user.Email,
				user.Credentials.Secret,
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue token with failed authentication",
			args: []string{
				user.Email,
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
				user.Email,
				user.Credentials.Secret,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			lg := mgsdk.Login{
				Identity: tc.args[0],
				Secret:   tc.args[1],
			}
			sdkCall := sdkMock.On("CreateToken", lg).Return(tc.token, tc.sdkerr)

			out := executeCommand(t, rootCmd, append([]string{tokCmd}, tc.args...)...)

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
		})
	}
}

func TestRefreshIssueTokenCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var tkn mgsdk.Token

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
				"token",
			},
			sdkerr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue refresh token with invalid args",
			args: []string{
				"token",
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "issue refresh token with invalid Username",
			args: []string{
				"invalidToken",
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
			token:         mgsdk.Token{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RefreshToken", mock.Anything).Return(tc.token, tc.sdkerr)

			out := executeCommand(t, rootCmd, append([]string{refTokCmd}, tc.args...)...)

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
		})
	}
}

func TestUpdateUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	userID := testsutil.GenerateUUID(t)

	tagUpdateType := "tags"
	emailUpdateType := "email"
	roleUpdateType := "role"
	newEmail := "newemail@example.com"
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
			desc: "update user tags with invalid json",
			args: []string{
				tagUpdateType,
				userID,
				"[\"tag1\", \"tag2\"",
				validToken,
			},
			sdkerr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update user tags with invalid token",
			args: []string{
				tagUpdateType,
				userID,
				newTagsJSON,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user email successfully",
			args: []string{
				emailUpdateType,
				userID,
				newEmail,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user email with invalid token",
			args: []string{
				emailUpdateType,
				userID,
				newEmail,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user successfully",
			args: []string{
				userID,
				newNameMetadataJSON,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user with invalid token",
			args: []string{
				userID,
				newNameMetadataJSON,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user with invalid json",
			args: []string{
				userID,
				"{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}",
				validToken,
			},
			sdkerr:        errors.NewSDKError(errors.New("unexpected end of JSON input")),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.New("unexpected end of JSON input")),
			logType:       errLog,
		},
		{
			desc: "update user role successfully",
			args: []string{
				roleUpdateType,
				userID,
				newRole,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user role with invalid token",
			args: []string{
				roleUpdateType,
				userID,
				newRole,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user with invalid args",
			args: []string{
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
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdateUser", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
			sdkCall1 := sdkMock.On("UpdateUserTags", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
			sdkCall2 := sdkMock.On("UpdateUserIdentity", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
			sdkCall3 := sdkMock.On("UpdateUserRole", mock.Anything, mock.Anything).Return(tc.user, tc.sdkerr)
			switch {
			case tc.args[0] == tagUpdateType:
				var u mgsdk.User
				u.Tags = []string{"tag1", "tag2"}
				u.ID = tc.args[1]

				sdkCall1 = sdkMock.On("UpdateUserTags", u, tc.args[3]).Return(tc.user, tc.sdkerr)
			case tc.args[0] == emailUpdateType:
				var u mgsdk.User
				u.Email = tc.args[2]
				u.ID = tc.args[1]

				sdkCall2 = sdkMock.On("UpdateUserEmail", u, tc.args[3]).Return(tc.user, tc.sdkerr)
			case tc.args[0] == roleUpdateType && len(tc.args) == 4:
				sdkCall3 = sdkMock.On("UpdateUserRole", mgsdk.User{
					Role: tc.args[2],
				}, tc.args[3]).Return(tc.user, tc.sdkerr)
			case tc.args[0] == userID:
				sdkCall = sdkMock.On("UpdateUser", mgsdk.User{
					FirstName: "new name",
					Metadata: mgsdk.Metadata{
						"key": "value",
					},
				}, tc.args[2]).Return(tc.user, tc.sdkerr)
			}
			out := executeCommand(t, rootCmd, append([]string{updCmd}, tc.args...)...)

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
		})
	}
}

func TestGetUserProfileCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
		},
		{
			desc: "get user profile with invalid args",
			args: []string{
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "get user profile with invalid token",
			args: []string{
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UserProfile", tc.args[0]).Return(tc.user, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{profCmd}, tc.args...)...)

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
		})
	}
}

func TestResetPasswordRequestCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
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
				exampleEmail,
			},
			sdkerr:  nil,
			logType: okLog,
		},
		{
			desc: "request password reset with invalid args",
			args: []string{
				exampleEmail,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "failed request password reset",
			args: []string{
				exampleEmail,
			},
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ResetPasswordRequest", tc.args[0]).Return(tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{resPassReqCmd}, tc.args...)...)

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

func TestResetPasswordCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
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
				newPassword,
				newPassword,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "reset password with invalid token",
			args: []string{
				newPassword,
				newPassword,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ResetPassword", tc.args[0], tc.args[1], tc.args[2]).Return(tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{resPassCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}

func TestUpdatePasswordCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
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
				oldPassword,
				newPassword,
				validToken,
				extraArg,
			},
			sdkerr:  nil,
			logType: usageLog,
			user:    user,
		},
		{
			desc: "update password with invalid token",
			args: []string{
				oldPassword,
				newPassword,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdatePassword", tc.args[0], tc.args[1], tc.args[2]).Return(tc.user, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{passCmd}, tc.args...)...)

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
		})
	}
}

func TestEnableUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "enable user with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableUser", tc.args[0], tc.args[1]).Return(tc.user, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{enableCmd}, tc.args...)...)

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
		})
	}
}

func TestDisableUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "disable user with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableUser", tc.args[0], tc.args[1]).Return(tc.user, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{disableCmd}, tc.args...)...)

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
		})
	}
}

func TestDeleteUserCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "delete user with invalid args",
			args: []string{
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete user with invalid token",
			args: []string{
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
				user.ID,
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteUser", mock.Anything, mock.Anything).Return(tc.sdkerr)
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

func TestListUserChannelsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list user channels with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListUserChannels", tc.args[0], mock.Anything, tc.args[1]).Return(tc.page, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{chansCmd}, tc.args...)...)

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
		})
	}
}

func TestListUserClientsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	th := mgsdk.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "testclient",
	}

	var pg mgsdk.ClientsPage

	cases := []struct {
		desc          string
		args          []string
		sdkerr        errors.SDKError
		errLogMessage string
		client        mgsdk.Client
		page          mgsdk.ClientsPage
		logType       outputLog
	}{
		{
			desc: "list user clients successfully",
			args: []string{
				user.ID,
				validToken,
			},
			sdkerr:  nil,
			logType: entityLog,
			page: mgsdk.ClientsPage{
				Clients: []mgsdk.Client{th},
			},
		},
		{
			desc: "list user clients with invalid args",
			args: []string{
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list user clients with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListUserClients", tc.args[0], mock.Anything, tc.args[1]).Return(tc.page, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{cliCmd}, tc.args...)...)

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
		})
	}
}

func TestListUserDomainsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list user domains with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var pg mgsdk.DomainsPage
			sdkCall := sdkMock.On("ListUserDomains", tc.args[0], mock.Anything, tc.args[1]).Return(tc.page, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{domsCmd}, tc.args...)...)

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
		})
	}
}

func TestListUserGroupsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
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
				user.ID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "list user groups with invalid token",
			args: []string{
				user.ID,
				invalidToken,
			},
			logType:       errLog,
			sdkerr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var pg mgsdk.GroupsPage
			sdkCall := sdkMock.On("ListUserGroups", tc.args[0], mock.Anything, tc.args[1]).Return(tc.page, tc.sdkerr)
			out := executeCommand(t, rootCmd, append([]string{grpCmd}, tc.args...)...)

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
		})
	}
}
