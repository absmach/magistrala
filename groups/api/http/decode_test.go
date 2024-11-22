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

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
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
				PageMeta: groups.PageMeta{
					Limit:   10,
					Actions: []string{},
				},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}&level=2&t&permission=random&list_perms=true",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: listGroupsReq{
				PageMeta: groups.PageMeta{
					Status: groups.EnabledStatus,
					Offset: 10,
					Limit:  10,
					Name:   "random",
					Metadata: groups.Metadata{
						"test": "test",
					},
					Actions: []string{},
				},
			},
			err: nil,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}

func TestDecodeRetrieveGroupHierarchy(t *testing.T) {
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
			resp: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Direction: -1,
				},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?tree=true&level=2&dir=-1",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     2,
					Direction: -1,
					Tree:      true,
				},
			},
			err: nil,
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
			desc: "valid request with invalid direction",
			url:  "http://localhost:8080?dir=random",
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
		resp, err := decodeRetrieveGroupHierarchy(context.Background(), req)
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
			resp: listChildrenGroupsReq{
				startLevel: 1,
				endLevel:   0,
				PageMeta: groups.PageMeta{
					Limit:   10,
					Actions: []string{},
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
			resp: listChildrenGroupsReq{
				startLevel: 1,
				endLevel:   0,
				PageMeta: groups.PageMeta{
					Status: groups.EnabledStatus,
					Offset: 10,
					Limit:  10,
					Name:   "random",
					Metadata: groups.Metadata{
						"test": "test",
					},
					Actions: []string{},
				},
			},
			err: nil,
		},
		{
			desc: "valid request with invalid page metadata",
			url:  "http://localhost:8080?metadata=random",
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
		resp, err := decodeListChildrenGroupsRequest(context.Background(), req)
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
				Limit:   10,
				Actions: []string{},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}",
			resp: groups.PageMeta{
				Status: groups.EnabledStatus,
				Offset: 10,
				Limit:  10,
				Name:   "random",
				Metadata: groups.Metadata{
					"test": "test",
				},
				Actions: []string{},
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
			resp: groupReq{},
			err:  nil,
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
			resp: changeGroupStatusReq{},
			err:  nil,
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
		resp, err := DecodeChangeGroupStatusRequest(context.Background(), req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
	}
}
