// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/stretchr/testify/assert"
)

func TestDecodeListGroupsRequest(t *testing.T) {
	cases := []struct {
		desc   string
		url    string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc:   "valid request with no parameters",
			url:    "http://localhost:8080",
			header: map[string][]string{},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}&level=2&parent_id=random&tree=true&dir=-1&member_kind=random&permission=random&list_perms=true",
			header: map[string][]string{
				"Authorization": {"Bearer validToken"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 10,
						Limit:  10,
						Name:   "random",
						Metadata: clients.Metadata{
							"test": "test",
						},
					},
					Level:      2,
					ParentID:   "random",
					Permission: "random",
					Direction:  -1,
					ListPerms:  true,
				},
				token: "validToken",
				tree:  true,
			},
			err: nil,
		},
		{
			desc: "valid request with user",
			url:  "http://localhost:8080?user=valid",
			header: map[string][]string{
				"Authorization": {"Bearer validToken"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 0,
						Limit:  10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
				token:      "validToken",
				memberKind: "users",
				memberID:   "valid",
			},
			err: nil,
		},
		{
			desc: "valid request with group",
			url:  "http://localhost:8080?group=valid",
			header: map[string][]string{
				"Authorization": {"Bearer validToken"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 0,
						Limit:  10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
				token:      "validToken",
				memberKind: "groups",
				memberID:   "valid",
			},
			err: nil,
		},
		{
			desc: "valid request with domain",
			url:  "http://localhost:8080?domain=valid",
			header: map[string][]string{
				"Authorization": {"Bearer validToken"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 0,
						Limit:  10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
				token:      "validToken",
				memberKind: "domains",
				memberID:   "valid",
			},
			err: nil,
		},
		{
			desc: "valid request with channel",
			url:  "http://localhost:8080?channel=valid",
			header: map[string][]string{
				"Authorization": {"Bearer validToken"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 0,
						Limit:  10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
				token:      "validToken",
				memberKind: "channels",
				memberID:   "valid",
			},
			err: nil,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid level",
			url:  "http://localhost:8080?level=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid parent",
			url:  "http://localhost:8080?parent_id=random&parent_id=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid tree",
			url:  "http://localhost:8080?tree=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid dir",
			url:  "http://localhost:8080?dir=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid member kind",
			url:  "http://localhost:8080?member_kind=random",
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
			},
			err: nil,
		},
		{
			desc: "valid request with invalid permission",
			url:  "http://localhost:8080?permission=random&permission=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid list permission",
			url:  "http://localhost:8080?&list_perms=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		parsedURL, err := url.Parse(tc.url)
		assert.NoError(t, err)

		req := &http.Request{
			URL:    parsedURL,
			Header: tc.header,
		}
		resp, err := DecodeListGroupsRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected error %v to contain %v", tc.desc, err, tc.err))
	}
}

func TestDecodeListParentsRequest(t *testing.T) {
	cases := []struct {
		desc   string
		url    string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc:   "valid request with no parameters",
			url:    "http://localhost:8080",
			header: map[string][]string{},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					Permission: api.DefPermission,
					Direction:  +1,
				},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}&level=2&parent_id=random&tree=true&dir=-1&member_kind=random&permission=random&list_perms=true",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 10,
						Limit:  10,
						Name:   "random",
						Metadata: clients.Metadata{
							"test": "test",
						},
					},
					Level:      2,
					Permission: "random",
					Direction:  +1,
					ListPerms:  true,
				},
				token: "123",
				tree:  true,
			},
			err: nil,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid level",
			url:  "http://localhost:8080?level=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid tree",
			url:  "http://localhost:8080?tree=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid permission",
			url:  "http://localhost:8080?permission=random&permission=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid list permission",
			url:  "http://localhost:8080?&list_perms=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		parsedURL, err := url.Parse(tc.url)
		assert.NoError(t, err)

		req := &http.Request{
			URL:    parsedURL,
			Header: tc.header,
		}
		resp, err := DecodeListParentsRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeListChildrenRequest(t *testing.T) {
	cases := []struct {
		desc   string
		url    string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc:   "valid request with no parameters",
			url:    "http://localhost:8080",
			header: map[string][]string{},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					Permission: api.DefPermission,
					Direction:  -1,
				},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}&level=2&parent_id=random&tree=true&dir=-1&member_kind=random&permission=random&list_perms=true",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Status: clients.EnabledStatus,
						Offset: 10,
						Limit:  10,
						Name:   "random",
						Metadata: clients.Metadata{
							"test": "test",
						},
					},
					Level:      2,
					Permission: "random",
					Direction:  -1,
					ListPerms:  true,
				},
				token: "123",
				tree:  true,
			},
			err: nil,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid level",
			url:  "http://localhost:8080?level=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid tree",
			url:  "http://localhost:8080?tree=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid permission",
			url:  "http://localhost:8080?permission=random&permission=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid list permission",
			url:  "http://localhost:8080?&list_perms=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		parsedURL, err := url.Parse(tc.url)
		assert.NoError(t, err)

		req := &http.Request{
			URL:    parsedURL,
			Header: tc.header,
		}
		resp, err := DecodeListChildrenRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeListMembersRequest(t *testing.T) {
	cases := []struct {
		desc   string
		url    string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc:   "valid request with no parameters",
			url:    "http://localhost:8080",
			header: map[string][]string{},
			resp: listMembersReq{
				permission: api.DefPermission,
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?member_kind=random&permission=random",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: listMembersReq{
				token:      "123",
				memberKind: "random",
				permission: "random",
			},
			err: nil,
		},
		{
			desc: "valid request with invalid permission",
			url:  "http://localhost:8080?permission=random&permission=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid member kind",
			url:  "http://localhost:8080?member_kind=random&member_kind=random",
			resp: nil,
			err:  apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		parsedURL, err := url.Parse(tc.url)
		assert.NoError(t, err)

		req := &http.Request{
			URL:    parsedURL,
			Header: tc.header,
		}
		resp, err := DecodeListMembersRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodePageMeta(t *testing.T) {
	cases := []struct {
		desc string
		url  string
		resp groups.PageMeta
		err  error
	}{
		{
			desc: "valid request with no parameters",
			url:  "http://localhost:8080",
			resp: groups.PageMeta{
				Limit: 10,
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}",
			resp: groups.PageMeta{
				Status: clients.EnabledStatus,
				Offset: 10,
				Limit:  10,
				Name:   "random",
				Metadata: clients.Metadata{
					"test": "test",
				},
			},
			err: nil,
		},
		{
			desc: "valid request with invalid status",
			url:  "http://localhost:8080?status=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid status duplicated",
			url:  "http://localhost:8080?status=random&status=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid offset",
			url:  "http://localhost:8080?offset=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid limit",
			url:  "http://localhost:8080?limit=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid name",
			url:  "http://localhost:8080?name=random&name=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
			resp: groups.PageMeta{},
			err:  apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		parsedURL, err := url.Parse(tc.url)
		assert.NoError(t, err)

		req := &http.Request{URL: parsedURL}
		resp, err := decodePageMeta(req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeGroupCreate(t *testing.T) {
	cases := []struct {
		desc   string
		body   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			body: `{"name": "random", "description": "random"}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: createGroupReq{
				Group: groups.Group{
					Name:        "random",
					Description: "random",
				},
				token: "123",
			},
			err: nil,
		},
		{
			desc: "invalid content type",
			body: `{"name": "random", "description": "random"}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {"text/plain"},
			},
			resp: nil,
			err:  apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "invalid request body",
			body: `data`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: nil,
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080", strings.NewReader(tc.body))
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeGroupCreate(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeGroupUpdate(t *testing.T) {
	cases := []struct {
		desc   string
		body   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			body: `{"name": "random", "description": "random"}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: updateGroupReq{
				Name:        "random",
				Description: "random",
				token:       "123",
			},
			err: nil,
		},
		{
			desc: "invalid content type",
			body: `{"name": "random", "description": "random"}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {"text/plain"},
			},
			resp: nil,
			err:  apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "invalid request body",
			body: `data`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: nil,
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPut, "http://localhost:8080", strings.NewReader(tc.body))
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeGroupUpdate(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeGroupRequest(t *testing.T) {
	cases := []struct {
		desc   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: groupReq{
				token: "123",
			},
			err: nil,
		},
		{
			desc: "empty token",
			resp: groupReq{},
			err:  nil,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", http.NoBody)
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeGroupRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeGroupPermsRequest(t *testing.T) {
	cases := []struct {
		desc   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: groupPermsReq{
				token: "123",
			},
			err: nil,
		},
		{
			desc: "empty token",
			resp: groupPermsReq{},
			err:  nil,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", http.NoBody)
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeGroupPermsRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeChangeGroupStatus(t *testing.T) {
	cases := []struct {
		desc   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: changeGroupStatusReq{
				token: "123",
			},
			err: nil,
		},
		{
			desc: "empty token",
			resp: changeGroupStatusReq{},
			err:  nil,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", http.NoBody)
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeChangeGroupStatus(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeAssignMembersRequest(t *testing.T) {
	cases := []struct {
		desc   string
		body   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			body: `{"member_kind": "random", "members": ["random"]}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: assignReq{
				MemberKind: "random",
				Members:    []string{"random"},
				token:      "123",
			},
			err: nil,
		},
		{
			desc: "invalid content type",
			body: `{"member_kind": "random", "members": ["random"]}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {"text/plain"},
			},
			resp: nil,
			err:  apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "invalid request body",
			body: `data`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: nil,
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080", strings.NewReader(tc.body))
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeAssignMembersRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeUnassignMembersRequest(t *testing.T) {
	cases := []struct {
		desc   string
		body   string
		header map[string][]string
		resp   interface{}
		err    error
	}{
		{
			desc: "valid request",
			body: `{"member_kind": "random", "members": ["random"]}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: unassignReq{
				MemberKind: "random",
				Members:    []string{"random"},
				token:      "123",
			},
			err: nil,
		},
		{
			desc: "invalid content type",
			body: `{"member_kind": "random", "members": ["random"]}`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {"text/plain"},
			},
			resp: nil,
			err:  apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "invalid request body",
			body: `data`,
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
				"Content-Type":  {api.ContentType},
			},
			resp: nil,
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080", strings.NewReader(tc.body))
		assert.NoError(t, err)
		req.Header = tc.header
		resp, err := DecodeUnassignMembersRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}
