// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/url"
	"strings"
	"testing"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/users"
	"github.com/stretchr/testify/assert"
)

const (
	valid      = "valid"
	secret     = "QJg58*aMan7j"
	name       = "user"
	validEmail = "example@domain.com"
)

var validID = testsutil.GenerateUUID(&testing.T{})

func TestCreateUserReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createUserReq
		err  error
	}{
		{
			desc: "valid request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: valid,
						Secret:   secret,
					},
				},
			},
			err: nil,
		},
		{
			desc: "name too long",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: strings.Repeat("a", api.MaxNameSize+1),
					LastName:  valid,
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "missing email in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Credentials: users.Credentials{
						Username: valid,
						Secret:   secret,
					},
				},
			},
			err: apiutil.ErrMissingEmail,
		},
		{
			desc: "missing secret in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: valid,
					},
				},
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "invalid secret in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: valid,
						Secret:   "invalid",
					},
				},
			},
			err: apiutil.ErrPasswordFormat,
		},
		{
			desc: "missing username in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: "",
						Secret:   secret,
					},
				},
			},
			err: apiutil.ErrMissingUsername,
		},
		{
			desc: "username that is too long in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: strings.Repeat("a", 33),
						Secret:   secret,
					},
				},
			},
			err: apiutil.ErrInvalidUsername,
		},
		{
			desc: "invalid username format in request",
			req: createUserReq{
				User: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     validEmail,
					Credentials: users.Credentials{
						Username: "_invalid@username",
						Secret:   secret,
					},
				},
			},
			err: apiutil.ErrInvalidUsername,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
	}
}

func TestViewUserReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  viewUserReq
		err  error
	}{
		{
			desc: "valid request",
			req: viewUserReq{
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: viewUserReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestListUsersReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listUsersReq
		err  error
	}{
		{
			desc: "valid request",
			req: listUsersReq{
				limit: 10,
			},
			err: nil,
		},
		{
			desc: "limit too big",
			req: listUsersReq{
				limit: api.MaxLimitSize + 1,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "limit too small",
			req: listUsersReq{
				limit: 0,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "invalid direction",
			req: listUsersReq{
				limit: 10,
				dir:   "invalid",
			},
			err: apiutil.ErrInvalidDirection,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestSearchUsersReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  searchUsersReq
		err  error
	}{
		{
			desc: "valid request",
			req: searchUsersReq{
				Username: name,
			},
			err: nil,
		},
		{
			desc: "empty query",
			req:  searchUsersReq{},
			err:  apiutil.ErrEmptySearchQuery,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err)
	}
}

func TestUpdateUserReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateUserReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserReq{
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateUserTagsReqValidate(t *testing.T) {
	tags := []string{"tag1", "tag2"}
	cases := []struct {
		desc string
		req  updateUserTagsReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserTagsReq{
				id:   validID,
				Tags: &tags,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserTagsReq{
				id:   "",
				Tags: &tags,
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateUsernameReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateUsernameReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUsernameReq{
				id:       validID,
				Username: "validUsername",
			},
			err: nil,
		},
		{
			desc: "missing user ID",
			req: updateUsernameReq{
				id:       "",
				Username: "validUsername",
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "name too long",
			req: updateUsernameReq{
				id:       validID,
				Username: strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
	}
}

func TestUpdateProfilePictureReqValidate(t *testing.T) {
	base64EncodedString := "https://example.com/profile.jpg"

	parsedURL, err := url.Parse(base64EncodedString)
	if err != nil {
		t.Fatalf("Error parsing URL: %v", err)
	}
	url := parsedURL.String()
	cases := []struct {
		desc string
		req  updateProfilePictureReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateProfilePictureReq{
				id:             validID,
				ProfilePicture: &url,
			},
			err: nil,
		},
		{
			desc: "empty ID",
			req: updateProfilePictureReq{
				id:             "",
				ProfilePicture: &url,
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
	}
}

func TestUpdateUserRoleReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateUserRoleReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserRoleReq{
				id:   validID,
				Role: "admin",
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserRoleReq{
				id:   "",
				Role: "admin",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateUserEmailReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateEmailReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateEmailReq{
				id:    validID,
				Email: "example@example.com",
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateEmailReq{
				id:    "",
				Email: "example@example.com",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateUserSecretReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateUserSecretReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserSecretReq{
				OldSecret: secret,
				NewSecret: secret,
			},
			err: nil,
		},
		{
			desc: "missing old secret",
			req: updateUserSecretReq{
				OldSecret: "",
				NewSecret: secret,
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "missing new secret",
			req: updateUserSecretReq{
				OldSecret: secret,
				NewSecret: "",
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "invalid new secret",
			req: updateUserSecretReq{
				OldSecret: secret,
				NewSecret: "invalid",
			},
			err: apiutil.ErrPasswordFormat,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err)
	}
}

func TestChangeUserStatusReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  changeUserStatusReq
		err  error
	}{
		{
			desc: "valid request",
			req: changeUserStatusReq{
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: changeUserStatusReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestLoginUserReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  loginUserReq
		err  error
	}{
		{
			desc: "valid request with identity",
			req: loginUserReq{
				Username: "example",
				Password: secret,
			},
			err: nil,
		},
		{
			desc: "empty identity",
			req: loginUserReq{
				Username: "",
				Password: secret,
			},
			err: apiutil.ErrMissingUsernameEmail,
		},
		{
			desc: "empty secret",
			req: loginUserReq{
				Password: "",
				Username: "example",
			},
			err: apiutil.ErrMissingPass,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestTokenReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  tokenReq
		err  error
	}{
		{
			desc: "valid request",
			req: tokenReq{
				RefreshToken: valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: tokenReq{
				RefreshToken: "",
			},
			err: apiutil.ErrBearerToken,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestPasswResetReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  passResetReq
		err  error
	}{
		{
			desc: "valid request",
			req: passResetReq{
				Email: "example@example.com",
			},
			err: nil,
		},
		{
			desc: "empty email",
			req: passResetReq{
				Email: "",
			},
			err: apiutil.ErrMissingEmail,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestResetTokenReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  resetTokenReq
		err  error
	}{
		{
			desc: "valid request",
			req: resetTokenReq{
				Password: secret,
				ConfPass: secret,
			},
			err: nil,
		},
		{
			desc: "empty password",
			req: resetTokenReq{
				Password: "",
				ConfPass: secret,
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "empty confpass",
			req: resetTokenReq{
				Password: secret,
				ConfPass: "",
			},
			err: apiutil.ErrMissingConfPass,
		},
		{
			desc: "mismatching password and confpass",
			req: resetTokenReq{
				Password: "secret",
				ConfPass: secret,
			},
			err: apiutil.ErrInvalidResetPass,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err)
	}
}
