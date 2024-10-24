// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"strings"
	"testing"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

const (
	valid   = "valid"
	invalid = "invalid"
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
				client: clients.Client{
					ID:   validID,
					Name: valid,
				},
			},
			err: nil,
		},
		{
			desc: "name too long",
			req: createClientReq{
				client: clients.Client{
					ID:   validID,
					Name: strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "invalid id",
			req: createClientReq{
				client: clients.Client{
					ID:   invalid,
					Name: valid,
				},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestCreateClientsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createClientsReq
		err  error
	}{
		{
			desc: "valid request",
			req: createClientsReq{
				Clients: []clients.Client{
					{
						ID:   validID,
						Name: valid,
					},
				},
			},
			err: nil,
		},
		{
			desc: "empty list",
			req: createClientsReq{
				Clients: []clients.Client{},
			},
			err: apiutil.ErrEmptyList,
		},
		{
			desc: "name too long",
			req: createClientsReq{
				Clients: []clients.Client{
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
				Clients: []clients.Client{
					{
						ID:   invalid,
						Name: valid,
					},
				},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: viewClientPermsReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
			desc: "invalid visibility",
			req: listClientsReq{
				limit:      10,
				visibility: "invalid",
			},
			err: apiutil.ErrInvalidVisibilityType,
		},
		{
			desc: "name too long",
			req: listClientsReq{
				limit: 10,
				name:  strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
				groupID: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: listMembersReq{
				groupID: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
		{
			desc: "name too long",
			req: updateClientReq{
				id:   validID,
				Name: strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
				id:     validID,
				Secret: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: updateClientCredentialsReq{
				id:     "",
				Secret: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty secret",
			req: updateClientCredentialsReq{
				id:     validID,
				Secret: "",
			},
			err: apiutil.ErrMissingSecret,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
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
				id: validID,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: deleteClientReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "%s: expected %s got %s\n", tc.desc, tc.err, err)
		})
	}
}
