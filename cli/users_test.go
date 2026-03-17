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
	mgsdk "github.com/absmach/supermq/pkg/sdk"
	sdkmocks "github.com/absmach/supermq/pkg/sdk/mocks"
	"github.com/absmach/supermq/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var user = mgsdk.User{
	ID:        testsutil.GenerateUUID(&testing.T{}),
	FirstName: "testuserfirstname",
	LastName:  "testuserfirstname",
	Email:     "testuser@example.com",
	Credentials: mgsdk.Credentials{
		Secret:   "testpassword",
		Username: "testusername",
	},
	Status: users.EnabledStatus.String(),
}


func TestCreateUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var usr mgsdk.User

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "create user successfully with token",
			args: []string{
				createCmd,
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Username,
				user.Credentials.Secret,
				validToken,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "create user successfully without token",
			args: []string{
				createCmd,
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Username,
				user.Credentials.Secret,
			},
			user:    user,
			logType: entityLog,
		},
		{
			desc: "failed to create user",
			args: []string{
				createCmd,
				user.FirstName,
				user.LastName,
				user.Email,
				user.Credentials.Username,
				user.Credentials.Secret,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
		{
			desc: "create user with invalid args",
			args: []string{
				createCmd,
				user.FirstName,
				user.Credentials.Username,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.sdkErr)
			if len(tc.args) == 6 {
				sdkUser := mgsdk.User{
					FirstName: tc.args[1],
					LastName:  tc.args[2],
					Email:     tc.args[3],
					Credentials: mgsdk.Credentials{
						Username: tc.args[4],
						Secret:   tc.args[5],
					},
					Status: users.EnabledStatus.String(),
				}
				sdkCall = sdkMock.On("CreateUser", mock.Anything, sdkUser, "").Return(tc.user, tc.sdkErr)
			} else if len(tc.args) == 7 {
				sdkUser := mgsdk.User{
					FirstName: tc.args[1],
					LastName:  tc.args[2],
					Email:     tc.args[3],
					Credentials: mgsdk.Credentials{
						Username: tc.args[4],
						Secret:   tc.args[5],
					},
					Status: users.EnabledStatus.String(),
				}
				sdkCall = sdkMock.On("CreateUser", mock.Anything, sdkUser, tc.args[6]).Return(tc.user, tc.sdkErr)
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
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		page          mgsdk.UsersPage
		logType       outputLog
	}{
		{
			desc: "get users successfully",
			args: []string{
				all,
				getCmd,
				validToken,
			},
			sdkErr: nil,
			page: mgsdk.UsersPage{
				Users: []mgsdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "get user successfully with id",
			args: []string{
				userID,
				getCmd,
				validToken,
			},
			sdkErr:  nil,
			user:    user,
			logType: entityLog,
		},
		{
			desc: "get user with invalid id",
			args: []string{
				invalidID,
				getCmd,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest).Error()),
			user:          mgsdk.User{},
			logType:       errLog,
		},
		{
			desc: "get users successfully with offset and limit",
			args: []string{
				all,
				getCmd,
				validToken,
				"--offset=2",
				"--limit=5",
			},
			sdkErr: nil,
			page: mgsdk.UsersPage{
				Users: []mgsdk.User{user},
			},
			logType: entityLog,
		},
		{
			desc: "get users with invalid token",
			args: []string{
				all,
				getCmd,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			page:          mgsdk.UsersPage{},
			logType:       errLog,
		},
		{
			desc: "get users with invalid args",
			args: []string{
				all,
				getCmd,
				validToken,
				extraArg,
			},
			errLogMessage: "cli users <user_id|all> get <user_auth_token>",
			logType:       usageLog,
		},
		{
			desc: "get user with failed get operation",
			args: []string{
				userID,
				getCmd,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusInternalServerError),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusInternalServerError).Error()),
			user:          mgsdk.User{},
			logType:       errLog,
		},
		{
			desc: "get user without operation",
			args: []string{
				userID,
			},
			errLogMessage: "users <user_id|all> <get|update|enable|disable|delete> [args...]",
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("Users", mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
			var sdkCall1 *mock.Call
			if len(tc.args) >= 3 {
				sdkCall1 = sdkMock.On("User", mock.Anything, tc.args[0], tc.args[2]).Return(tc.user, tc.sdkErr)
			}

			out = executeCommand(t, rootCmd, tc.args...)

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
				assert.True(t, strings.Contains(out, tc.errLogMessage), fmt.Sprintf("%s invalid usage: expected to contain %s, got: %s", tc.desc, tc.errLogMessage, out))
			}

			if tc.logType == entityLog {
				if tc.args[0] != all {
					assert.Equal(t, tc.user, usr, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.user, usr))
				} else {
					assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
				}
			}

			sdkCall.Unset()
			if sdkCall1 != nil {
				sdkCall1.Unset()
			}
		})
	}
}

func TestIssueTokenCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	var tkn mgsdk.Token
	invalidPassword := "wrong_password"

	token := mgsdk.Token{
		AccessToken:  testsutil.GenerateUUID(t),
		RefreshToken: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		token         mgsdk.Token
		logType       outputLog
	}{
		{
			desc: "issue token successfully",
			args: []string{
				tokCmd,
				user.Email,
				user.Credentials.Secret,
			},
			sdkErr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue token with failed authentication",
			args: []string{
				tokCmd,
				user.Email,
				invalidPassword,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
			token:         mgsdk.Token{},
		},
		{
			desc: "issue token with invalid args",
			args: []string{
				tokCmd,
				user.Email,
				user.Credentials.Secret,
				extraArg,
			},
			errLogMessage: "cli users token <username> <password>",
			logType:       usageLog,
		},
		{
			desc: "issue token with missing password",
			args: []string{
				tokCmd,
				user.Email,
			},
			errLogMessage: "cli users token <username> <password>",
			logType:       usageLog,
		},
		{
			desc: "issue token with missing username",
			args: []string{
				tokCmd,
			},
			errLogMessage: "cli users token <username> <password>",
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if len(tc.args) >= 3 {
				lg := mgsdk.Login{
					Username: tc.args[1],
					Password: tc.args[2],
				}
				sdkCall = sdkMock.On("CreateToken", mock.Anything, lg).Return(tc.token, tc.sdkErr)
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
				assert.True(t, strings.Contains(out, tc.errLogMessage), fmt.Sprintf("%s invalid usage: expected to contain %s, got: %s", tc.desc, tc.errLogMessage, out))
			}

			if sdkCall != nil {
				sdkCall.Unset()
			}
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
		sdkErr        errors.SDKError
		errLogMessage string
		token         mgsdk.Token
		logType       outputLog
	}{
		{
			desc: "issue refresh token successfully without domain id",
			args: []string{
				refTokCmd,
				"token",
			},
			sdkErr:  nil,
			logType: entityLog,
			token:   token,
		},
		{
			desc: "issue refresh token with invalid args",
			args: []string{
				refTokCmd,
				"token",
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "issue refresh token with invalid Username",
			args: []string{
				refTokCmd,
				"invalidToken",
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
			token:         mgsdk.Token{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RefreshToken", mock.Anything, mock.Anything).Return(tc.token, tc.sdkErr)

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
	newMetadataJSON := "{\"metadata\":{\"key\": \"value\"}}"
	newPrivateMetadataJSON := "{\"private_metadata\":{\"key\": \"value\"}}"

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "update user tags successfully",
			args: []string{
				userID,
				updateCmd,
				tagUpdateType,
				newTagsJSON,
				validToken,
			},
			sdkErr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user tags with invalid json",
			args: []string{
				userID,
				updateCmd,
				tagUpdateType,
				"[\"tag1\", \"tag2\"",
				validToken,
			},
			sdkErr:        errors.NewSDKError(errEndJSONInput),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errEndJSONInput),
			logType:       errLog,
		},
		{
			desc: "update user tags with invalid token",
			args: []string{
				userID,
				updateCmd,
				tagUpdateType,
				newTagsJSON,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user public metadata successfully",
			args: []string{
				userID,
				updateCmd,
				newPrivateMetadataJSON,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user public metadata with invalid json",
			args: []string{
				userID,
				updateCmd,
				"{\"private_metadata\":{\"key\": \"value\"",
				validToken,
			},
			sdkErr:        errors.NewSDKError(errEndJSONInput),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errEndJSONInput),
			logType:       errLog,
		},
		{
			desc: "update user metadata successfully",
			args: []string{
				userID,
				updateCmd,
				newMetadataJSON,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user metadata with invalid json",
			args: []string{
				userID,
				updateCmd,
				"{\"metadata\":{\"key\": \"value\"",
				validToken,
			},
			sdkErr:        errors.NewSDKError(errEndJSONInput),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errEndJSONInput),
			logType:       errLog,
		},
		{
			desc: "update user email successfully",
			args: []string{
				userID,
				updateCmd,
				emailUpdateType,
				newEmail,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user email with invalid token",
			args: []string{
				userID,
				updateCmd,
				emailUpdateType,
				newEmail,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user successfully",
			args: []string{
				userID,
				updateCmd,
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
				updateCmd,
				newNameMetadataJSON,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user with invalid json",
			args: []string{
				userID,
				updateCmd,
				"{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}",
				validToken,
			},
			sdkErr:        errors.NewSDKError(errEndJSONInput),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errEndJSONInput),
			logType:       errLog,
		},
		{
			desc: "update user role successfully",
			args: []string{
				userID,
				updateCmd,
				roleUpdateType,
				newRole,
				validToken,
			},
			logType: entityLog,
			user:    user,
		},
		{
			desc: "update user role with invalid token",
			args: []string{
				userID,
				updateCmd,
				roleUpdateType,
				newRole,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "update user with invalid args",
			args: []string{
				userID,
				updateCmd,
				roleUpdateType,
				newRole,
				validToken,
				extraArg,
			},
			errLogMessage: "cli users <user_id> update role <role> <user_auth_token>",
			logType:       usageLog,
		},
		{
			desc: "update user without specifying what to update",
			args: []string{
				userID,
				updateCmd,
			},
			errLogMessage: `cli users <user_id> update <JSON_string|tags|username|email|role> [args...]
Available update options:
  cli users <user_id> update <JSON_string> <user_auth_token>
  cli users <user_id> update tags <tags> <user_auth_token>
  cli users <user_id> update username <username> <user_auth_token>
  cli users <user_id> update email <email> <user_auth_token>
  cli users <user_id> update role <role> <user_auth_token>`,
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdateUser", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.sdkErr)
			sdkCall1 := sdkMock.On("UpdateUserTags", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.sdkErr)
			sdkCall2 := sdkMock.On("UpdateUserIdentity", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.sdkErr)
			sdkCall3 := sdkMock.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.sdkErr)
			switch {
			case len(tc.args) > 2 && tc.args[2] == tagUpdateType:
				var u mgsdk.User
				u.Tags = []string{"tag1", "tag2"}
				u.ID = tc.args[0]

				sdkCall1 = sdkMock.On("UpdateUserTags", mock.Anything, u, tc.args[4]).Return(tc.user, tc.sdkErr)
			case len(tc.args) > 2 && tc.args[2] == emailUpdateType:
				var u mgsdk.User
				u.Email = tc.args[3]
				u.ID = tc.args[0]

				sdkCall2 = sdkMock.On("UpdateUserEmail", mock.Anything, u, tc.args[4]).Return(tc.user, tc.sdkErr)
			case len(tc.args) > 2 && tc.args[2] == roleUpdateType && len(tc.args) >= 5:
				sdkCall3 = sdkMock.On("UpdateUserRole", mock.Anything, mgsdk.User{
					Role: tc.args[3],
				}, tc.args[4]).Return(tc.user, tc.sdkErr)
			case len(tc.args) == 4: // Basic user update
				sdkCall = sdkMock.On("UpdateUser", mock.Anything, mgsdk.User{
					FirstName: "new name",
					PrivateMetadata: mgsdk.Metadata{
						"key": "value",
					},
				}, tc.args[3]).Return(tc.user, tc.sdkErr)
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
				assert.True(t, strings.Contains(out, tc.errLogMessage), fmt.Sprintf("%s invalid usage: expected to contain %s, got: %s", tc.desc, tc.errLogMessage, out))
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
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "get user profile successfully",
			args: []string{
				profCmd,
				validToken,
			},
			sdkErr:  nil,
			logType: entityLog,
		},
		{
			desc: "get user profile with invalid args",
			args: []string{
				profCmd,
				validToken,
				extraArg,
			},
			errLogMessage: "cli users profile <user_auth_token>",
			logType:       usageLog,
		},
		{
			desc: "get user profile with invalid token",
			args: []string{
				profCmd,
				"invalid_token_string",
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
		{
			desc: "get user profile with missing token",
			args: []string{
				profCmd,
			},
			errLogMessage: "cli users profile <user_auth_token>",
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var sdkCall *mock.Call
			if len(tc.args) >= 2 {
				sdkCall = sdkMock.On("UserProfile", mock.Anything, tc.args[1]).Return(tc.user, tc.sdkErr)
			}
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.True(t, strings.Contains(out, tc.errLogMessage), fmt.Sprintf("%s invalid usage: expected to contain %s, got: %s", tc.desc, tc.errLogMessage, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &usr)
				assert.Nil(t, err)
				assert.Equal(t, tc.user, usr, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.user, usr))
			}
			if sdkCall != nil {
				sdkCall.Unset()
			}
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
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "request password reset successfully",
			args: []string{
				resPassReqCmd,
				exampleEmail,
			},
			sdkErr:  nil,
			logType: okLog,
		},
		{
			desc: "request password reset with invalid args",
			args: []string{
				resPassReqCmd,
				exampleEmail,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "failed request password reset",
			args: []string{
				resPassReqCmd,
				exampleEmail,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ResetPasswordRequest", mock.Anything, tc.args[1]).Return(tc.sdkErr)
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

func TestResetPasswordCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)
	newPassword := "new-password"

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "reset password successfully",
			args: []string{
				resPassCmd,
				newPassword,
				newPassword,
				validToken,
			},
			sdkErr:  nil,
			logType: okLog,
		},
		{
			desc: "reset password with invalid args",
			args: []string{
				resPassCmd,
				newPassword,
				newPassword,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "reset password with invalid token",
			args: []string{
				resPassCmd,
				newPassword,
				newPassword,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ResetPassword", mock.Anything, tc.args[1], tc.args[2], tc.args[3]).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

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
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "update password successfully",
			args: []string{
				passCmd,
				oldPassword,
				newPassword,
				validToken,
			},
			sdkErr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "reset password with invalid args",
			args: []string{
				passCmd,
				oldPassword,
				newPassword,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			sdkErr:        nil,
			logType:       usageLog,
			user:          user,
		},
		{
			desc: "update password with invalid token",
			args: []string{
				passCmd,
				oldPassword,
				newPassword,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("UpdatePassword", mock.Anything, tc.args[1], tc.args[2], tc.args[3]).Return(tc.user, tc.sdkErr)
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
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "enable user successfully",
			args: []string{
				user.ID,
				enableCmd,
				validToken,
			},
			sdkErr:  nil,
			user:    user,
			logType: entityLog,
		},
		{
			desc: "enable user with invalid args",
			args: []string{
				user.ID,
				enableCmd,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "enable user with invalid token",
			args: []string{
				user.ID,
				enableCmd,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EnableUser", mock.Anything, tc.args[0], tc.args[2]).Return(tc.user, tc.sdkErr)
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
		sdkErr        errors.SDKError
		errLogMessage string
		user          mgsdk.User
		logType       outputLog
	}{
		{
			desc: "disable user successfully",
			args: []string{
				user.ID,
				disableCmd,
				validToken,
			},
			sdkErr:  nil,
			logType: entityLog,
			user:    user,
		},
		{
			desc: "disable user with invalid args",
			args: []string{
				user.ID,
				disableCmd,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "disable user with invalid token",
			args: []string{
				user.ID,
				disableCmd,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DisableUser", mock.Anything, tc.args[0], tc.args[2]).Return(tc.user, tc.sdkErr)
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
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete user successfully",
			args: []string{
				user.ID,
				delCmd,
				validToken,
			},
			logType: okLog,
		},
		{
			desc: "delete user with invalid args",
			args: []string{
				user.ID,
				delCmd,
				validToken,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
		{
			desc: "delete user with invalid token",
			args: []string{
				user.ID,
				delCmd,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with invalid user ID",
			args: []string{
				invalidID,
				delCmd,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with failed to delete",
			args: []string{
				user.ID,
				delCmd,
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity).Error()),
			logType:       errLog,
		},
		{
			desc: "delete user with invalid args",
			args: []string{
				user.ID,
				delCmd,
				extraArg,
			},
			errLogMessage: rootCmd.Use,
			logType:       usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteUser", mock.Anything, mock.Anything, mock.Anything).Return(tc.sdkErr)
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

func TestSearchUsersCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	usersCmd := cli.NewUsersCmd()
	rootCmd := setFlags(usersCmd)

	usersPage := mgsdk.UsersPage{
		Users: []mgsdk.User{user},
		PageRes: mgsdk.PageRes{
			Total:  1,
			Offset: 0,
			Limit:  10,
		},
	}

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		usersPage     mgsdk.UsersPage
		logType       outputLog
	}{
		{
			desc: "search users by username successfully",
			args: []string{
				"search",
				"username=testuser",
				validToken,
			},
			usersPage: usersPage,
			logType:   entityLog,
		},
		{
			desc: "search users with missing token",
			args: []string{
				"search",
				"username=testuser",
			},
			logType: usageLog,
		},
		{
			desc: "search users with missing query",
			args: []string{
				"search",
				validToken,
			},
			logType: usageLog,
		},
		{
			desc: "search users with extra arguments",
			args: []string{
				"search",
				"username=testuser",
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "search users with service error",
			args: []string{
				"search",
				"username=testuser",
				validToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest).Error()),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("SearchUsers", mock.Anything, mock.Anything, mock.Anything).Return(tc.usersPage, tc.sdkErr)
			out := executeCommand(t, rootCmd, tc.args...)

			switch tc.logType {
			case entityLog:
				var page mgsdk.UsersPage
				err := json.Unmarshal([]byte(out), &page)
				assert.Nil(t, err, fmt.Sprintf("unexpected error: %v", err))
				assert.Equal(t, tc.usersPage, page, fmt.Sprintf("%s unexpected response: expected %v got %v", tc.desc, tc.usersPage, page))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}

			sdkCall.Unset()
		})
	}
}
