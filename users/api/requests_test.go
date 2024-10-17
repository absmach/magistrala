// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/stretchr/testify/assert"
)

const (
	valid   = "valid"
	invalid = "invalid"
	secret  = "QJg58*aMan7j"
	name    = "client"
)

var validID = testsutil.GenerateUUID(&testing.T{})

func TestCreateClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: createClientReq{
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
					Credentials: mgclients.Credentials{
						Identity: "example@example.com",
						Secret:   secret,
					},
				},
			},
			err: nil,
		},
		{
			desc: "name too long",
			req: createClientReq{
				client: mgclients.Client{
					ID:   validID,
					Name: strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "missing identity in request",
			req: createClientReq{
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
					Credentials: mgclients.Credentials{
						Secret: valid,
					},
				},
			},
			err: apiutil.ErrMissingIdentity,
		},
		{
			desc: "missing secret in request",
			req: createClientReq{
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
					Credentials: mgclients.Credentials{
						Identity: "example@example.com",
					},
				},
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "invalid secret in request",
			req: createClientReq{
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
					Credentials: mgclients.Credentials{
						Identity: "example@example.com",
						Secret:   "invalid",
					},
				},
			},
			err: apiutil.ErrPasswordFormat,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err)
	}
}

func TestViewClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  viewClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: viewClientReq{
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: viewClientReq{
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

func TestListClientsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listClientsReq
		err  error
	}{
		{
			desc: "valid request",
			req: listClientsReq{
				limit: 10,
			},
			err: nil,
		},
		{
			desc: "limit too big",
			req: listClientsReq{
				limit: api.MaxLimitSize + 1,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "limit too small",
			req: listClientsReq{
				limit: 0,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "invalid direction",
			req: listClientsReq{
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

func TestSearchClientsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  searchClientsReq
		err  error
	}{
		{
			desc: "valid request",
			req: searchClientsReq{
				Name: name,
			},
			err: nil,
		},
		{
			desc: "empty query",
			req:  searchClientsReq{},
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

func TestUpdateClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientReq{
				id:   validID,
				Name: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateClientReq{
				id:   "",
				Name: valid,
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateClientTagsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientTagsReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientTagsReq{
				id:   validID,
				Tags: []string{"tag1", "tag2"},
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateClientTagsReq{
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

func TestUpdateClientRoleReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientRoleReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientRoleReq{
				id:   validID,
				Role: "admin",
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateClientRoleReq{
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

func TestUpdateClientIdentityReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientIdentityReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientIdentityReq{
				id:       validID,
				Identity: "example@example.com",
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateClientIdentityReq{
				id:       "",
				Identity: "example@example.com",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateClientSecretReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientSecretReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientSecretReq{
				OldSecret: secret,
				NewSecret: secret,
			},
			err: nil,
		},
		{
			desc: "missing old secret",
			req: updateClientSecretReq{
				OldSecret: "",
				NewSecret: secret,
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "missing new secret",
			req: updateClientSecretReq{
				OldSecret: secret,
				NewSecret: "",
			},
			err: apiutil.ErrMissingPass,
		},
		{
			desc: "invalid new secret",
			req: updateClientSecretReq{
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

func TestChangeClientStatusReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  changeClientStatusReq
		err  error
	}{
		{
			desc: "valid request",
			req: changeClientStatusReq{
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: changeClientStatusReq{
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

func TestLoginClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  loginClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: loginClientReq{
				Identity: "eaxmple,example.com",
				Secret:   secret,
			},
			err: nil,
		},
		{
			desc: "empty identity",
			req: loginClientReq{
				Identity: "",
				Secret:   secret,
			},
			err: apiutil.ErrMissingIdentity,
		},
		{
			desc: "empty secret",
			req: loginClientReq{
				Identity: "eaxmple,example.com",
				Secret:   "",
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
				groupID:  validID,
				GroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty group id",
			req: assignGroupsReq{
				groupID:  "",
				GroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: assignGroupsReq{
				groupID:  validID,
				GroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
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
				groupID:  validID,
				GroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty group id",
			req: unassignGroupsReq{
				groupID:  "",
				GroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: unassignGroupsReq{
				groupID:  validID,
				GroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}
