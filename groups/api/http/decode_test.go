// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
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
					Limit: 10,
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
				PageMeta: groups.PageMeta{
					Status: groups.EnabledStatus,
					Offset: 10,
					Limit:  10,
					Name:   "random",
					Metadata: groups.Metadata{
						"test": "test",
					},
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
			url:  "http://localhost:8080?member_kind=random&member_kind=random",
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
				HierarchyPageMeta: groups.HierarchyPageMeta{},
			},
			err: nil,
		},
		{
			desc: "valid request with all parameters",
			url:  "http://localhost:8080?status=enabled&offset=10&limit=10&name=random&metadata={\"test\":\"test\"}&level=2&parent_id=random&tree=true&dir=-1&member_kind=random&permission=random&list_perms=true",
			header: map[string][]string{
				"Authorization": {"Bearer 123"},
			},
			resp: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{},
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
			resp: listGroupsReq{
				PageMeta: groups.PageMeta{
					Limit: 10,
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
				PageMeta: groups.PageMeta{
					Status: groups.EnabledStatus,
					Offset: 10,
					Limit:  10,
					Name:   "random",
					Metadata: groups.Metadata{
						"test": "test",
					},
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
				Limit: 10,
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

func TestDecodeChangeGroupStatusRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeChangeGroupStatusRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeChangeGroupStatusRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeChangeGroupStatusRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeRetrieveGroupHierarchy(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRetrieveGroupHierarchy(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeRetrieveGroupHierarchy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeRetrieveGroupHierarchy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeAddParentGroupRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAddParentGroupRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeAddParentGroupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeAddParentGroupRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeRemoveParentGroupRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRemoveParentGroupRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeRemoveParentGroupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeRemoveParentGroupRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeAddChildrenGroupsRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAddChildrenGroupsRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeAddChildrenGroupsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeAddChildrenGroupsRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeRemoveChildrenGroupsRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRemoveChildrenGroupsRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeRemoveChildrenGroupsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeRemoveChildrenGroupsRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeRemoveAllChildrenGroupsRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRemoveAllChildrenGroupsRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeRemoveAllChildrenGroupsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeRemoveAllChildrenGroupsRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeListChildrenGroupsRequest(t *testing.T) {
	type args struct {
		in0 context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeListChildrenGroupsRequest(tt.args.in0, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeListChildrenGroupsRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeListChildrenGroupsRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeHierarchyPageMeta(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    groups.HierarchyPageMeta
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHierarchyPageMeta(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeHierarchyPageMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeHierarchyPageMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodePageMeta(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    groups.PageMeta
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodePageMeta(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodePageMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodePageMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}
