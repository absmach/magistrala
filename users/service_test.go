// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"

	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wrong string = "wrong-value"

var (
	user            = users.User{Email: "user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	nonExistingUser = users.User{Email: "non-ex-user@example.com", Password: "password", Metadata: map[string]interface{}{"role": "user"}}
	host            = "example.com"
	groupName       = "Mainflux"
)

func newService() users.Service {
	userRepo := mocks.NewUserRepository()
	groupRepo := mocks.NewGroupRepository()
	hasher := mocks.NewHasher()
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	e := mocks.NewEmailer()

	return users.New(userRepo, groupRepo, hasher, auth, e)
}

func TestRegister(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "register new user",
			user: user,
			err:  nil,
		},
		{
			desc: "register existing user",
			user: user,
			err:  users.ErrConflict,
		},
		{
			desc: "register new user with empty password",
			user: users.User{
				Email:    user.Email,
				Password: "",
			},
			err: users.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := svc.Register(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	noAuthUser := users.User{
		Email:    "email@test.com",
		Password: "12345678",
	}

	cases := map[string]struct {
		user users.User
		err  error
	}{
		"login with good credentials": {
			user: user,
			err:  nil,
		},
		"login with wrong e-mail": {
			user: users.User{
				Email:    wrong,
				Password: user.Password,
			},
			err: users.ErrUnauthorizedAccess,
		},
		"login with wrong password": {
			user: users.User{
				Email:    user.Email,
				Password: wrong,
			},
			err: users.ErrUnauthorizedAccess,
		},
		"login failed auth": {
			user: noAuthUser,
			err:  users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewUser(t *testing.T) {
	svc := newService()
	id, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	u := user
	u.Password = ""

	cases := map[string]struct {
		user   users.User
		token  string
		userID string
		err    error
	}{
		"view user with authorized token": {
			user:   u,
			token:  token,
			userID: id,
			err:    nil,
		},
		"view user with unauthorized token": {
			user:   users.User{},
			token:  "",
			userID: id,
			err:    users.ErrUnauthorizedAccess,
		},
		"view user with authorized token and invalid user id": {
			user:   users.User{},
			token:  token,
			userID: "",
			err:    users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewUser(context.Background(), tc.token, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewProfile(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	u := user
	u.Password = ""

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"valid token's user info": {
			user:  u,
			token: token,
			err:   nil,
		},
		"invalid token's user info": {
			user:  users.User{},
			token: "",
			err:   users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewProfile(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
func TestListUsers(t *testing.T) {
	svc := newService()

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var nUsers = uint64(10)

	for i := uint64(1); i < nUsers; i++ {
		email := fmt.Sprintf("TestListUsers%d@example.com", i)
		user := users.User{
			Email:    email,
			Password: "passpass",
		}
		_, err := svc.Register(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		token  string
		offset uint64
		limit  uint64
		email  string
		size   uint64
		err    error
	}{
		"list users with authorized token": {
			token: token,
			size:  0,
			err:   nil,
		},
		"list user with unauthorized token": {
			token: "",
			size:  0,
			err:   users.ErrUnauthorizedAccess,
		},
		"list users with offset and limit": {
			token:  token,
			offset: 6,
			limit:  nUsers,
			size:   nUsers - 6,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListUsers(context.Background(), tc.token, tc.offset, tc.limit, tc.email, nil)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateUser(t *testing.T) {
	svc := newService()

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	token, err := svc.Login(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user.Metadata = map[string]interface{}{"role": "test"}

	cases := map[string]struct {
		user  users.User
		token string
		err   error
	}{
		"update user with valid token": {
			user:  user,
			token: token,
			err:   nil,
		},
		"update user with invalid token": {
			user:  user,
			token: "non-existent",
			err:   users.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		err := svc.UpdateUser(context.Background(), tc.token, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		email string
		err   error
	}{
		"valid user reset token":  {user.Email, nil},
		"invalid user rest token": {nonExistingUser.Email, users.ErrUserNotFound},
	}

	for desc, tc := range cases {
		err := svc.GenerateResetToken(context.Background(), tc.email, host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChangePassword(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user error: %s", err))
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token       string
		password    string
		oldPassword string
		err         error
	}{
		"valid user change password ":                    {token, "newpassword", user.Password, nil},
		"valid user change password with wrong password": {token, "newpassword", "wrongpassword", users.ErrUnauthorizedAccess},
		"valid user change password invalid token":       {"", "newpassword", user.Password, users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		err := svc.ChangePassword(context.Background(), tc.token, tc.password, tc.oldPassword)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestResetPassword(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	resetToken, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 2})
	assert.Nil(t, err, fmt.Sprintf("Generating reset token expected to succeed: %s", err))
	cases := map[string]struct {
		token    string
		password string
		err      error
	}{
		"valid user reset password ":   {resetToken.GetValue(), user.Email, nil},
		"invalid user reset password ": {"", "newpassword", users.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		err := svc.ResetPassword(context.Background(), tc.token, tc.password)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSendPasswordReset(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user error: %s", err))
	token, _ := svc.Login(context.Background(), user)

	cases := map[string]struct {
		token string
		email string
		err   error
	}{
		"valid user reset password ": {token, user.Email, nil},
	}

	for desc, tc := range cases {
		err := svc.SendPasswordReset(context.Background(), host, tc.email, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))

	}
}

func TestCreateGroup(t *testing.T) {
	svc := newService()
	_, err := svc.Register(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("registering user expected to succeed: %s", err))

	token, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("authenticating user expected to succeed: %s", err))

	uuid, err := uuidProvider.New().ID()
	assert.Nil(t, err, fmt.Sprintf("generating uuid expected to succeed: %s", err))

	group := users.Group{
		ID:   uuid,
		Name: groupName,
	}

	cases := []struct {
		desc  string
		group users.Group
		err   error
	}{
		{
			desc:  "create new group",
			group: group,
			err:   nil,
		},
		{
			desc:  "create group with existing name",
			group: group,
			err:   users.ErrGroupConflict,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateGroup(context.Background(), token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService()

	_, err := svc.Register(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("registering user expected to succeed: %s", err))

	token, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("authenticating user expected to succeed: %s", err))

	group := users.Group{
		Name: groupName,
	}

	saved, err := svc.CreateGroup(context.Background(), token, group)
	assert.Nil(t, err, fmt.Sprintf("generating uuid expected to succeed: %s", err))

	group.Description = "test description"
	group.Name = "NewName"
	group.ID = saved.ID
	group.OwnerID = saved.OwnerID

	cases := []struct {
		desc  string
		group users.Group
		err   error
	}{
		{
			desc:  "update group",
			group: group,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateGroup(context.Background(), token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		g, err := svc.ViewGroup(context.Background(), token, saved.ID)
		assert.Nil(t, err, fmt.Sprintf("retrieve group failed: %s", err))
		assert.Equal(t, tc.group.Description, g.Description, tc.desc, tc.err)
		assert.Equal(t, tc.group.Name, g.Name, tc.desc, tc.err)
		assert.Equal(t, tc.group.ID, g.ID, tc.desc, tc.err)
		assert.Equal(t, tc.group.OwnerID, g.OwnerID, tc.desc, tc.err)
	}
}

func TestRemoveGroup(t *testing.T) {
	svc := newService()

	_, err := svc.Register(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("registering user expected to succeed: %s", err))

	token, err := svc.Login(context.Background(), user)
	assert.Nil(t, err, fmt.Sprintf("authenticating user expected to succeed: %s", err))

	group := users.Group{
		Name: groupName,
	}

	saved, err := svc.CreateGroup(context.Background(), token, group)
	assert.Nil(t, err, fmt.Sprintf("generating uuid expected to succeed: %s", err))

	group.Description = "test description"
	group.Name = "NewName"
	group.ID = saved.ID
	group.OwnerID = saved.OwnerID

	cases := []struct {
		desc  string
		group users.Group
		err   error
	}{
		{
			desc:  "remove existing group",
			group: group,
			err:   nil,
		},
		{
			desc:  "remove non existing group",
			group: group,
			err:   users.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveGroup(context.Background(), token, tc.group.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
