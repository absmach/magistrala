// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	pauth "github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validGroupResp = groups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        valid,
		Description: valid,
		Domain:      testsutil.GenerateUUID(&testing.T{}),
		Parent:      testsutil.GenerateUUID(&testing.T{}),
		Metadata: clients.Metadata{
			"name": "test",
		},
		Children:  []*groups.Group{},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(&testing.T{}),
		Status:    clients.EnabledStatus,
	}
	validID = testsutil.GenerateUUID(&testing.T{})
)

func TestCreateGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		kind    string
		session interface{}
		req     createGroupReq
		svcResp groups.Group
		svcErr  error
		resp    createGroupRes
		err     error
	}{
		{
			desc:    "successfully with groups kind",
			kind:    policies.NewGroupKind,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: createGroupReq{
				token: valid,
				Group: groups.Group{
					Name: valid,
				},
			},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    createGroupRes{created: true, Group: validGroupResp},
			err:     nil,
		},
		{
			desc:    "successfully with channels kind",
			kind:    policies.NewChannelKind,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: createGroupReq{
				token: valid,
				Group: groups.Group{
					Name: valid,
				},
			},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    createGroupRes{created: true, Group: validGroupResp},
			err:     nil,
		},
		{
			desc:    "unsuccessfully with invalid session",
			kind:    policies.NewGroupKind,
			session: nil,
			req: createGroupReq{
				Group: groups.Group{
					Name: valid,
				},
			},
			resp: createGroupRes{created: false},
			err:  svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with repo error",
			kind:    policies.NewGroupKind,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: createGroupReq{
				token: valid,
				Group: groups.Group{
					Name: valid,
				},
			},
			svcResp: groups.Group{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    createGroupRes{created: false},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("CreateGroup", ctx, tc.session, tc.kind, tc.req.Group).Return(tc.svcResp, tc.svcErr)
		resp, err := CreateGroupEndpoint(svc, tc.kind)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(createGroupRes)
		switch err {
		case nil:
			assert.Equal(t, response.Code(), http.StatusCreated)
			assert.Equal(t, response.Headers()["Location"], fmt.Sprintf("/groups/%s", response.ID))
		default:
			assert.Equal(t, response.Code(), http.StatusOK)
			assert.Empty(t, response.Headers())
		}
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestViewGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     groupReq
		session interface{}
		svcResp groups.Group
		svcErr  error
		resp    viewGroupRes
		err     error
	}{
		{
			desc:    "successfully",
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: groupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    viewGroupRes{Group: validGroupResp},
			err:     nil,
		},
		{
			desc: "unsuccessfully with invalid session",
			req: groupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    viewGroupRes{},
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with repo error",
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: groupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			svcResp: groups.Group{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    viewGroupRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("ViewGroup", ctx, tc.session, tc.req.id).Return(tc.svcResp, tc.svcErr)
		resp, err := ViewGroupEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(viewGroupRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestViewGroupPermsEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     groupPermsReq
		session interface{}
		svcResp []string
		svcErr  error
		resp    viewGroupPermsRes
		err     error
	}{
		{
			desc: "successfully",
			req: groupPermsReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: []string{
				valid,
			},
			svcErr: nil,
			resp:   viewGroupPermsRes{Permissions: []string{valid}},
			err:    nil,
		},
		{
			desc: "unsuccessfully with invalid session",
			req: groupPermsReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			resp: viewGroupPermsRes{},
			err:  svcerr.ErrAuthorization,
		},
		{
			desc: "unsuccessfully with invalid request",
			req: groupPermsReq{
				id: testsutil.GenerateUUID(t),
			},
			resp: viewGroupPermsRes{},
			err:  apiutil.ErrValidation,
		},
		{
			desc:    "unsuccessfully with repo error",
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req: groupPermsReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			svcResp: []string{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    viewGroupPermsRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("ViewGroupPerms", ctx, tc.session, tc.req.id).Return(tc.svcResp, tc.svcErr)
		resp, err := ViewGroupPermsEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(viewGroupPermsRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestEnableGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     changeGroupStatusReq
		session interface{}
		svcResp groups.Group
		svcErr  error
		resp    changeStatusRes
		err     error
	}{
		{
			desc: "successfully",
			req: changeGroupStatusReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    changeStatusRes{Group: validGroupResp},
			err:     nil,
		},
		{
			desc: "unsuccessfully with invalid session",
			req: changeGroupStatusReq{
				id: testsutil.GenerateUUID(t),
			},
			resp: changeStatusRes{},
			err:  svcerr.ErrAuthorization,
		},
		{
			desc: "unsuccessfully with repo error",
			req: changeGroupStatusReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Group{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    changeStatusRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("EnableGroup", ctx, tc.session, tc.req.id).Return(tc.svcResp, tc.svcErr)
		resp, err := EnableGroupEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(changeStatusRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestDisableGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     changeGroupStatusReq
		session interface{}
		svcResp groups.Group
		svcErr  error
		resp    changeStatusRes
		err     error
	}{
		{
			desc: "successfully",
			req: changeGroupStatusReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    changeStatusRes{Group: validGroupResp},
			err:     nil,
		},
		{
			desc: "unsuccessfully with invalid session",
			req: changeGroupStatusReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			resp: changeStatusRes{},
			err:  svcerr.ErrAuthorization,
		},
		{
			desc: "unsuccessfully with repo error",
			req: changeGroupStatusReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Group{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    changeStatusRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("DisableGroup", ctx, tc.session, tc.req.id).Return(tc.svcResp, tc.svcErr)
		resp, err := DisableGroupEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(changeStatusRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestDeleteGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     groupReq
		session interface{}
		svcErr  error
		resp    deleteGroupRes
		err     error
	}{
		{
			desc: "successfully",
			req: groupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    deleteGroupRes{deleted: true},
			err:     nil,
		},
		{
			desc: "unsuccessfully with repo error",
			req: groupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  svcerr.ErrAuthorization,
			resp:    deleteGroupRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		svcCall := svc.On("DeleteGroup", ctx, tc.session, tc.req.id).Return(tc.svcErr)
		resp, err := DeleteGroupEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(deleteGroupRes)
		switch err {
		case nil:
			assert.Equal(t, response.Code(), http.StatusNoContent)
		default:
			assert.Equal(t, response.Code(), http.StatusBadRequest)
		}
		assert.Empty(t, response.Headers())
		assert.True(t, response.Empty())
		svcCall.Unset()
	}
}

func TestUpdateGroupEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc    string
		req     updateGroupReq
		session interface{}
		svcResp groups.Group
		svcErr  error
		resp    updateGroupRes
		err     error
	}{
		{
			desc: "successfully",
			req: updateGroupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
				Name:  valid,
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: validGroupResp,
			svcErr:  nil,
			resp:    updateGroupRes{Group: validGroupResp},
			err:     nil,
		},
		{
			desc: "unsuccessfully with invalid session",
			req: updateGroupReq{
				id:   testsutil.GenerateUUID(t),
				Name: valid,
			},
			resp: updateGroupRes{},
			err:  svcerr.ErrAuthorization,
		},
		{
			desc: "unsuccessfully with repo error",
			req: updateGroupReq{
				token: valid,
				id:    testsutil.GenerateUUID(t),
				Name:  valid,
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Group{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    updateGroupRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		group := groups.Group{
			ID:          tc.req.id,
			Name:        tc.req.Name,
			Description: tc.req.Description,
			Metadata:    tc.req.Metadata,
		}
		svcCall := svc.On("UpdateGroup", ctx, tc.session, group).Return(tc.svcResp, tc.svcErr)
		resp, err := UpdateGroupEndpoint(svc)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(updateGroupRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestListGroupsEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	childGroup := groups.Group{
		ID:          testsutil.GenerateUUID(t),
		Name:        valid,
		Description: valid,
		Domain:      testsutil.GenerateUUID(t),
		Parent:      validGroupResp.ID,
		Metadata: clients.Metadata{
			"name": "test",
		},
		Level:     -1,
		Children:  []*groups.Group{},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(t),
		Status:    clients.EnabledStatus,
	}
	parentGroup := groups.Group{
		ID:          testsutil.GenerateUUID(t),
		Name:        valid,
		Description: valid,
		Domain:      testsutil.GenerateUUID(t),
		Metadata: clients.Metadata{
			"name": "test",
		},
		Level:     1,
		Children:  []*groups.Group{},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(t),
		Status:    clients.EnabledStatus,
	}

	validGroupResp.Children = append(validGroupResp.Children, &childGroup)
	parentGroup.Children = append(parentGroup.Children, &validGroupResp)

	cases := []struct {
		desc       string
		memberKind string
		req        listGroupsReq
		session    interface{}
		svcResp    groups.Page
		svcErr     error
		resp       groupPageRes
		err        error
	}{
		{
			desc:       "successfully",
			memberKind: policies.ThingsKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
				},
				token:      valid,
				memberKind: policies.ThingsKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{
				Groups: []groups.Group{validGroupResp},
			},
			svcErr: nil,
			resp: groupPageRes{
				Groups: []viewGroupRes{
					{
						Group: validGroupResp,
					},
				},
			},
			err: nil,
		},
		{
			desc: "successfully with empty member kind",
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
				},
				token:      valid,
				memberKind: policies.ThingsKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{
				Groups: []groups.Group{validGroupResp},
			},
			svcErr: nil,
			resp: groupPageRes{
				Groups: []viewGroupRes{
					{
						Group: validGroupResp,
					},
				},
			},
			err: nil,
		},
		{
			desc:       "successfully with tree",
			memberKind: policies.ThingsKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
				},
				tree:       true,
				token:      valid,
				memberKind: policies.ThingsKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{
				Groups: []groups.Group{validGroupResp, childGroup},
			},
			svcErr: nil,
			resp: groupPageRes{
				Groups: []viewGroupRes{
					{
						Group: validGroupResp,
					},
				},
			},
			err: nil,
		},
		{
			desc:       "list children groups successfully without tree",
			memberKind: policies.UsersKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					ParentID:  validGroupResp.ID,
					Direction: -1,
				},
				tree:       false,
				token:      valid,
				memberKind: policies.UsersKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{
				Groups: []groups.Group{validGroupResp, childGroup},
			},
			svcErr: nil,
			resp: groupPageRes{
				Groups: []viewGroupRes{
					{
						Group: childGroup,
					},
				},
			},
			err: nil,
		},
		{
			desc:       "list parent group successfully without tree",
			memberKind: policies.UsersKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
					ParentID:  validGroupResp.ID,
					Direction: 1,
				},
				tree:       false,
				token:      valid,
				memberKind: policies.UsersKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{
				Groups: []groups.Group{parentGroup, validGroupResp},
			},
			svcErr: nil,
			resp: groupPageRes{
				Groups: []viewGroupRes{
					{
						Group: parentGroup,
					},
				},
			},
			err: nil,
		},
		{
			desc:       "unsuccessfully with invalid request",
			memberKind: policies.ThingsKind,
			session:    pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			req:        listGroupsReq{},
			resp:       groupPageRes{},
			err:        apiutil.ErrValidation,
		},
		{
			desc:       "unsuccessfully with repo error",
			memberKind: policies.ThingsKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
				},
				token:      valid,
				memberKind: policies.ThingsKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.Page{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    groupPageRes{},
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with invalid session",
			memberKind: policies.ThingsKind,
			req: listGroupsReq{
				Page: groups.Page{
					PageMeta: groups.PageMeta{
						Limit: 10,
					},
				},
				token:      valid,
				memberKind: policies.ThingsKind,
				memberID:   testsutil.GenerateUUID(t),
			},
			resp: groupPageRes{},
			err:  svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		if tc.memberKind != "" {
			tc.req.memberKind = tc.memberKind
		}
		svcCall := svc.On("ListGroups", ctx, tc.session, tc.req.memberKind, tc.req.memberID, tc.req.Page).Return(tc.svcResp, tc.svcErr)
		resp, err := ListGroupsEndpoint(svc, mock.Anything, tc.memberKind)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(groupPageRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestListMembersEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc       string
		memberKind string
		req        listMembersReq
		session    interface{}
		svcResp    groups.MembersPage
		svcErr     error
		resp       listMembersRes
		err        error
	}{
		{
			desc:       "successfully",
			memberKind: policies.ThingsKind,
			req: listMembersReq{
				token:      valid,
				memberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.MembersPage{
				Members: []groups.Member{
					{
						ID:   valid,
						Type: valid,
					},
				},
			},
			svcErr: nil,
			resp: listMembersRes{
				Members: []groups.Member{
					{
						ID:   valid,
						Type: valid,
					},
				},
			},
			err: nil,
		},
		{
			desc: "successfully with empty member kind",
			req: listMembersReq{
				token:      valid,
				memberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.MembersPage{
				Members: []groups.Member{
					{
						ID:   valid,
						Type: valid,
					},
				},
			},
			svcErr: nil,
			resp: listMembersRes{
				Members: []groups.Member{
					{
						ID:   valid,
						Type: valid,
					},
				},
			},
			err: nil,
		},
		{
			desc:       "unsuccessfully with invalid request",
			memberKind: policies.ThingsKind,
			req:        listMembersReq{},
			session:    pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			resp:       listMembersRes{},
			err:        apiutil.ErrValidation,
		},
		{
			desc:       "unsuccessfully with repo error",
			memberKind: policies.ThingsKind,
			req: listMembersReq{
				token:      valid,
				memberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcResp: groups.MembersPage{},
			svcErr:  svcerr.ErrAuthorization,
			resp:    listMembersRes{},
			err:     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		if tc.memberKind != "" {
			tc.req.memberKind = tc.memberKind
		}
		svcCall := svc.On("ListMembers", ctx, tc.session, tc.req.groupID, tc.req.permission, tc.req.memberKind).Return(tc.svcResp, tc.svcErr)
		resp, err := ListMembersEndpoint(svc, tc.memberKind)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(listMembersRes)
		assert.Equal(t, response.Code(), http.StatusOK)
		assert.Empty(t, response.Headers())
		assert.False(t, response.Empty())
		svcCall.Unset()
	}
}

func TestAssignMembersEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc       string
		relation   string
		session    interface{}
		memberKind string
		req        assignReq
		svcErr     error
		resp       assignRes
		err        error
	}{
		{
			desc:       "successfully",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: assignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    assignRes{assigned: true},
			err:     nil,
		},
		{
			desc:     "successfully with empty member kind",
			relation: policies.ContributorRelation,
			req: assignReq{
				token:      valid,
				groupID:    testsutil.GenerateUUID(t),
				MemberKind: policies.ThingsKind,
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    assignRes{assigned: true},
			err:     nil,
		},
		{
			desc:       "successfully with empty relation",
			memberKind: policies.ThingsKind,
			req: assignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    assignRes{assigned: true},
			err:     nil,
		},
		{
			desc:       "unsuccessfully with invalid request",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req:        assignReq{},
			session:    pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			resp:       assignRes{},
			err:        apiutil.ErrValidation,
		},
		{
			desc:       "unsuccessfully with repo error",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: assignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  svcerr.ErrAuthorization,
			resp:    assignRes{},
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with invalid session",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: assignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			resp: assignRes{},
			err:  svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		if tc.memberKind != "" {
			tc.req.MemberKind = tc.memberKind
		}
		if tc.relation != "" {
			tc.req.Relation = tc.relation
		}
		svcCall := svc.On("Assign", ctx, tc.session, tc.req.groupID, tc.req.Relation, tc.req.MemberKind, tc.req.Members).Return(tc.svcErr)
		resp, err := AssignMembersEndpoint(svc, tc.relation, tc.memberKind)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(assignRes)
		switch err {
		case nil:
			assert.Equal(t, response.Code(), http.StatusCreated)
		default:
			assert.Equal(t, response.Code(), http.StatusBadRequest)
		}
		assert.Empty(t, response.Headers())
		assert.True(t, response.Empty())
		svcCall.Unset()
	}
}

func TestUnassignMembersEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	cases := []struct {
		desc       string
		relation   string
		memberKind string
		req        unassignReq
		session    interface{}
		svcErr     error
		resp       unassignRes
		err        error
	}{
		{
			desc:       "successfully",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: unassignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    unassignRes{unassigned: true},
			err:     nil,
		},
		{
			desc:     "successfully with empty member kind",
			relation: policies.ContributorRelation,
			req: unassignReq{
				token:      valid,
				groupID:    testsutil.GenerateUUID(t),
				MemberKind: policies.ThingsKind,
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  nil,
			resp:    unassignRes{unassigned: true},
			err:     nil,
		},
		{
			desc:       "successfully with empty relation",
			memberKind: policies.ThingsKind,
			req: unassignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			svcErr:  nil,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			resp:    unassignRes{unassigned: true},
			err:     nil,
		},
		{
			desc:       "unsuccessfully with invalid request",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req:        unassignReq{},
			session:    pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			resp:       unassignRes{},
			err:        apiutil.ErrValidation,
		},
		{
			desc:       "unsuccessfully with repo error",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: unassignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			svcErr:  svcerr.ErrAuthorization,
			resp:    unassignRes{},
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with invalid session",
			relation:   policies.ContributorRelation,
			memberKind: policies.ThingsKind,
			req: unassignReq{
				token:      valid,
				MemberKind: policies.ThingsKind,
				groupID:    testsutil.GenerateUUID(t),
				Members: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
			resp: unassignRes{},
			err:  svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		ctx := context.WithValue(context.Background(), api.SessionKey, tc.session)
		if tc.memberKind != "" {
			tc.req.MemberKind = tc.memberKind
		}
		if tc.relation != "" {
			tc.req.Relation = tc.relation
		}
		svcCall := svc.On("Unassign", ctx, tc.session, tc.req.groupID, tc.req.Relation, tc.req.MemberKind, tc.req.Members).Return(tc.svcErr)
		resp, err := UnassignMembersEndpoint(svc, tc.relation, tc.memberKind)(ctx, tc.req)
		assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
		response := resp.(unassignRes)
		switch err {
		case nil:
			assert.Equal(t, response.Code(), http.StatusCreated)
		default:
			assert.Equal(t, response.Code(), http.StatusBadRequest)
		}
		assert.Empty(t, response.Headers())
		assert.True(t, response.Empty())
		svcCall.Unset()
	}
}
