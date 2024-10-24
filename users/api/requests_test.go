// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/url"
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/users"
	"github.com/stretchr/testify/assert"
)

const (
	valid   = "valid"
	invalid = "invalid"
	secret  = "QJg58*aMan7j"
	name    = "user"
)

var (
	validID = testsutil.GenerateUUID(&testing.T{})
	domain  = testsutil.GenerateUUID(&testing.T{})
)

func TestCreateUserReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createUserReq
		err  error
	}{
		{
			desc: "valid request",
			req: createUserReq{
				user: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     "example@domain.com",
					Credentials: users.Credentials{
						Username: "example",
						Secret:   secret,
					},
				},
			},
			err: nil,
		},
		{
			desc: "name too long",
			req: createUserReq{
				user: users.User{
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
				user: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Credentials: users.Credentials{
						Username: "example",
						Secret:   secret,
					},
				},
			},
			err: apiutil.ErrMissingEmail,
		},
		{
			desc: "missing secret in request",
			req: createUserReq{
				user: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     "example@domain.com",
					Credentials: users.Credentials{
						Username: "example",
					},
				},
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "invalid secret in request",
			req: createUserReq{
				user: users.User{
					ID:        validID,
					FirstName: valid,
					LastName:  valid,
					Email:     "example@domain.com",
					Credentials: users.Credentials{
						Username: "example",
						Secret:   "invalid",
					},
				},
			},
			err: apiutil.ErrPasswordFormat,
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

func TestListMembersByObjectReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listMembersByObjectReq
		err  error
	}{
		{
			desc: "valid request",
			req: listMembersByObjectReq{
				objectKind: "group",
				objectID:   validID,
			},
			err: nil,
		},
		{
			desc: "empty object kind",
			req: listMembersByObjectReq{
				objectKind: "",
				objectID:   validID,
			},
			err: apiutil.ErrMissingMemberKind,
		},
		{
			desc: "empty object id",
			req: listMembersByObjectReq{
				objectKind: "group",
				objectID:   "",
			},
			err: apiutil.ErrMissingID,
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
				id:       validID,
				Username: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserReq{
				id:       "",
				Username: valid,
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
	cases := []struct {
		desc string
		req  updateUserTagsReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserTagsReq{
				id:   validID,
				Tags: []string{"tag1", "tag2"},
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserTagsReq{
				id:   "",
				Tags: []string{"tag1", "tag2"},
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
	cases := []struct {
		desc string
		req  updateProfilePictureReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateProfilePictureReq{
				id:             validID,
				ProfilePicture: *parsedURL,
			},
			err: nil,
		},
		{
			desc: "empty ID",
			req: updateProfilePictureReq{
				id:             "",
				ProfilePicture: *parsedURL,
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
		req  updateUserEmailReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateUserEmailReq{
				id:    validID,
				Email: "example@example.com",
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateUserEmailReq{
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
			desc: "valid request",
			req: loginUserReq{
				Email:  "eaxmple,example.com",
				Secret: secret,
			},
			err: nil,
		},
		{
			desc: "empty email",
			req: loginUserReq{
				Email:  "",
				Secret: secret,
			},
			err: apiutil.ErrMissingEmail,
		},
		{
			desc: "empty secret",
			req: loginUserReq{
				Secret: "",
				Email:  "eaxmple,example.com",
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
		req  passwResetReq
		err  error
	}{
		{
			desc: "valid request",
			req: passwResetReq{
				Email: "example@example.com",
				Host:  "example.com",
			},
			err: nil,
		},
		{
			desc: "empty email",
			req: passwResetReq{
				Email: "",
				Host:  "example.com",
			},
			err: apiutil.ErrMissingEmail,
		},
		{
			desc: "empty host",
			req: passwResetReq{
				Email: "example@example.com",
				Host:  "",
			},
			err: apiutil.ErrMissingHost,
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
				Token:    valid,
				Password: secret,
				ConfPass: secret,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: resetTokenReq{
				Token:    "",
				Password: secret,
				ConfPass: secret,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty password",
			req: resetTokenReq{
				Token:    valid,
				Password: "",
				ConfPass: secret,
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "empty confpass",
			req: resetTokenReq{
				Token:    valid,
				Password: secret,
				ConfPass: "",
			},
			err: apiutil.ErrMissingConfPass,
		},
		{
			desc: "mismatching password and confpass",
			req: resetTokenReq{
				Token:    valid,
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

func TestAssignUsersRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  assignUsersReq
		err  error
	}{
		{
			desc: "valid request",
			req: assignUsersReq{
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: assignUsersReq{
				groupID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty users",
			req: assignUsersReq{
				groupID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty relation",
			req: assignUsersReq{
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: "",
			},
			err: apiutil.ErrMissingRelation,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUnassignUsersRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  unassignUsersReq
		err  error
	}{
		{
			desc: "valid request",
			req: unassignUsersReq{
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: unassignUsersReq{
				groupID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty users",
			req: unassignUsersReq{
				groupID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty relation",
			req: unassignUsersReq{
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: "",
			},
			err: nil,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestAssignGroupsRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  assignGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: assignGroupsReq{
				domainID: domain,
				groupID:  validID,
				GroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty group id",
			req: assignGroupsReq{
				domainID: domain,
				groupID:  "",
				GroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: assignGroupsReq{
				domainID: domain,
				groupID:  validID,
				GroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty domain id",
			req: assignGroupsReq{
				domainID: "",
				groupID:  validID,
				GroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingDomainID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUnassignGroupsRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  unassignGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: unassignGroupsReq{
				domainID: domain,
				groupID:  validID,
				GroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty group id",
			req: unassignGroupsReq{
				domainID: domain,
				groupID:  "",
				GroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: unassignGroupsReq{
				domainID: domain,
				groupID:  validID,
				GroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty domain id",
			req: unassignGroupsReq{
				domainID: "",
				groupID:  validID,
				GroupIDs: []string{valid},
			},
			err: apiutil.ErrMissingDomainID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}
