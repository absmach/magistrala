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

	"github.com/absmach/magistrala/groups"
	httpapi "github.com/absmach/magistrala/groups/api/http"
	"github.com/absmach/magistrala/groups/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	policies "github.com/absmach/magistrala/pkg/policies"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
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

func setupGroups() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	authn := new(authnmocks.Authentication)
	httpapi.MakeHandler(svc, authn, mux, logger, "")

	return httptest.NewServer(mux), svc, authn
}

func TestCreateGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
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
	uGroup.Metadata = groups.Metadata{
		"key": make(chan int),
	}
	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupReq        sdk.Group
		svcReq          groups.Group
		svcRes          groups.Group
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "create group successfully",
			domainID: domainID,
			token:    validToken,
			groupReq: createGroupReq,
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
			},
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:     "create group with existing name",
			domainID: domainID,
			token:    validToken,
			groupReq: createGroupReq,
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
			},
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:     "create group with parent",
			domainID: domainID,
			token:    validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
				ParentID:    pGroup.Parent,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
				Parent:      pGroup.Parent,
			},
			svcRes:   pGroup,
			svcErr:   nil,
			response: psdkGroup,
			err:      nil,
		},
		{
			desc:     "create group with invalid parent",
			domainID: domainID,
			token:    validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
				ParentID:    wrongID,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
				Parent:      wrongID,
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "create group with invalid token",
			domainID: domainID,
			token:    invalidToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
			},
			svcRes:          groups.Group{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create group with empty token",
			domainID: domainID,
			token:    "",
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq:   groups.Group{},
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "create group with missing name",
			domainID: domainID,
			token:    validToken,
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
			desc:     "create group with name that is too long",
			domainID: domainID,
			token:    validToken,
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
			desc:     "create group with request that cannot be marshalled",
			domainID: domainID,
			token:    validToken,
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
			desc:     "create group with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupReq: sdk.Group{
				Name:        gName,
				Description: description,
				Metadata:    validMetadata,
			},
			svcReq: groups.Group{
				Name:        gName,
				Description: description,
				Metadata:    groups.Metadata{"role": "client"},
			},
			svcRes:   uGroup,
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("CreateGroup", mock.Anything, tc.session, policies.NewGroupKind, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateGroup(tc.groupReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateGroup", mock.Anything, tc.session, policies.NewGroupKind, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	ts, gsvc, auth := setupGroups()
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
			Status:   groups.EnabledStatus.String(),
		}
		grps = append(grps, gr)
	}

	cases := []struct {
		desc            string
		token           string
		domainID        string
		session         mgauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          groups.Page
		svcRes          groups.Page
		svcErr          error
		authenticateErr error
		response        sdk.GroupsPage
		err             errors.SDKError
	}{
		{
			desc:     "list groups successfully",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  100,
				},
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
			desc:     "list groups with invalid token",
			token:    invalidToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  100,
				},
			},
			svcRes:          groups.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list groups with empty token",
			domainID: domainID,
			token:    "",
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
			},
			svcReq:          groups.Page{},
			svcRes:          groups.Page{},
			svcErr:          nil,
			response:        sdk.GroupsPage{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list groups with zero limit",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  10,
				},
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
			desc:     "list groups with limit greater than max",
			domainID: domainID,
			token:    validToken,
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
			desc:     "list groups with given name",
			domainID: domainID,
			token:    validToken,
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
					Metadata: groups.Metadata{
						"name": "user_89",
					},
				},
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
			desc:     "list groups with invalid level",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  100,
				Level:  6,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidLevel), http.StatusBadRequest),
		},
		{
			desc:     "list groups with invalid page metadata",
			domainID: domainID,
			token:    validToken,
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
			desc:     "list groups with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{{
					ID:   generateUUID(t),
					Name: "group_1",
					Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Groups(tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListParentGroups(t *testing.T) {
	ts, gsvc, auth := setupGroups()
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
			Status:   groups.EnabledStatus.String(),
			ParentID: parentID,
			Level:    1,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}

	cases := []struct {
		desc            string
		token           string
		domainID        string
		session         mgauthn.Session
		pageMeta        sdk.PageMetadata
		parentID        string
		svcReq          groups.Page
		svcRes          groups.Page
		svcErr          error
		authenticateErr error
		response        sdk.GroupsPage
		err             errors.SDKError
	}{
		{
			desc:     "list parent groups successfully",
			domainID: domainID,
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
				ParentID:   parentID,
				Permission: policies.ViewPermission,
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
			domainID: domainID,
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
				ParentID:   parentID,
				Permission: policies.ViewPermission,
				Direction:  1,
				Level:      sdk.MaxLevel,
			},
			svcRes:          groups.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.GroupsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list parent groups with empty token",
			domainID: domainID,
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
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list parent groups with zero limit",
			domainID: domainID,
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
				ParentID:   parentID,
				Permission: policies.ViewPermission,
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
			domainID: domainID,
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
			domainID: domainID,
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
					Metadata: groups.Metadata{
						"name": "user_89",
					},
				},
				ParentID:   parentID,
				Permission: policies.ViewPermission,
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
			domainID: domainID,
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
			domainID: domainID,
			token:    validToken,
			parentID: parentID,
			pageMeta: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				DomainID: domainID,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ParentID:   parentID,
				Permission: policies.ViewPermission,
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
					Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Parents(tc.parentID, tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListChildrenGroups(t *testing.T) {
	ts, gsvc, auth := setupGroups()
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
			Status:   groups.EnabledStatus.String(),
			ParentID: parentID,
			Level:    -1,
		}
		parentID = gr.ID
		grps = append(grps, gr)
	}
	childID := grps[0].ID

	cases := []struct {
		desc            string
		token           string
		domainID        string
		session         mgauthn.Session
		childID         string
		pageMeta        sdk.PageMetadata
		svcReq          groups.Page
		svcRes          groups.Page
		svcErr          error
		authenticateErr error
		response        sdk.GroupsPage
		err             errors.SDKError
	}{
		{
			desc:     "list children groups successfully",
			domainID: domainID,
			token:    validToken,
			childID:  childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ParentID:   childID,
				Permission: policies.ViewPermission,
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
			desc:     "list children groups with invalid token",
			domainID: domainID,
			token:    invalidToken,
			childID:  childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ParentID:   childID,
				Permission: policies.ViewPermission,
				Direction:  -1,
				Level:      sdk.MaxLevel,
			},
			svcRes:          groups.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.GroupsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list children groups with empty token",
			domainID: domainID,
			token:    "",
			childID:  childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list children groups with zero limit",
			domainID: domainID,
			token:    validToken,
			childID:  childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  10,
				},
				ParentID:   childID,
				Permission: policies.ViewPermission,
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
			desc:     "list children groups with limit greater than max",
			domainID: domainID,
			token:    validToken,
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
			desc:     "list children groups with given metadata",
			domainID: domainID,
			token:    validToken,
			childID:  childID,
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
					Metadata: groups.Metadata{
						"name": "user_89",
					},
				},
				ParentID:   childID,
				Permission: policies.ViewPermission,
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
			desc:     "list children groups with invalid page metadata",
			domainID: domainID,
			token:    validToken,
			childID:  childID,
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
			desc:     "list children groups with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			childID:  childID,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: offset,
					Limit:  limit,
				},
				ParentID:   childID,
				Permission: policies.ViewPermission,
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
					Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Children(tc.childID, tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		svcRes          groups.Group
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "view group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes:   group,
			svcErr:   nil,
			response: sdkGroup,
			err:      nil,
		},
		{
			desc:            "view group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         group.ID,
			svcRes:          groups.Group{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view group with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: groups.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
		{
			desc:     "view group with empty id",
			domainID: domainID,
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ViewGroup", mock.Anything, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Group(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroup", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewGroupPermissions(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "view group permissions successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes:   []string{policies.ViewPermission, policies.MembershipPermission},
			svcErr:   nil,
			response: sdk.Group{
				Permissions: []string{policies.ViewPermission, policies.MembershipPermission},
			},
			err: nil,
		},
		{
			desc:            "view group permissions with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         group.ID,
			svcRes:          []string{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view group permissions with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view group permissions with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view group permissions with empty id",
			domainID: domainID,
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ViewGroupPerms", mock.Anything, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupPermissions(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroupPerms", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
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
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupReq        sdk.Group
		svcReq          groups.Group
		svcRes          groups.Group
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "update group successfully",
			domainID: domainID,
			token:    validToken,
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
				Metadata:    groups.Metadata{"key": "value"},
			},
			svcRes:   convertGroup(upGroup),
			svcErr:   nil,
			response: upGroup,
			err:      nil,
		},
		{
			desc:     "update group name with invalid group id",
			domainID: domainID,
			token:    validToken,
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
				Metadata:    groups.Metadata{"key": "value"},
			},
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update group name with invalid token",
			domainID: domainID,
			token:    invalidToken,
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
				Metadata:    groups.Metadata{"key": "value"},
			},
			svcRes:          groups.Group{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update group name with empty token",
			domainID: domainID,
			token:    "",
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
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update group with empty id",
			domainID: domainID,
			token:    validToken,
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
			desc:     "update group with request that can't be marshalled",
			domainID: domainID,
			token:    validToken,
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
			desc:     "update group with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
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
				Metadata:    groups.Metadata{"key": "value"},
			},
			svcRes: groups.Group{
				ID:   group.ID,
				Name: updatedName,
				Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("UpdateGroup", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateGroup(tc.groupReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateGroup", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enGroup := sdkGroup
	enGroup.Status = groups.EnabledStatus.String()

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		svcRes          groups.Group
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "enable group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes:   convertGroup(enGroup),
			svcErr:   nil,
			response: enGroup,
			err:      nil,
		},
		{
			desc:     "enable group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:            "enable group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         group.ID,
			svcRes:          groups.Group{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "enable group with empty id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "enable group with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("EnableGroup", mock.Anything, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableGroup(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableGroup", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	disGroup := sdkGroup
	disGroup.Status = groups.DisabledStatus.String()

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		svcRes          groups.Group
		svcErr          error
		authenticateErr error
		response        sdk.Group
		err             errors.SDKError
	}{
		{
			desc:     "disable group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes:   convertGroup(disGroup),
			svcErr:   nil,
			response: disGroup,
			err:      nil,
		},
		{
			desc:     "disable group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcRes:   groups.Group{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:            "disable group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         group.ID,
			svcRes:          groups.Group{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Group{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "disable group with empty id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			svcRes:   groups.Group{},
			svcErr:   nil,
			response: sdk.Group{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "disable group with service response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcRes: groups.Group{
				ID:   group.ID,
				Name: "group_1",
				Metadata: groups.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("DisableGroup", mock.Anything, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableGroup(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableGroup", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "delete group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:            "delete group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         group.ID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "delete group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete group with empty id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			svcErr:   nil,
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("DeleteGroup", mock.Anything, tc.session, tc.groupID).Return(tc.svcErr)
			err := mgsdk.DeleteGroup(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteGroup", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddUserToGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		addUserReq      sdk.UsersRelationRequest
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "add user to group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "add user to group with invalid token",
			domainID: domainID,
			token:    invalidToken,
			groupID:  group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add user to group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add user to group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add user to group with empty group id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add users to group with empty relation",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRelation), http.StatusBadRequest),
		},
		{
			desc:     "add users to group with empty user ids",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.groupID, tc.addUserReq.Relation, policies.UsersKind, tc.addUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.AddUserToGroup(tc.groupID, tc.addUserReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.session, tc.groupID, tc.addUserReq.Relation, policies.UsersKind, tc.addUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveUserFromGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		removeUserReq   sdk.UsersRelationRequest
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove user from group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "remove user from group with invalid token",
			domainID: domainID,
			token:    invalidToken,
			groupID:  group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove user from group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove user from group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove user from group with empty group id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove users from group with empty user ids",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.groupID, tc.removeUserReq.Relation, policies.UsersKind, tc.removeUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.RemoveUserFromGroup(tc.groupID, tc.removeUserReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.session, tc.groupID, tc.removeUserReq.Relation, policies.UsersKind, tc.removeUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
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
		Status:      groups.EnabledStatus.String(),
	}
	return gr
}
