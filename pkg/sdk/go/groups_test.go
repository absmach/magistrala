// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/users/api"
	umocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	sdkGroup           = generateTestGroup(&testing.T{})
	group              = convertGroup(sdkGroup)
	updatedName        = "updated_name"
	updatedDescription = "updated_description"
)

func setupGroups() (*httptest.Server, *mocks.Service) {
	usvc := new(umocks.Service)
	gsvc := new(mocks.Service)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	api.MakeHandler(usvc, gsvc, mux, logger, "", passRegex, provider)

	return httptest.NewServer(mux), gsvc
}

func TestCreateGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	createGroupReq := sdk.Group{
		Name:        gName,
		Description: description,
		Metadata:    validMetadata,
	}
	pGroup := group
	pGroup.Parent = testsutil.GenerateUUID(t)
	psdkGroup := sdkGroup
	psdkGroup.ParentID = pGroup.Parent

	uGroup := group
	uGroup.Metadata = mgclients.Metadata{
		"key": make(chan int),
	}
	cases := []struct {
		desc     string
		token    string
		groupReq sdk.Group
		svcReq   groups.Group
		svcRes   groups.Group
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:     "create group successfully",
			token:    validToken,
			groupReq: createGroupReq,
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
			},
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:     "create group with existing name",
			token:    validToken,
			groupReq: createGroupReq,
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
			},
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:  "create group with parent",
			token: validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
				ParentID:    pGroup.Parent,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
				Parent:      pGroup.Parent,
			},
			svcRes:   pGroup,
			svcErr:   nil,
			response: psdkGroup,
			err:      nil,
		},
		{
			desc:  "create group with invalid parent",
			token: validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
				ParentID:    wrongID,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
				Parent:      wrongID,
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:  "create group with invalid token",
			token: invalidToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "create group with empty token",
			token: "",
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "create group with missing name",
			token: validToken,
			groupReq: sdk.Group{
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "create group with name that is too long",
			token: validToken,
			groupReq: sdk.Group{
				Name:        strings.Repeat("a", 1025),
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "create group with request that cannot be marshalled",
			token: validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "create group with service response that cannot be unmarshalled",
			token: validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    mgclients.Metadata{"role": "client"},
			},
			svcRes:   uGroup,
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("CreateGroup", mock.Anything, tc.token, auth.NewGroupKind, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateGroup(tc.groupReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateGroup", mock.Anything, tc.token, auth.NewGroupKind, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	ts, gsvc := setupGroups()
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
			Status:   mgclients.EnabledStatus.String(),
		}
		grps = append(grps, gr)
	}

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   groups.Page
		svcRes   groups.Page
		svcErr   error
		response sdk.GroupsPage
		err      errors.SDKError
	}{
		{
			desc:  "list groups successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  100,
				},
				Permission: auth.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps)),
				},
				Groups: convertGroups(grps),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps)),
				},
				Groups: grps,
			},
			err: nil,
		},
		{
			desc:  "list groups with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  100,
				},
				Permission: auth.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list groups with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list groups with zero limit",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  10,
				},
				Permission: auth.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps[0:10])),
				},
				Groups: convertGroups(grps[0:10]),
			},
			svcErr: nil,
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps[0:10])),
				},
				Groups: grps[0:10],
			},
			err: nil,
		},
		{
			desc:  "list groups with limit greater than max",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  110,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:  "list groups with given name",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: sdk.Metadata{
					"name": "user_89",
				},
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
					Metadata: mgclients.Metadata{
						"name": "user_89",
					},
				},
				Permission: auth.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: convertGroups([]sdk.Group{grps[89]}),
			},
			svcErr: nil,
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Groups: []sdk.Group{grps[89]},
			},
			err: nil,
		},
		{
			desc:  "list groups with invalid level",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
				Level:  6,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidLevel), http.StatusInternalServerError),
		},
		{
			desc:  "list groups with invalid page metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list groups with service response that cannot be unmarshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				Permission: auth.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{{
					ID:   generateUUID(t),
					Name: "group_1",
					Metadata: mgclients.Metadata{
						"key": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Groups(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListGroupsByUser(t *testing.T) {
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
		userID   string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Group
	}{
		{
			desc:     "get a list of groups with valid user id",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			userID:   validID,
			err:      nil,
			response: grps[offset:limit],
		},
		{
			desc:     "get a list of groups with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of groups with invalid user id",
			token:    token,
			offset:   offset,
			limit:    0,
			userID:   "invalid",
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			response: nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertGroups(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
			User:   tc.userID,
		}
		page, err := mgsdk.ListUserGroups(pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListParentGroups(t *testing.T) {
	ts, gsvc := setupGroups()
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
			Status:   mgclients.EnabledStatus.String(),
			ParentID: parentID,
			Level:    1,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		parentID string
		svcReq   groups.Page
		svcRes   groups.Page
		svcErr   error
		response sdk.GroupsPage
		err      errors.SDKError
	}{
		{
			desc:     "list parent groups successfully",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         parentID,
				Permission: auth.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps[offset:limit])),
				},
				Groups: convertGroups(grps[offset:limit]),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps[offset:limit])),
				},
				Groups: grps[offset:limit],
			},
			err: nil,
		},
		{
			desc:     "list parent groups with invalid token",
			token:    invalidToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         parentID,
				Permission: auth.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes:   groups.Page{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list parent groups with empty token",
			token:    "",
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "list parent groups with zero limit",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  10,
				},
				ID:         parentID,
				Permission: auth.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps[offset:10])),
				},
				Groups: convertGroups(grps[offset:10]),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps[offset:10])),
				},
				Groups: grps[offset:10],
			},
			err: nil,
		},
		{
			desc:     "list parent groups with limit greater than max",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  110,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:     "list parent groups with given metadata",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"name": "user_89",
				},
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
					Metadata: mgclients.Metadata{
						"name": "user_89",
					},
				},
				ID:         parentID,
				Permission: auth.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: convertGroups([]sdk.Group{grps[89]}),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Groups: []sdk.Group{grps[89]},
			},
			err: nil,
		},
		{
			desc:     "list parent groups with invalid page metadata",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "list parent groups with service response that cannot be unmarshalled",
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         parentID,
				Permission: auth.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{{
					ID:   generateUUID(t),
					Name: "group_1",
					Metadata: mgclients.Metadata{
						"key": make(chan int),
					},
					Level: 1,
				}},
			},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Parents(tc.parentID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListGroupsByChannel(t *testing.T) {
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
		desc      string
		token     string
		status    clients.Status
		total     uint64
		offset    uint64
		limit     uint64
		level     int
		name      string
		channelID string
		metadata  sdk.Metadata
		err       errors.SDKError
		response  []sdk.Group
	}{
		{
			desc:      "get a list of groups with valid user id",
			token:     token,
			limit:     limit,
			offset:    offset,
			total:     total,
			channelID: validID,
			err:       nil,
			response:  grps[offset:limit],
		},
		{
			desc:      "get a list of groups with invalid token",
			token:     invalidToken,
			offset:    offset,
			limit:     limit,
			channelID: validID,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response:  nil,
		},
		{
			desc:      "get a list of groups with empty token",
			token:     "",
			offset:    offset,
			limit:     limit,
			channelID: validID,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response:  nil,
		},
		{
			desc:      "get a list of groups with invalid user id",
			token:     token,
			channelID: invalidIdentity,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			response:  nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllSubjects", mock.Anything, mock.Anything).Return(&magistrala.ListSubjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall4 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertGroups(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset:  tc.offset,
			Limit:   tc.limit,
			Level:   uint64(tc.level),
			Channel: tc.channelID,
		}
		page, err := mgsdk.ListChannelUserGroups(pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall4.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestListChildrenGroups(t *testing.T) {
	ts, gsvc := setupGroups()
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
			Status:   mgclients.EnabledStatus.String(),
			ParentID: parentID,
			Level:    -1,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}
	childID := grps[0].ID

	cases := []struct {
		desc     string
		token    string
		childID  string
		pageMeta sdk.PageMetadata
		svcReq   groups.Page
		svcRes   groups.Page
		svcErr   error
		response sdk.GroupsPage
		err      errors.SDKError
	}{
		{
			desc:    "list children groups successfully",
			token:   validToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         childID,
				Permission: auth.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps[offset:limit])),
				},
				Groups: convertGroups(grps[offset:limit]),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps[offset:limit])),
				},
				Groups: grps[offset:limit],
			},
			err: nil,
		},
		{
			desc:    "list children groups with invalid token",
			token:   invalidToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         childID,
				Permission: auth.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes:   groups.Page{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "list children groups with empty token",
			token:   "",
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:    "list children groups with zero limit",
			token:   validToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  10,
				},
				ID:         childID,
				Permission: auth.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: uint64(len(grps[offset:10])),
				},
				Groups: convertGroups(grps[offset:10]),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(grps[offset:10])),
				},
				Groups: grps[offset:10],
			},
			err: nil,
		},
		{
			desc:  "list children groups with limit greater than max",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  110,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:    "list children groups with given metadata",
			token:   validToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"name": "user_89",
				},
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
					Metadata: mgclients.Metadata{
						"name": "user_89",
					},
				},
				ID:         childID,
				Permission: auth.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: convertGroups([]sdk.Group{grps[89]}),
			},
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Groups: []sdk.Group{grps[89]},
			},
			err: nil,
		},
		{
			desc:    "list children groups with invalid page metadata",
			token:   validToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:    "list children groups with service response that cannot be unmarshalled",
			token:   validToken,
			childID: childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ID:         childID,
				Permission: auth.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{{
					ID:   generateUUID(t),
					Name: "group_1",
					Metadata: mgclients.Metadata{
						"key": make(chan int),
					},
					Level: -1,
				}},
			},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Children(tc.childID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.token, auth.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		groupID  string
		svcRes   groups.Group
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:     "view group successfully",
			token:    validToken,
			groupID:  group.ID,
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:     "view group with invalid token",
			token:    invalidToken,
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view group with empty token",
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view group with invalid group id",
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:    "view group with service response that cannot be unmarshalled",
			token:   validToken,
			groupID: group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
		{
			desc:     "view group with empty id",
			token:    validToken,
			groupID:  "",
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("ViewGroup", mock.Anything, tc.token, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Group(tc.groupID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroup", mock.Anything, tc.token, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewGroupPermissions(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		groupID  string
		svcRes   []string
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:    "view group permissions successfully",
			token:   validToken,
			groupID: group.ID,
			svcRes:  []string{auth.ViewPermission, auth.MembershipPermission},
			svcErr:  nil,
			response: sdk.Group{
				Permissions: []string{auth.ViewPermission, auth.MembershipPermission},
			},
			err: nil,
		},
		{
			desc:     "view group permissions with invalid token",
			token:    invalidToken,
			groupID:  group.ID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view group permissions with empty token",
			token:    "",
			groupID:  group.ID,
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view group permissions with invalid group id",
			token:    validToken,
			groupID:  wrongID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view group permissions with empty id",
			token:    validToken,
			groupID:  "",
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("ViewGroupPerms", mock.Anything, tc.token, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupPermissions(tc.groupID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroupPerms", mock.Anything, tc.token, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	upGroup := sdkGroup
	upGroup.Name = updatedName
	upGroup.Description = updatedDescription
	upGroup.Metadata = sdk.Metadata{"key": "value"}

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group.ID = generateUUID(t)

	cases := []struct {
		desc     string
		token    string
		groupReq sdk.Group
		svcReq   groups.Group
		svcRes   groups.Group
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:  "update group successfully",
			token: validToken,
			groupReq: sdk.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq: groups.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    mgclients.Metadata{"key": "value"},
			},
			svcRes:   convertGroup(upGroup),
			svcErr:   nil,
			response: upGroup,
			err:      nil,
		},
		{
			desc:  "update group name with invalid group id",
			token: validToken,
			groupReq: sdk.Group{
				ID:          wrongID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq: groups.Group{
				ID:          wrongID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    mgclients.Metadata{"key": "value"},
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:  "update group name with invalid token",
			token: invalidToken,
			groupReq: sdk.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq: groups.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    mgclients.Metadata{"key": "value"},
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update group name with empty token",
			token: "",
			groupReq: sdk.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update group with empty id",
			token: validToken,
			groupReq: sdk.Group{
				ID:          "",
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:  "update group with request that can't be marshalled",
			token: validToken,
			groupReq: sdk.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": make(chan int)},
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update group with service response that cannot be unmarshalled",
			token: validToken,
			groupReq: sdk.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    sdk.Metadata{"key": "value"},
			},
			svcReq: groups.Group{
				ID:          group.ID,
				Name:        updatedName,
				Description: updatedDescription,
				Metadata:    mgclients.Metadata{"key": "value"},
			},
			svcRes: groups.Group{
				ID:   group.ID,
				Name: updatedName,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("UpdateGroup", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateGroup(tc.groupReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateGroup", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestEnableGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enGroup := sdkGroup
	enGroup.Status = mgclients.EnabledStatus.String()

	cases := []struct {
		desc     string
		token    string
		groupID  string
		svcRes   groups.Group
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:     "enable group successfully",
			token:    validToken,
			groupID:  group.ID,
			svcRes:   convertGroup(enGroup),
			svcErr:   nil,
			response: enGroup,
			err:      nil,
		},
		{
			desc:     "enable group with invalid group id",
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "enable group with invalid token",
			token:    invalidToken,
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable group with empty token",
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "enable group with empty id",
			token:    validToken,
			groupID:  "",
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "enable group with service response that cannot be unmarshalled",
			token:   validToken,
			groupID: group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("EnableGroup", mock.Anything, tc.token, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableGroup(tc.groupID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableGroup", mock.Anything, tc.token, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	disGroup := sdkGroup
	disGroup.Status = mgclients.DisabledStatus.String()

	cases := []struct {
		desc     string
		token    string
		groupID  string
		svcRes   groups.Group
		svcErr   error
		response sdk.Group
		err      errors.SDKError
	}{
		{
			desc:     "disable group successfully",
			token:    validToken,
			groupID:  group.ID,
			svcRes:   convertGroup(disGroup),
			svcErr:   nil,
			response: disGroup,
			err:      nil,
		},
		{
			desc:     "disable group with invalid group id",
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "disable group with invalid token",
			token:    invalidToken,
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable group with empty token",
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "disable group with empty id",
			token:    validToken,
			groupID:  "",
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "disable group with service response that cannot be unmarshalled",
			token:   validToken,
			groupID: group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("DisableGroup", mock.Anything, tc.token, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableGroup(tc.groupID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableGroup", mock.Anything, tc.token, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc    string
		token   string
		groupID string
		svcErr  error
		err     errors.SDKError
	}{
		{
			desc:    "delete group successfully",
			token:   validToken,
			groupID: group.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "delete group with invalid group id",
			token:   validToken,
			groupID: wrongID,
			svcErr:  svcerr.ErrRemoveEntity,
			err:     errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:    "delete group with invalid token",
			token:   invalidToken,
			groupID: group.ID,
			svcErr:  svcerr.ErrAuthentication,
			err:     errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "delete group with empty token",
			token:   "",
			groupID: group.ID,
			svcErr:  nil,
			err:     errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:    "delete group with empty id",
			token:   validToken,
			groupID: "",
			svcErr:  nil,
			err:     errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("DeleteGroup", mock.Anything, tc.token, tc.groupID).Return(tc.svcErr)
			err := mgsdk.DeleteGroup(tc.groupID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteGroup", mock.Anything, tc.token, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestAddUserToGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc       string
		token      string
		groupID    string
		addUserReq sdk.UsersRelationRequest
		svcErr     error
		err        errors.SDKError
	}{
		{
			desc:    "add user to group successfully",
			token:   validToken,
			groupID: group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "add user to group with invalid token",
			token:   invalidToken,
			groupID: group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "add user to group with empty token",
			token:   "",
			groupID: group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:    "add user to group with invalid group id",
			token:   validToken,
			groupID: wrongID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:    "add user to group with empty group id",
			token:   validToken,
			groupID: "",
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "add users to group with empty relation",
			token:   validToken,
			groupID: group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRelation), http.StatusBadRequest),
		},
		{
			desc:    "add users to group with empty user ids",
			token:   validToken,
			groupID: group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("Assign", mock.Anything, tc.token, tc.groupID, tc.addUserReq.Relation, auth.UsersKind, tc.addUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.AddUserToGroup(tc.groupID, tc.addUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.token, tc.groupID, tc.addUserReq.Relation, auth.UsersKind, tc.addUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestRemoveUserFromGroup(t *testing.T) {
	ts, gsvc := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc          string
		token         string
		groupID       string
		removeUserReq sdk.UsersRelationRequest
		svcErr        error
		err           errors.SDKError
	}{
		{
			desc:    "remove user from group successfully",
			token:   validToken,
			groupID: group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "remove user from group with invalid token",
			token:   invalidToken,
			groupID: group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "remove user from group with empty token",
			token:   "",
			groupID: group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:    "remove user from group with invalid group id",
			token:   validToken,
			groupID: wrongID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:    "remove user from group with empty group id",
			token:   validToken,
			groupID: "",
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "remove users from group with empty user ids",
			token:   validToken,
			groupID: group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := gsvc.On("Unassign", mock.Anything, tc.token, tc.groupID, tc.removeUserReq.Relation, auth.UsersKind, tc.removeUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.RemoveUserFromGroup(tc.groupID, tc.removeUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.token, tc.groupID, tc.removeUserReq.Relation, auth.UsersKind, tc.removeUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func generateTestGroup(t *testing.T) sdk.Group {
	createdAt, err := time.Parse(time.RFC3339, "2023-03-03T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	updatedAt := createdAt
	gr := sdk.Group{
		ID:          testsutil.GenerateUUID(t),
		DomainID:    testsutil.GenerateUUID(t),
		Name:        gName,
		Description: description,
		Metadata:    sdk.Metadata{"role": "client"},
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Status:      mgclients.EnabledStatus.String(),
	}
	return gr
}
