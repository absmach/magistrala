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

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/groups"
	httpapi "github.com/absmach/supermq/groups/api/http"
	"github.com/absmach/supermq/groups/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	oauth2mocks "github.com/absmach/supermq/pkg/oauth2/mocks"
	"github.com/absmach/supermq/pkg/roles"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/absmach/supermq/pkg/uuid"
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

	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return(roleName)
	authn := new(authnmocks.Authentication)
	httpapi.MakeHandler(svc, authn, mux, logger, "", idp)

	return httptest.NewServer(mux), svc, authn
}

func TestCreateGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
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
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("CreateGroup", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, []roles.RoleProvision{}, tc.svcErr)
			resp, err := mgsdk.CreateGroup(tc.groupReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateGroup", mock.Anything, tc.session, tc.svcReq)
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
		GroupsURL: ts.URL,
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
		session         smqauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          groups.PageMeta
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
			svcReq: groups.PageMeta{
				Offset:  offset,
				Limit:   100,
				Actions: []string{},
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
			svcReq: groups.PageMeta{
				Offset:  offset,
				Limit:   100,
				Actions: []string{},
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
			svcReq:          groups.PageMeta{},
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
			svcReq: groups.PageMeta{
				Offset:  offset,
				Limit:   10,
				Actions: []string{},
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
			svcReq:   groups.PageMeta{},
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
			svcReq: groups.PageMeta{
				Offset: 0,
				Limit:  10,
				Metadata: groups.Metadata{
					"name": "user_89",
				},
				Actions: []string{},
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
			svcReq:   groups.PageMeta{},
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
			svcReq: groups.PageMeta{
				Offset:  offset,
				Limit:   limit,
				Actions: []string{},
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Groups(tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, tc.svcReq)
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
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
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

func TestUpdateGroup(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	upGroup := sdkGroup
	upGroup.Name = updatedName
	upGroup.Description = updatedDescription
	upGroup.Metadata = sdk.Metadata{"key": "value"}

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group.ID = generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
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
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enGroup := sdkGroup
	enGroup.Status = groups.EnabledStatus.String()

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
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
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	disGroup := sdkGroup
	disGroup.Status = groups.DisabledStatus.String()

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
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
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
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

func TestSetGroupParent(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)
	parentID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		groupID         string
		parentID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "set group parent successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  groupID,
			parentID: parentID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "set group parent with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         groupID,
			parentID:        parentID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "set group parent with empty token",
			domainID: domainID,
			token:    "",
			groupID:  groupID,
			parentID: parentID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "set group parent with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			parentID: parentID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "set group parent with empty group id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			parentID: parentID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "set group parent with empty parent id",
			domainID: domainID,
			token:    validToken,
			groupID:  groupID,
			parentID: "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidIDFormat), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("AddParentGroup", mock.Anything, tc.session, tc.groupID, tc.parentID).Return(tc.svcErr)
			err := mgsdk.SetGroupParent(tc.groupID, tc.domainID, tc.parentID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AddParentGroup", mock.Anything, tc.session, tc.groupID, tc.parentID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveGroupParent(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)
	parentID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		groupID         string
		parentID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove group parent successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  groupID,
			parentID: parentID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove group parent with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         groupID,
			parentID:        parentID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove group parent with empty token",
			domainID: domainID,
			token:    "",
			groupID:  groupID,
			parentID: parentID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove group parent with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			parentID: parentID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove group parent with empty group id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			parentID: parentID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RemoveParentGroup", mock.Anything, tc.session, tc.groupID).Return(tc.svcErr)
			err := mgsdk.RemoveGroupParent(tc.groupID, tc.domainID, tc.parentID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveParentGroup", mock.Anything, tc.session, tc.groupID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddChildrenGroups(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)
	childID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		groupID         string
		childrenIDs     []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:        "add children group successfully",
			domainID:    domainID,
			token:       validToken,
			groupID:     groupID,
			childrenIDs: []string{childID},
			svcErr:      nil,
			err:         nil,
		},
		{
			desc:            "add children group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         groupID,
			childrenIDs:     []string{childID},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "add children group with empty token",
			domainID:    domainID,
			token:       "",
			groupID:     groupID,
			childrenIDs: []string{childID},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "add children group with invalid group id",
			domainID:    domainID,
			token:       validToken,
			groupID:     wrongID,
			childrenIDs: []string{childID},
			svcErr:      svcerr.ErrAuthorization,
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:        "add children group with empty group id",
			domainID:    domainID,
			token:       validToken,
			groupID:     "",
			childrenIDs: []string{childID},
			svcErr:      nil,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:        "add children group with empty children ids",
			domainID:    domainID,
			token:       validToken,
			groupID:     groupID,
			childrenIDs: []string{},
			svcErr:      nil,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingChildrenGroupIDs), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("AddChildrenGroups", mock.Anything, tc.session, tc.groupID, tc.childrenIDs).Return(tc.svcErr)
			err := mgsdk.AddChildren(tc.groupID, tc.domainID, tc.childrenIDs, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AddChildrenGroups", mock.Anything, tc.session, tc.groupID, tc.childrenIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveChildrenGroups(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)
	childID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		groupID         string
		childrenIDs     []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:        "remove children group successfully",
			domainID:    domainID,
			token:       validToken,
			groupID:     groupID,
			childrenIDs: []string{childID},
			svcErr:      nil,
			err:         nil,
		},
		{
			desc:            "remove children group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         groupID,
			childrenIDs:     []string{childID},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "remove children group with empty token",
			domainID:    domainID,
			token:       "",
			groupID:     groupID,
			childrenIDs: []string{childID},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "remove children group with invalid group id",
			domainID:    domainID,
			token:       validToken,
			groupID:     wrongID,
			childrenIDs: []string{childID},
			svcErr:      svcerr.ErrAuthorization,
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:        "remove children group with empty group id",
			domainID:    domainID,
			token:       validToken,
			groupID:     "",
			childrenIDs: []string{childID},
			svcErr:      nil,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:        "remove children group with empty children ids",
			domainID:    domainID,
			token:       validToken,
			groupID:     groupID,
			childrenIDs: []string{},
			svcErr:      nil,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingChildrenGroupIDs), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RemoveChildrenGroups", mock.Anything, tc.session, tc.groupID, tc.childrenIDs).Return(tc.svcErr)
			err := mgsdk.RemoveChildren(tc.groupID, tc.domainID, tc.childrenIDs, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveChildrenGroups", mock.Anything, tc.session, tc.groupID, tc.childrenIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllChildrenGroups(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		groupID         string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all children group successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  groupID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all children group with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			groupID:         groupID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all children group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  groupID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all children group with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all children group with empty group id",
			domainID: domainID,
			token:    validToken,
			groupID:  "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RemoveAllChildrenGroups", mock.Anything, tc.session, tc.groupID).Return(tc.svcErr)
			err := mgsdk.RemoveAllChildren(tc.groupID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveAllChildrenGroups", mock.Anything, tc.session, tc.groupID)
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
		GroupsURL: ts.URL,
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
		session         smqauthn.Session
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
			childID:  childID,
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
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListChildrenGroups", mock.Anything, tc.session, tc.childID, int64(1), int64(0), mock.Anything).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Children(tc.childID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListChildrenGroups", mock.Anything, tc.session, tc.childID, int64(1), int64(0), mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestHierarchy(t *testing.T) {
	ts, gsvc, auth := setupGroups()
	defer ts.Close()

	var grps []sdk.Group
	conf := sdk.Config{
		GroupsURL: ts.URL,
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
		session         smqauthn.Session
		groupID         string
		pageMeta        sdk.PageMetadata
		svcReq          groups.HierarchyPageMeta
		svcRes          groups.HierarchyPage
		svcErr          error
		authenticateErr error
		response        sdk.GroupsHierarchyPage
		err             errors.SDKError
	}{
		{
			desc:     "list hierarchy successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  childID,
			pageMeta: sdk.PageMetadata{
				Level: 2,
				Tree:  false,
			},
			svcReq: groups.HierarchyPageMeta{
				Level:     2,
				Direction: -1,
				Tree:      false,
			},
			svcRes: groups.HierarchyPage{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     2,
					Direction: +1,
					Tree:      false,
				},
				Groups: convertGroups(grps[1:]),
			},
			response: sdk.GroupsHierarchyPage{
				Level:     2,
				Direction: +1,
				Groups:    grps[1:],
			},
			err: nil,
		},
		{
			desc:     "list hierarchy with invalid token",
			domainID: domainID,
			token:    validToken,
			groupID:  childID,
			pageMeta: sdk.PageMetadata{
				Level: 2,
				Tree:  false,
			},
			svcReq: groups.HierarchyPageMeta{
				Level:     2,
				Direction: -1,
				Tree:      false,
			},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.GroupsHierarchyPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list hierarchy with empty token",
			domainID: domainID,
			token:    "",
			groupID:  childID,
			pageMeta: sdk.PageMetadata{
				Level: 2,
				Tree:  false,
			},
			svcReq: groups.HierarchyPageMeta{
				Level:     2,
				Direction: -1,
				Tree:      false,
			},
			svcErr:   nil,
			response: sdk.GroupsHierarchyPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list hierarchy with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			pageMeta: sdk.PageMetadata{
				Level: 2,
				Tree:  false,
			},
			svcReq: groups.HierarchyPageMeta{
				Level:     2,
				Direction: -1,
				Tree:      false,
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list hierarchy with response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupID:  childID,
			pageMeta: sdk.PageMetadata{
				Level: 2,
				Tree:  false,
			},
			svcReq: groups.HierarchyPageMeta{
				Level:     2,
				Direction: -1,
				Tree:      false,
			},
			svcRes: groups.HierarchyPage{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     2,
					Direction: +1,
					Tree:      false,
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
			response: sdk.GroupsHierarchyPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("RetrieveGroupHierarchy", mock.Anything, tc.session, tc.groupID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Hierarchy(tc.groupID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveGroupHierarchy", mock.Anything, tc.session, tc.groupID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestCreateGroupRole(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	optionalActions := []string{"create", "update"}
	optionalMembers := []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)}
	rReq := sdk.RoleReq{
		RoleName:        roleName,
		OptionalActions: optionalActions,
		OptionalMembers: optionalMembers,
	}
	userID := testsutil.GenerateUUID(t)
	groupID := testsutil.GenerateUUID(t)
	now := time.Now().UTC()
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      rReq.RoleName,
		EntityID:  groupID,
		CreatedBy: userID,
		CreatedAt: now,
	}
	roleProvision := roles.RoleProvision{
		Role:            role,
		OptionalActions: optionalActions,
		OptionalMembers: optionalMembers,
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleReq         sdk.RoleReq
		svcRes          roles.RoleProvision
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "create group role successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleReq:  rReq,
			svcRes:   roleProvision,
			svcErr:   nil,
			response: convertRoleProvision(roleProvision),
			err:      nil,
		},
		{
			desc:            "create group role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleReq:         rReq,
			svcRes:          roles.RoleProvision{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create group role with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "create group role with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "create group role with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidIDFormat), http.StatusBadRequest),
		},
		{
			desc:     "create group role with empty role name",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleReq: sdk.RoleReq{
				RoleName:        "",
				OptionalActions: []string{"create", "update"},
				OptionalMembers: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			svcRes:   roles.RoleProvision{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleName), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("AddRole", mock.Anything, tc.session, tc.groupID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateGroupRole(tc.groupID, tc.domainID, tc.roleReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AddRole", mock.Anything, tc.session, tc.groupID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListGroupRoles(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	groupID := testsutil.GenerateUUID(t)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  groupID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		pageMeta        sdk.PageMetadata
		svcRes          roles.RolePage
		svcErr          error
		authenticateErr error
		response        sdk.RolesPage
		err             errors.SDKError
	}{
		{
			desc:     "list group roles successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes: roles.RolePage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Roles:  []roles.Role{role},
			},
			svcErr: nil,
			response: sdk.RolesPage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Roles:  []sdk.Role{convertRole(role)},
			},
			err: nil,
		},
		{
			desc:     "list group roles with invalid token",
			token:    invalidToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:          roles.RolePage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.RolesPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list group roles with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   roles.RolePage{},
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list group roles with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   roles.RolePage{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group roles with empty group id",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			groupID:  "",
			svcRes:   roles.RolePage{},
			svcErr:   nil,
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RetrieveAllRoles", mock.Anything, tc.session, tc.groupID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupRoles(tc.groupID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveAllRoles", mock.Anything, tc.session, tc.groupID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewGroupRole(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	groupID := testsutil.GenerateUUID(t)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  groupID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "view group role successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   role.ID,
			svcRes:   role,
			svcErr:   nil,
			response: convertRole(role),
			err:      nil,
		},
		{
			desc:            "view group role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          role.ID,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view group role with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   role.ID,
			svcRes:   roles.Role{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view group role with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view group role with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "view group role with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			svcRes:   roles.Role{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RetrieveRole", mock.Anything, tc.session, tc.groupID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupRole(tc.groupID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveRole", mock.Anything, tc.session, tc.groupID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateGroupRole(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	groupID := testsutil.GenerateUUID(t)
	roleID := testsutil.GenerateUUID(t)
	newRoleName := "newTest"
	userID := testsutil.GenerateUUID(t)
	createdAt := time.Now().UTC().Add(-time.Hour)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      newRoleName,
		EntityID:  groupID,
		CreatedBy: userID,
		CreatedAt: createdAt,
		UpdatedBy: userID,
		UpdatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		newRoleName     string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:        "update group role successfully",
			token:       validToken,
			domainID:    domainID,
			groupID:     groupID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      role,
			svcErr:      nil,
			response:    convertRole(role),
			err:         nil,
		},
		{
			desc:            "update group role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			newRoleName:     newRoleName,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "update group role with empty token",
			token:       "",
			domainID:    domainID,
			groupID:     groupID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "update group role with invalid group id",
			token:       validToken,
			domainID:    domainID,
			groupID:     testsutil.GenerateUUID(t),
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			svcErr:      svcerr.ErrAuthorization,
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:        "update group role with empty group id",
			token:       validToken,
			domainID:    domainID,
			groupID:     "",
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			svcErr:      nil,
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("UpdateRoleName", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.newRoleName).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateGroupRole(tc.groupID, tc.roleID, tc.newRoleName, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateRoleName", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.newRoleName)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteGroupRole(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete group role successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "delete group role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "delete group role with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete group role with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "delete group role with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "delete group role with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RemoveRole", mock.Anything, tc.session, tc.groupID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.DeleteGroupRole(tc.groupID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveRole", mock.Anything, tc.session, tc.groupID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddGroupRoleActions(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		actions         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add group role actions successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  actions,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "add group role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add group role actions with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add group role actions with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add group role actions with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add group role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add group role actions with empty actions",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  []string{},
			svcErr:   nil,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyEntityType), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleAddActions", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.actions).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddGroupRoleActions(tc.groupID, tc.roleID, tc.domainID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddActions", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListGroupRoleActions(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "list group role actions successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "list group role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list group role actions with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list group role actions with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group role actions with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "list group role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleListActions", mock.Anything, tc.session, tc.groupID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupRoleActions(tc.groupID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListActions", mock.Anything, tc.session, tc.groupID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveGroupRoleActions(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		actions         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove group role actions successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  actions,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove group role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove group role actions with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove group role actions with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove group role actions with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove group role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove group role actions with empty actions",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			actions:  []string{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyEntityType), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveActions", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.actions).Return(tc.svcErr)
			err := mgsdk.RemoveGroupRoleActions(tc.groupID, tc.roleID, tc.domainID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveActions", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllGroupRoleActions(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all group role actions successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all group role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all group role actions with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all group role actions with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all group role actions with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove all group role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all group role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveAllActions", mock.Anything, tc.session, tc.groupID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllGroupRoleActions(tc.groupID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllActions", mock.Anything, tc.session, tc.groupID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddGroupRoleMembers(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		members         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add group role members successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  members,
			svcRes:   members,
			svcErr:   nil,
			response: members,
			err:      nil,
		},
		{
			desc:            "add group role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add group role members with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add group role members with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add group role members with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add group role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add group role members with empty members",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  []string{},
			svcErr:   nil,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleMembers), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleAddMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.members).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddGroupRoleMembers(tc.groupID, tc.roleID, tc.domainID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListGroupRoleMembers(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		pageMeta        sdk.PageMetadata
		svcRes          roles.MembersPage
		svcErr          error
		authenticateErr error
		response        sdk.RoleMembersPage
		err             errors.SDKError
	}{
		{
			desc:     "list group role members successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			svcRes: roles.MembersPage{
				Total:   2,
				Offset:  0,
				Limit:   5,
				Members: members,
			},
			svcErr: nil,
			response: sdk.RoleMembersPage{
				Total:   2,
				Offset:  0,
				Limit:   5,
				Members: members,
			},
			err: nil,
		},
		{
			desc:     "list group role members with invalid token",
			token:    invalidToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list group role members with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list group role members with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group role members with empty group id",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			groupID: "",
			roleID:  roleID,
			err:     errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "list group role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: invalid,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group role members with empty role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: "",
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleListMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.GroupRoleMembers(tc.groupID, tc.roleID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveGroupRoleMembers(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		members         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove group role members successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  members,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove group role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove group role members with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove group role members with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove group role members with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove group role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove group role members with empty members",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			members:  []string{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleMembers), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.members).Return(tc.svcErr)
			err := mgsdk.RemoveGroupRoleMembers(tc.groupID, tc.roleID, tc.domainID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveMembers", mock.Anything, tc.session, tc.groupID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllGroupRoleMembers(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	groupID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		groupID         string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all group role members successfully",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all group role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			groupID:         groupID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all group role members with empty token",
			token:    "",
			domainID: domainID,
			groupID:  groupID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all group role members with invalid group id",
			token:    validToken,
			domainID: domainID,
			groupID:  testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all group role members with empty group id",
			token:    validToken,
			domainID: domainID,
			groupID:  "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove all group role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all group role members with empty role id",
			token:    validToken,
			domainID: domainID,
			groupID:  groupID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveAllMembers", mock.Anything, tc.session, tc.groupID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllGroupRoleMembers(tc.groupID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllMembers", mock.Anything, tc.session, tc.groupID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListAvailableGroupRoleActions(t *testing.T) {
	ts, csvc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		GroupsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "list available role actions successfully",
			token:    validToken,
			domainID: domainID,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "list available role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list available role actions with empty token",
			token:    "",
			domainID: domainID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list available role actions with empty domain id",
			token:    validToken,
			domainID: "",
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("ListAvailableActions", mock.Anything, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AvailableGroupRoleActions(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListAvailableActions", mock.Anything, tc.session)
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
