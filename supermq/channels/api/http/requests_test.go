// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"strings"
	"testing"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/stretchr/testify/assert"
)

func TestCreateChannelReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  createChannelReq
		err  error
	}{
		{
			desc: "valid request",
			req: createChannelReq{
				Channel: channels.Channel{
					Name: valid,
				},
			},
			err: nil,
		},
		{
			desc: "long name",
			req: createChannelReq{
				Channel: channels.Channel{
					Name: strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "missing channel ID",
			req: createChannelReq{
				Channel: channels.Channel{
					ID: "	",
				},
			},
			err: apiutil.ErrMissingChannelID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateChannelsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  createChannelsReq
		err  error
	}{
		{
			desc: "valid request",
			req: createChannelsReq{
				Channels: []channels.Channel{
					{
						Name: valid,
					},
				},
			},
			err: nil,
		},
		{
			desc: "long name",
			req: createChannelsReq{
				Channels: []channels.Channel{
					{
						Name: strings.Repeat("a", api.MaxNameSize+1),
					},
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "missing channel ID",
			req: createChannelsReq{
				Channels: []channels.Channel{
					{
						ID: "	",
					},
				},
			},
			err: apiutil.ErrMissingChannelID,
		},
		{
			desc: "empty list",
			req: createChannelsReq{
				Channels: []channels.Channel{},
			},
			err: apiutil.ErrEmptyList,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewChannelReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  viewChannelReq
		err  error
	}{
		{
			desc: "valid request",
			req: viewChannelReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: viewChannelReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListChannelsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  listChannelsReq
		err  error
	}{
		{
			desc: "valid request",
			req: listChannelsReq{
				limit: 10,
			},
			err: nil,
		},
		{
			desc: "limit is 0",
			req: listChannelsReq{
				limit: 0,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "limit is greater than max limit",
			req: listChannelsReq{
				limit: api.MaxLimitSize + 1,
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "name is too long",
			req: listChannelsReq{
				limit: 10,
				name:  strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateChannelReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateChannelReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateChannelReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: updateChannelReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "name is too long",
			req: updateChannelReq{
				id:   valid,
				Name: strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateChannelTagsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updateChannelTagsReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateChannelTagsReq{
				id:   valid,
				Tags: []string{"tag1", "tag2"},
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: updateChannelTagsReq{
				id:   "",
				Tags: []string{"tag1", "tag2"},
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSetChannelsParentGroupReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  setChannelParentGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: setChannelParentGroupReq{
				id:            valid,
				ParentGroupID: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: setChannelParentGroupReq{
				id:            "",
				ParentGroupID: valid,
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing parent group ID",
			req: setChannelParentGroupReq{
				id:            valid,
				ParentGroupID: "",
			},
			err: apiutil.ErrMissingParentGroupID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveChannelParentGroupReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  removeChannelParentGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: removeChannelParentGroupReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: removeChannelParentGroupReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeChannelStatusReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  changeChannelStatusReq
		err  error
	}{
		{
			desc: "valid request",
			req: changeChannelStatusReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: changeChannelStatusReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnectChannelClientsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  connectChannelClientsRequest
		err  error
	}{
		{
			desc: "valid request",
			req: connectChannelClientsRequest{
				channelID: valid,
				ClientIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: nil,
		},
		{
			desc: "missing channel ID",
			req: connectChannelClientsRequest{
				channelID: "",
				ClientIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing client IDs",
			req: connectChannelClientsRequest{
				channelID: valid,
				ClientIDs: []string{},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing connection types",
			req: connectChannelClientsRequest{
				channelID: valid,
				ClientIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{},
			},
			err: apiutil.ErrMissingConnectionType,
		},
		{
			desc: "invalid client ID",
			req: connectChannelClientsRequest{
				channelID: valid,
				ClientIDs: []string{"client1", "invalid"},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnectChannelClientReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  disconnectChannelClientsRequest
		err  error
	}{
		{
			desc: "valid request",
			req: disconnectChannelClientsRequest{
				channelID: testsutil.GenerateUUID(t),
				ClientIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: nil,
		},
		{
			desc: "missing channel ID",
			req: disconnectChannelClientsRequest{
				channelID: "",
				ClientIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "invalid channel ID",
			req: disconnectChannelClientsRequest{
				channelID: "invalid",
				ClientIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "missing client IDs",
			req: disconnectChannelClientsRequest{
				channelID: testsutil.GenerateUUID(t),
				ClientIds: []string{},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing connection types",
			req: disconnectChannelClientsRequest{
				channelID: testsutil.GenerateUUID(t),
				ClientIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:     []connections.ConnType{},
			},
			err: apiutil.ErrMissingConnectionType,
		},
		{
			desc: "invalid client ID",
			req: disconnectChannelClientsRequest{
				channelID: testsutil.GenerateUUID(t),
				ClientIds: []string{"client1", "invalid"},
				Types:     []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnectReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  connectRequest
		err  error
	}{
		{
			desc: "valid request",
			req: connectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: nil,
		},
		{
			desc: "missing channel IDs",
			req: connectRequest{
				ChannelIds: []string{},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing client IDs",
			req: connectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing connection types",
			req: connectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{},
			},
			err: apiutil.ErrMissingConnectionType,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnectReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  disconnectRequest
		err  error
	}{
		{
			desc: "valid request",
			req: disconnectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: nil,
		},
		{
			desc: "missing channel IDs",
			req: disconnectRequest{
				ChannelIds: []string{},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing client IDs",
			req: disconnectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing connection types",
			req: disconnectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{},
			},
			err: apiutil.ErrMissingConnectionType,
		},
		{
			desc: "invalid client ID",
			req: disconnectRequest{
				ChannelIds: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				ClientIds:  []string{"client1", "invalid"},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "invalid channel ID",
			req: disconnectRequest{
				ChannelIds: []string{"invalid", testsutil.GenerateUUID(t)},
				ClientIds:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				Types:      []connections.ConnType{connections.Publish},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDeleteChannelReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  deleteChannelReq
		err  error
	}{
		{
			desc: "valid request",
			req: deleteChannelReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "missing ID",
			req: deleteChannelReq{
				id: "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
