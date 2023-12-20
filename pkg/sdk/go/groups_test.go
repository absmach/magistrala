// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/groups/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/api"
	umocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupGroups() (*httptest.Server, *mocks.Repository, *authmocks.Service) {
	crepo := new(umocks.Repository)
	grepo := new(mocks.Repository)

	auth := new(authmocks.Service)
	csvc := users.NewService(crepo, auth, emailer, phasher, idProvider, passRegex, true)
	gsvc := groups.NewService(grepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(csvc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), grepo, auth
}

func TestCreateGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()
	group := sdk.Group{
		Name:     "groupName",
		Metadata: validMetadata,
		Status:   clients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	cases := []struct {
		desc  string
		group sdk.Group
		token string
		err   errors.SDKError
	}{
		{
			desc:  "create group successfully",
			group: group,
			token: token,
			err:   nil,
		},
		{
			desc:  "create group with existing name",
			group: group,
			token: token,
			err:   nil,
		},
		{
			desc:  "create group with parent",
			token: token,
			group: sdk.Group{
				Name:     gName,
				ParentID: testsutil.GenerateUUID(t),
				Status:   clients.EnabledStatus.String(),
			},
			err: nil,
		},
		{
			desc:  "create group with invalid parent",
			token: token,
			group: sdk.Group{
				Name:     gName,
				ParentID: mocks.WrongID,
				Status:   clients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusInternalServerError),
		},
		{
			desc:  "create group with invalid owner",
			token: token,
			group: sdk.Group{
				Name:    gName,
				OwnerID: mocks.WrongID,
				Status:  clients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(sdk.ErrFailedCreation, http.StatusInternalServerError),
		},
		{
			desc:  "create group with missing name",
			token: token,
			group: sdk.Group{
				Status: clients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "create a group with every field defined",
			token: token,
			group: sdk.Group{
				ID:          generateUUID(t),
				OwnerID:     "owner",
				ParentID:    "parent",
				Name:        "name",
				Description: description,
				Metadata:    validMetadata,
				Level:       1,
				Children:    []*sdk.Group{&group},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      clients.EnabledStatus.String(),
			},
			err: nil,
		},
		{
			desc:  "create a group that can't be marshalled",
			token: token,
			group: sdk.Group{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			err: errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Authorized: true}, nil)
		repoCall2 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall3 := grepo.On("Save", mock.Anything, mock.Anything).Return(convertGroup(sdk.Group{}), tc.err)
		rGroup, err := mgsdk.CreateGroup(tc.group, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if err == nil {
			assert.NotEmpty(t, rGroup, fmt.Sprintf("%s: expected not nil on client ID", tc.desc))
			ok := repoCall3.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListGroups(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	var grps []sdk.Group
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		gr := sdk.Group{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("group_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
			Status:   clients.EnabledStatus.String(),
		}
		grps = append(grps, gr)
	}

	cases := []struct {
		desc     string
		token    string
		status   clients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		ownerID  string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Group
	}{
		{
			desc:     "get a list of groups",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: grps[offset:limit],
		},
		{
			desc:     "get a list of groups with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      nil,
			response: nil,
		},
		{
			desc:     "get a list of groups with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Group(nil),
		},
		{
			desc:     "get a list of groups with given name",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
		{
			desc:     "get a list of groups with level",
			token:    token,
			offset:   0,
			limit:    1,
			level:    1,
			err:      nil,
			response: []sdk.Group{grps[0]},
		},
		{
			desc:     "get a list of groups with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertGroups(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
		}
		page, err := mgsdk.Groups(pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListParentGroups(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	var grps []sdk.Group
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	parentID := ""
	for i := 10; i < 100; i++ {
		gr := sdk.Group{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("group_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
			Status:   clients.EnabledStatus.String(),
			ParentID: parentID,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}

	cases := []struct {
		desc     string
		token    string
		status   clients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		ownerID  string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Group
	}{
		{
			desc:     "get a list of groups",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: grps[offset:limit],
		},
		{
			desc:     "get a list of groups with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      nil,
			response: nil,
		},
		{
			desc:     "get a list of groups with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Group(nil),
		},
		{
			desc:     "get a list of groups with given name",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
		{
			desc:     "get a list of groups with level",
			token:    token,
			offset:   0,
			limit:    1,
			level:    1,
			err:      nil,
			response: []sdk.Group{grps[0]},
		},
		{
			desc:     "get a list of groups with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertGroups(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
		}
		page, err := mgsdk.Parents(parentID, pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListChildrenGroups(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	var grps []sdk.Group
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	parentID := ""
	for i := 10; i < 100; i++ {
		gr := sdk.Group{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("group_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
			Status:   clients.EnabledStatus.String(),
			ParentID: parentID,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}
	childID := grps[0].ID

	cases := []struct {
		desc     string
		token    string
		status   clients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		ownerID  string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Group
	}{
		{
			desc:     "get a list of groups",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: grps[offset:limit],
		},
		{
			desc:     "get a list of groups with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      nil,
			response: nil,
		},
		{
			desc:     "get a list of groups with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Group(nil),
		},
		{
			desc:     "get a list of groups with given name",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
		{
			desc:     "get a list of groups with level",
			token:    token,
			offset:   0,
			limit:    1,
			level:    1,
			err:      nil,
			response: []sdk.Group{grps[0]},
		},
		{
			desc:     "get a list of groups with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Group{grps[89]},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertGroups(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
		}
		page, err := mgsdk.Children(childID, pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestViewGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	group := sdk.Group{
		Name:        "groupName",
		Description: description,
		Metadata:    validMetadata,
		Children:    []*sdk.Group{},
		Status:      clients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	group.ID = generateUUID(t)

	cases := []struct {
		desc     string
		token    string
		groupID  string
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:     "view group",
			token:    validToken,
			groupID:  group.ID,
			response: group,
			err:      nil,
		},
		{
			desc:     "view group with invalid token",
			token:    "wrongtoken",
			groupID:  group.ID,
			response: sdk.Group{Children: []*sdk.Group{}},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view group for wrong id",
			token:    validToken,
			groupID:  mocks.WrongID,
			response: sdk.Group{Children: []*sdk.Group{}},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall1 := grepo.On("RetrieveByID", mock.Anything, tc.groupID).Return(convertGroup(tc.response), tc.err)
		grp, err := mgsdk.Group(tc.groupID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if len(tc.response.Children) == 0 {
			tc.response.Children = nil
		}
		if len(grp.Children) == 0 {
			grp.Children = nil
		}
		assert.Equal(t, tc.response, grp, fmt.Sprintf("%s: expected metadata %v got %v\n", tc.desc, tc.response, grp))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.groupID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	group := sdk.Group{
		ID:          generateUUID(t),
		Name:        "groupName",
		Description: description,
		Metadata:    validMetadata,
	}

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group.ID = generateUUID(t)

	cases := []struct {
		desc     string
		token    string
		group    sdk.Group
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc: "update group name",
			group: sdk.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			response: sdk.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update group description",
			group: sdk.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			response: sdk.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update group metadata",
			group: sdk.Group{
				ID: group.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Group{
				ID: group.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update group name with invalid group id",
			group: sdk.Group{
				ID:   mocks.WrongID,
				Name: "NewName",
			},
			response: sdk.Group{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update group description with invalid group id",
			group: sdk.Group{
				ID:          mocks.WrongID,
				Description: "NewDescription",
			},
			response: sdk.Group{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update group metadata with invalid group id",
			group: sdk.Group{
				ID: mocks.WrongID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Group{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update group name with invalid token",
			group: sdk.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			response: sdk.Group{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update group description with invalid token",
			group: sdk.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			response: sdk.Group{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update group metadata with invalid token",
			group: sdk.Group{
				ID: group.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Group{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update a group that can't be marshalled",
			group: sdk.Group{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Group{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall1 := grepo.On("Update", mock.Anything, mock.Anything).Return(convertGroup(tc.response), tc.err)
		_, err := mgsdk.UpdateGroup(tc.group, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	group := sdk.Group{
		ID:        generateUUID(t),
		Name:      gName,
		OwnerID:   generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    clients.Disabled,
	}

	repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 := grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
	repoCall2 := grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(nil)
	_, err := mgsdk.EnableGroup("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound), fmt.Sprintf("Enable group with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "RetrieveByID was not called on enabling group")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	g := mggroups.Group{
		ID:        group.ID,
		Name:      group.Name,
		Owner:     group.OwnerID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    clients.DisabledStatus,
	}
	repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 = grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(g, nil)
	repoCall2 = grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(g, nil)
	res, err := mgsdk.EnableGroup(group.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Enable group with correct id: expected %v got %v", nil, err))
	assert.Equal(t, group, res, fmt.Sprintf("Enable group with correct id: expected %v got %v", group, res))
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, group.ID)
	assert.True(t, ok, "RetrieveByID was not called on enabling group")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on enabling group")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestDisableGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	group := sdk.Group{
		ID:        generateUUID(t),
		Name:      gName,
		OwnerID:   generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    clients.Enabled,
	}

	repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 := grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
	repoCall2 := grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
	_, err := mgsdk.DisableGroup("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound), fmt.Sprintf("Disable group with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "Memberships was not called on disabling group with wrong id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	g := mggroups.Group{
		ID:        group.ID,
		Name:      group.Name,
		Owner:     group.OwnerID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    clients.EnabledStatus,
	}

	repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 = grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(g, nil)
	repoCall2 = grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(g, nil)
	res, err := mgsdk.DisableGroup(group.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Disable group with correct id: expected %v got %v", nil, err))
	assert.Equal(t, group, res, fmt.Sprintf("Disable group with correct id: expected %v got %v", group, res))
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, group.ID)
	assert.True(t, ok, "RetrieveByID was not called on disabling group with correct id")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on disabling group with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestDeleteGroup(t *testing.T) {
	ts, grepo, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	group := sdk.Group{
		ID:        generateUUID(t),
		Name:      gName,
		OwnerID:   generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    clients.Enabled,
	}

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, nil)
	repoCall2 := grepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	err := mgsdk.DeleteGroup("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden), fmt.Sprintf("Delete group with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	repoCall = auth.On("DeletePolicy", mock.Anything, mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
	repoCall1 = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall2 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall3 := grepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	err = mgsdk.DeleteGroup(group.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Delete group with correct id: expected %v got %v", nil, err))
	ok := repoCall3.Parent.AssertCalled(t, "Delete", mock.Anything, group.ID)
	assert.True(t, ok, "Delete was not called on deleting group with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
	repoCall3.Unset()
}
