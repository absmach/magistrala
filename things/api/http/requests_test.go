// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

const (
	valid   = "valid"
	invalid = "invalid"
)

var validID = testsutil.GenerateUUID(&testing.T{})

func TestCreateThingReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: createClientReq{
				token: valid,
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
				},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: createClientReq{
				token: "",
				client: mgclients.Client{
					ID:   validID,
					Name: valid,
				},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "name too long",
			req: createClientReq{
				token: valid,
				client: mgclients.Client{
					ID:   validID,
					Name: strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "invalid id",
			req: createClientReq{
				token: valid,
				client: mgclients.Client{
					ID:   invalid,
					Name: valid,
				},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err)
	}
}

func TestCreateThingsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createClientsReq
		err  error
	}{
		{
			desc: "valid request",
			req: createClientsReq{
				token: valid,
				Clients: []mgclients.Client{
					{
						ID:   validID,
						Name: valid,
					},
				},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: createClientsReq{
				token: "",
				Clients: []mgclients.Client{
					{
						ID:   validID,
						Name: valid,
					},
				},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty list",
			req: createClientsReq{
				token:   valid,
				Clients: []mgclients.Client{},
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "name too long",
			req: createClientsReq{
				token: valid,
				Clients: []mgclients.Client{
					{
						ID:   validID,
						Name: strings.Repeat("a", api.MaxNameSize+1),
					},
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "invalid id",
			req: createClientsReq{
				token: valid,
				Clients: []mgclients.Client{
					{
						ID:   invalid,
						Name: valid,
					},
				},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
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
				token: valid,
				id:    validID,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: viewClientReq{
				token: "",
				id:    validID,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: viewClientReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestViewClientPermsReq(t *testing.T) {
	cases := []struct {
		desc string
		req  viewClientPermsReq
		err  error
	}{
		{
			desc: "valid request",
			req: viewClientPermsReq{
				token: valid,
				id:    validID,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: viewClientPermsReq{
				token: "",
				id:    validID,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: viewClientPermsReq{
				token: valid,
				id:    "",
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
				token: valid,
				page: mgclients.Page{
					Limit: 10,
				},
			},
			err: nil,
		},
		{
			desc: "limit too big",
			req: listClientsReq{
				token: valid,
				page: mgclients.Page{
					Limit: api.MaxLimitSize + 1,
				},
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "limit too small",
			req: listClientsReq{
				token: valid,
				page: mgclients.Page{
					Limit: 0,
				},
			},
			err: apiutil.ErrLimitSize,
		},

		{
			desc: "name too long",
			req: listClientsReq{
				token: valid,
				page: mgclients.Page{
					Limit: 10,
					Name:  strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestListMembersReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listMembersReq
		err  error
	}{
		{
			desc: "valid request",
			req: listMembersReq{
				token:   valid,
				groupID: validID,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: listMembersReq{
				token:   "",
				groupID: validID,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: listMembersReq{
				token:   valid,
				groupID: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
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
				token: valid,
				id:    validID,
				Name:  valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: updateClientReq{
				token: "",
				id:    validID,
				Name:  valid,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: updateClientReq{
				token: valid,
				id:    "",
				Name:  valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "name too long",
			req: updateClientReq{
				token: valid,
				id:    validID,
				Name:  strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
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
				token: valid,
				id:    validID,
				Tags:  []string{"tag1", "tag2"},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: updateClientTagsReq{
				token: "",
				id:    validID,
				Tags:  []string{"tag1", "tag2"},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: updateClientTagsReq{
				token: valid,
				id:    "",
				Tags:  []string{"tag1", "tag2"},
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUpdateClientCredentialsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateClientCredentialsReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateClientCredentialsReq{
				token:  valid,
				id:     validID,
				Secret: valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: updateClientCredentialsReq{
				token:  "",
				id:     validID,
				Secret: valid,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: updateClientCredentialsReq{
				token:  valid,
				id:     "",
				Secret: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty secret",
			req: updateClientCredentialsReq{
				token:  valid,
				id:     validID,
				Secret: "",
			},
			err: apiutil.ErrMissingSecret,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
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
				token: valid,
				id:    validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: changeClientStatusReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestAssignUsersRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  assignUsersRequest
		err  error
	}{
		{
			desc: "valid request",
			req: assignUsersRequest{
				token:    valid,
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: assignUsersRequest{
				token:    "",
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: assignUsersRequest{
				token:    valid,
				groupID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty users",
			req: assignUsersRequest{
				token:    valid,
				groupID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty relation",
			req: assignUsersRequest{
				token:    valid,
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
		req  unassignUsersRequest
		err  error
	}{
		{
			desc: "valid request",
			req: unassignUsersRequest{
				token:    valid,
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: unassignUsersRequest{
				token:    "",
				groupID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: unassignUsersRequest{
				token:    valid,
				groupID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty users",
			req: unassignUsersRequest{
				token:    valid,
				groupID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "empty relation",
			req: unassignUsersRequest{
				token:    valid,
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

func TestAssignUserGroupsRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  assignUserGroupsRequest
		err  error
	}{
		{
			desc: "valid request",
			req: assignUserGroupsRequest{
				token:        valid,
				groupID:      validID,
				UserGroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: assignUserGroupsRequest{
				token:        "",
				groupID:      validID,
				UserGroupIDs: []string{validID},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty group id",
			req: assignUserGroupsRequest{
				token:        valid,
				groupID:      "",
				UserGroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: assignUserGroupsRequest{
				token:        valid,
				groupID:      validID,
				UserGroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestUnassignUserGroupsRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  unassignUserGroupsRequest
		err  error
	}{
		{
			desc: "valid request",
			req: unassignUserGroupsRequest{
				token:        valid,
				groupID:      validID,
				UserGroupIDs: []string{validID},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: unassignUserGroupsRequest{
				token:        "",
				groupID:      validID,
				UserGroupIDs: []string{validID},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty group id",
			req: unassignUserGroupsRequest{
				token:        valid,
				groupID:      "",
				UserGroupIDs: []string{validID},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty user group ids",
			req: unassignUserGroupsRequest{
				token:        valid,
				groupID:      validID,
				UserGroupIDs: []string{},
			},
			err: apiutil.ErrEmptyList,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestConnectChannelThingRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  connectChannelThingRequest
		err  error
	}{
		{
			desc: "valid request",
			req: connectChannelThingRequest{
				token:     valid,
				ChannelID: validID,
				ThingID:   validID,
			},
			err: nil,
		},
		{
			desc: "empty channel id",
			req: connectChannelThingRequest{
				token:     valid,
				ChannelID: "",
				ThingID:   validID,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty thing id",
			req: connectChannelThingRequest{
				token:     valid,
				ChannelID: validID,
				ThingID:   "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestDisconnectChannelThingRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  disconnectChannelThingRequest
		err  error
	}{
		{
			desc: "valid request",
			req: disconnectChannelThingRequest{
				token:     valid,
				ChannelID: validID,
				ThingID:   validID,
			},
			err: nil,
		},
		{
			desc: "empty channel id",
			req: disconnectChannelThingRequest{
				token:     valid,
				ChannelID: "",
				ThingID:   validID,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty thing id",
			req: disconnectChannelThingRequest{
				token:     valid,
				ChannelID: validID,
				ThingID:   "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestThingShareRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  thingShareRequest
		err  error
	}{
		{
			desc: "valid request",
			req: thingShareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty thing id",
			req: thingShareRequest{
				token:    valid,
				thingID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "empty user ids",
			req: thingShareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: svcerr.ErrCreateEntity,
		},
		{
			desc: "empty relation",
			req: thingShareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{validID},
				Relation: "",
			},
			err: svcerr.ErrCreateEntity,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestThingUnshareRequestValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  thingUnshareRequest
		err  error
	}{
		{
			desc: "valid request",
			req: thingUnshareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: nil,
		},
		{
			desc: "empty thing id",
			req: thingUnshareRequest{
				token:    valid,
				thingID:  "",
				UserIDs:  []string{validID},
				Relation: valid,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "empty user ids",
			req: thingUnshareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{},
				Relation: valid,
			},
			err: svcerr.ErrCreateEntity,
		},
		{
			desc: "empty relation",
			req: thingUnshareRequest{
				token:    valid,
				thingID:  validID,
				UserIDs:  []string{validID},
				Relation: "",
			},
			err: svcerr.ErrCreateEntity,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}

func TestDeleteClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  deleteClientReq
		err  error
	}{
		{
			desc: "valid request",
			req: deleteClientReq{
				token: valid,
				id:    validID,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: deleteClientReq{
				token: "",
				id:    validID,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: deleteClientReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, c := range cases {
		err := c.req.validate()
		assert.Equal(t, c.err, err, "%s: expected %s got %s\n", c.desc, c.err, err)
	}
}
