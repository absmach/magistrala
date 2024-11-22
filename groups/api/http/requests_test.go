// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
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

func TestRetrieveGroupHierarchyReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  retrieveGroupHierarchyReq
		err  error
	}{
		{
			desc: "valid request",
			req: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Tree:      true,
					Level:     1,
					Direction: -1,
				},
				id: valid,
			},
		},
		{
			desc: "invalid level",
			req: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Tree:      true,
					Level:     groups.MaxLevel + 1,
					Direction: -1,
				},
				id: valid,
			},
			err: apiutil.ErrLevel,
		},
		{
			desc: "empty id",
			req: retrieveGroupHierarchyReq{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Tree:      true,
					Level:     1,
					Direction: -1,
				},
			},
			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAddParentGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  addParentGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: addParentGroupReq{
				id:       testsutil.GenerateUUID(t),
				ParentID: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: addParentGroupReq{
				ParentID: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty parent id",
			req: addParentGroupReq{
				id: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "invalid parent id",
			req: addParentGroupReq{
				id:       testsutil.GenerateUUID(t),
				ParentID: "invalid",
			},
			err: apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "same id",
			req: addParentGroupReq{
				id:       validID,
				ParentID: validID,
			},
			err: apiutil.ErrSelfParentingNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveParentGroupReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  removeParentGroupReq
		err  error
	}{
		{
			desc: "valid request",
			req: removeParentGroupReq{
				id: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "empty id",
			req:  removeParentGroupReq{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestAddChildrenGroupsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  addChildrenGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: addChildrenGroupsReq{
				id:          testsutil.GenerateUUID(t),
				ChildrenIDs: []string{testsutil.GenerateUUID(t)},
			},
			err: nil,
		},
		{
			desc: "empty id",
			req: addChildrenGroupsReq{
				ChildrenIDs: []string{testsutil.GenerateUUID(t)},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty children ids",
			req: addChildrenGroupsReq{
				id: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingChildrenGroupIDs,
		},
		{
			desc: "invalid child id",
			req: addChildrenGroupsReq{
				id:          testsutil.GenerateUUID(t),
				ChildrenIDs: []string{"invalid"},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "self parenting",
			req: addChildrenGroupsReq{
				id:          validID,
				ChildrenIDs: []string{validID, testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			err: apiutil.ErrSelfParentingNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveChildrenGroupsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  removeChildrenGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: removeChildrenGroupsReq{
				id:          testsutil.GenerateUUID(t),
				ChildrenIDs: []string{testsutil.GenerateUUID(t)},
			},
			err: nil,
		},
		{
			desc: "empty id",
			req:  removeChildrenGroupsReq{},
			err:  apiutil.ErrMissingID,
		},
		{
			desc: "empty children ids",
			req: removeChildrenGroupsReq{
				id: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingChildrenGroupIDs,
		},
		{
			desc: "invalid child id",
			req: removeChildrenGroupsReq{
				id:          testsutil.GenerateUUID(t),
				ChildrenIDs: []string{"invalid"},
			},
			err: apiutil.ErrInvalidIDFormat,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveAllChildrenGroupsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  removeAllChildrenGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: removeAllChildrenGroupsReq{
				id: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "empty id",
			req:  removeAllChildrenGroupsReq{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestListChildrenGroupsReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  listChildrenGroupsReq
		err  error
	}{
		{
			desc: "valid request",
			req: listChildrenGroupsReq{
				id: validID,
				PageMeta: groups.PageMeta{
					Limit: 10,
				},
			},
			err: nil,
		},
		{
			desc: "empty id",
			req:  listChildrenGroupsReq{},
			err:  apiutil.ErrMissingID,
		},
		{
			desc: "invalid lower limit",
			req: listChildrenGroupsReq{
				id: validID,
				PageMeta: groups.PageMeta{
					Limit: 0,
				},
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "invalid upper limit",
			req: listChildrenGroupsReq{
				id: validID,
				PageMeta: groups.PageMeta{
					Limit: api.MaxLimitSize + 1,
				},
			},
			err: apiutil.ErrLimitSize,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}
