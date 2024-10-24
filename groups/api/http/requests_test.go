// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

var valid = "valid"

func TestCreateGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  createGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: createGroupReq{
				Group: groups.Group{
					Name: valid,
				},
			},
			err: nil,
		},
		{
			desc: "long name",
			req: createGroupReq{
				Group: groups.Group{
					Name: strings.Repeat("a", api.MaxNameSize+1),
				},
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "empty name",
			req: createGroupReq{
				Group: groups.Group{},
			},
			err: apiutil.ErrNameSize,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  updateGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: updateGroupReq{
				id:   valid,
				Name: valid,
			},
			err: nil,
		},
		{
			desc: "long name",
			req: updateGroupReq{
				id:   valid,
				Name: strings.Repeat("a", api.MaxNameSize+1),
			},
			err: apiutil.ErrNameSize,
		},
		{
			desc: "empty id",
			req: updateGroupReq{
				Name: valid,
			},
			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  listGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: listGroupsReq{
				PageMeta: groups.PageMeta{
					Limit: 10,
				},
			},
			err: nil,
		},
		{
			desc: "invalid upper level",
			req: listGroupsReq{
				PageMeta: groups.PageMeta{
					Limit: 10,
				},
			},
			err: apiutil.ErrInvalidLevel,
		},
		{
			desc: "invalid lower limit",
			req: listGroupsReq{
				PageMeta: groups.PageMeta{
					Limit: 0,
				},
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "invalid upper limit",
			req: listGroupsReq{
				PageMeta: groups.PageMeta{
					Limit: api.MaxLimitSize + 1,
				},
			},
			err: apiutil.ErrLimitSize,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  groupReq
		err  error
	}{
		{
			desc: "valid request",
			req: groupReq{
				id: valid,
			},
			err: nil,
		},

		{
			desc: "empty id",
			req:  groupReq{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeGroupStatusReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  changeGroupStatusReq
		err  error
	}{
		{
			desc: "valid request",
			req: changeGroupStatusReq{
				id: valid,
			},
			err: nil,
		},
		{
			desc: "empty id",
			req:  changeGroupStatusReq{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func Test_createGroupReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     createGroupReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("createGroupReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updateGroupReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     updateGroupReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("updateGroupReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listGroupsReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     listGroupsReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("listGroupsReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_groupReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     groupReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("groupReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_changeGroupStatusReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     changeGroupStatusReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("changeGroupStatusReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_retrieveGroupHierarchyReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     retrieveGroupHierarchyReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("retrieveGroupHierarchyReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_addParentGroupReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     addParentGroupReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("addParentGroupReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_removeParentGroupReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     removeParentGroupReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("removeParentGroupReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_addChildrenGroupsReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     addChildrenGroupsReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("addChildrenGroupsReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_removeChildrenGroupsReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     removeChildrenGroupsReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("removeChildrenGroupsReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_removeAllChildrenGroupsReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     removeAllChildrenGroupsReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("removeAllChildrenGroupsReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listChildrenGroupsReq_validate(t *testing.T) {
	tests := []struct {
		name    string
		req     listChildrenGroupsReq
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.validate(); (err != nil) != tt.wantErr {
				t.Errorf("listChildrenGroupsReq.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
