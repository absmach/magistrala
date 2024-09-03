// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	httpapi "github.com/absmach/magistrala/auth/api/http/domains"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	internalapi "github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	authDomain, sdkDomain = generateTestDomain(&testing.T{})
	authDomainReq         = auth.Domain{
		Name:     authDomain.Name,
		Metadata: authDomain.Metadata,
		Tags:     authDomain.Tags,
		Alias:    authDomain.Alias,
	}
	sdkDomainReq = sdk.Domain{
		Name:     sdkDomain.Name,
		Metadata: sdkDomain.Metadata,
		Tags:     sdkDomain.Tags,
		Alias:    sdkDomain.Alias,
	}
	updatedDomianName = "updated-domain"
)

func setupDomains() (*httptest.Server, *authmocks.Service) {
	svc := new(authmocks.Service)
	logger := mglog.NewMock()
	mux := chi.NewRouter()

	mux = httpapi.MakeHandler(svc, mux, logger)
	return httptest.NewServer(mux), svc
}

func TestCreateDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		domain   sdk.Domain
		svcReq   auth.Domain
		svcRes   auth.Domain
		svcErr   error
		response sdk.Domain
		err      error
	}{
		{
			desc:     "create domain successfully",
			token:    validToken,
			domain:   sdkDomainReq,
			svcReq:   authDomainReq,
			svcRes:   authDomain,
			svcErr:   nil,
			response: sdkDomain,
			err:      nil,
		},
		{
			desc:     "create domain with invalid token",
			token:    invalidToken,
			domain:   sdkDomainReq,
			svcReq:   authDomainReq,
			svcRes:   auth.Domain{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create domain with empty token",
			token:    "",
			domain:   sdkDomainReq,
			svcReq:   authDomainReq,
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "create domain with empty name",
			token: validToken,
			domain: sdk.Domain{
				Name:     "",
				Metadata: sdkDomain.Metadata,
				Tags:     sdkDomain.Tags,
				Alias:    sdkDomain.Alias,
			},
			svcReq:   auth.Domain{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingName, http.StatusBadRequest),
		},
		{
			desc:  "create domain with request that cannot be marshalled",
			token: validToken,
			domain: sdk.Domain{
				Name: sdkDomain.Name,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   auth.Domain{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:   "create domain with response that cannot be unmarshalled",
			token:  validToken,
			domain: sdkDomainReq,
			svcReq: authDomainReq,
			svcRes: auth.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("CreateDomain", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateDomain(tc.domain, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateDomain", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	upDomainSDK := sdkDomain
	upDomainSDK.Name = updatedDomianName
	upDomainAuth := authDomain
	upDomainAuth.Name = updatedDomianName

	cases := []struct {
		desc     string
		token    string
		domainID string
		domain   sdk.Domain
		svcRes   auth.Domain
		svcErr   error
		response sdk.Domain
		err      error
	}{
		{
			desc:     "update domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			domain: sdk.Domain{
				ID:   sdkDomain.ID,
				Name: updatedDomianName,
			},
			svcRes:   upDomainAuth,
			svcErr:   nil,
			response: upDomainSDK,
			err:      nil,
		},
		{
			desc:     "update domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			domain: sdk.Domain{
				ID:   sdkDomain.ID,
				Name: updatedDomianName,
			},
			svcRes:   auth.Domain{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			domain: sdk.Domain{
				ID:   sdkDomain.ID,
				Name: updatedDomianName,
			},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update domain with invalid domain ID",
			token:    validToken,
			domainID: wrongID,
			domain: sdk.Domain{
				ID:   wrongID,
				Name: updatedDomianName,
			},
			svcRes:   auth.Domain{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "update domain with empty id",
			token:    validToken,
			domainID: "",
			domain: sdk.Domain{
				Name: sdkDomain.Name,
			},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "update domain with request that cannot be marshalled",
			token:    validToken,
			domainID: sdkDomain.ID,
			domain: sdk.Domain{
				ID:   sdkDomain.ID,
				Name: sdkDomain.Name,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "update domain with response that cannot be unmarshalled",
			token:    validToken,
			domainID: sdkDomain.ID,
			domain: sdk.Domain{
				ID:   sdkDomain.ID,
				Name: sdkDomain.Name,
			},
			svcRes: auth.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateDomain", mock.Anything, tc.token, tc.domainID, mock.Anything).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateDomain(tc.domain, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateDomain", mock.Anything, tc.token, tc.domainID, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcRes   auth.Domain
		svcErr   error
		response sdk.Domain
		err      error
	}{
		{
			desc:     "view domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes:   authDomain,
			svcErr:   nil,
			response: sdkDomain,
			err:      nil,
		},
		{
			desc:     "view domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcRes:   auth.Domain{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view domain with invalid domain ID",
			token:    validToken,
			domainID: wrongID,
			svcRes:   auth.Domain{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view domain with empty id",
			token:    validToken,
			domainID: "",
			svcRes:   auth.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "view domain with response that cannot be unmarshalled",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes: auth.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveDomain", mock.Anything, tc.token, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Domain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveDomain", mock.Anything, tc.token, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDomainPermissions(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcRes   auth.Permissions
		svcErr   error
		response sdk.Domain
		err      error
	}{
		{
			desc:     "retrieve domain permissions successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes:   auth.Permissions{auth.ViewPermission},
			svcErr:   nil,
			response: sdk.Domain{
				Permissions: []string{auth.ViewPermission},
			},
			err: nil,
		},
		{
			desc:     "retrieve domain permissions with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcRes:   auth.Permissions{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "retrieve domain permissions with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   auth.Permissions{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "retrieve domain permissions with empty domain id",
			token:    validToken,
			domainID: "",
			svcRes:   auth.Permissions{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:     "retrieve domain permissions with invalid domain id",
			token:    validToken,
			domainID: wrongID,
			svcRes:   auth.Permissions{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveDomainPermissions", mock.Anything, tc.token, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DomainPermissions(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveDomainPermissions", mock.Anything, tc.token, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListDomians(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   auth.Page
		svcRes   auth.DomainsPage
		svcErr   error
		response sdk.DomainsPage
		err      error
	}{
		{
			desc:  "list domains successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{authDomain},
			},
			svcErr: nil,
			response: sdk.DomainsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Domains: []sdk.Domain{sdkDomain},
			},
			err: nil,
		},
		{
			desc:  "list domains with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes:   auth.DomainsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list domains with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   auth.Page{},
			svcRes:   auth.DomainsPage{},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list domains with invalid page metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   auth.Page{},
			svcRes:   auth.DomainsPage{},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list domains with request that cannot be marshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: auth.DomainsPage{
				Total: 1,
				Domains: []auth.Domain{{
					Name:     authDomain.Name,
					Metadata: mgclients.Metadata{"key": make(chan int)},
				}},
			},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListDomains", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Domains(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListDomains", mock.Anything, tc.token, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListUserDomains(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   auth.Page
		svcRes   auth.DomainsPage
		svcErr   error
		response sdk.DomainsPage
		err      error
	}{
		{
			desc:  "list user domains successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				User:   sdkDomain.CreatedBy,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{authDomain},
			},
			svcErr: nil,
			response: sdk.DomainsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Domains: []sdk.Domain{sdkDomain},
			},
			err: nil,
		},
		{
			desc:  "list user domains with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				User:   sdkDomain.CreatedBy,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes:   auth.DomainsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list user domains with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				User:   sdkDomain.CreatedBy,
			},
			svcReq:   auth.Page{},
			svcRes:   auth.DomainsPage{},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list user domains with request that cannot be marshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				User:   sdkDomain.CreatedBy,
			},
			svcReq: auth.Page{
				Offset: 0,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: auth.DomainsPage{
				Total: 1,
				Domains: []auth.Domain{{
					Name:     authDomain.Name,
					Metadata: mgclients.Metadata{"key": make(chan int)},
				}},
			},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
		{
			desc:  "list user domains with invalid page metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
				User: sdkDomain.CreatedBy,
			},
			svcReq:   auth.Page{},
			svcRes:   auth.DomainsPage{},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListUserDomains", mock.Anything, tc.token, tc.pageMeta.User, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListUserDomains(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListUserDomains", mock.Anything, tc.token, tc.pageMeta.User, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestEnableDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	enable := auth.EnabledStatus

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcReq   auth.DomainReq
		svcRes   auth.Domain
		svcErr   error
		err      error
	}{
		{
			desc:     "enable domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcReq: auth.DomainReq{
				Status: &enable,
			},
			svcRes: authDomain,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "enable domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcReq: auth.DomainReq{
				Status: &enable,
			},
			svcRes: auth.Domain{},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcReq:   auth.DomainReq{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "enable domain with empty domain id",
			token:    validToken,
			domainID: "",
			svcReq:   auth.DomainReq{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ChangeDomainStatus", mock.Anything, tc.token, tc.domainID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			err := mgsdk.EnableDomain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ChangeDomainStatus", mock.Anything, tc.token, tc.domainID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	disable := auth.DisabledStatus

	cases := []struct {
		desc     string
		token    string
		domainID string
		svcReq   auth.DomainReq
		svcRes   auth.Domain
		svcErr   error
		err      error
	}{
		{
			desc:     "disable domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcReq: auth.DomainReq{
				Status: &disable,
			},
			svcRes: authDomain,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "disable domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcReq: auth.DomainReq{
				Status: &disable,
			},
			svcRes: auth.Domain{},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcReq:   auth.DomainReq{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "disable domain with empty domain id",
			token:    validToken,
			domainID: "",
			svcReq:   auth.DomainReq{},
			svcRes:   auth.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ChangeDomainStatus", mock.Anything, tc.token, tc.domainID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			err := mgsdk.DisableDomain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ChangeDomainStatus", mock.Anything, tc.token, tc.domainID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestAddUserToDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	newUser := testsutil.GenerateUUID(t)

	cases := []struct {
		desc             string
		token            string
		domainID         string
		addUserDomainReq sdk.UsersRelationRequest
		svcErr           error
		err              error
	}{
		{
			desc:     "add user to domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{newUser},
				Relation: auth.MemberRelation,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "add user to domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{newUser},
				Relation: auth.MemberRelation,
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add user to domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{newUser},
				Relation: auth.MemberRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add user to domain with empty domain id",
			token:    validToken,
			domainID: "",
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{newUser},
				Relation: auth.MemberRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:     "add user to domain with empty user id",
			token:    validToken,
			domainID: sdkDomain.ID,
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{},
				Relation: auth.MemberRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:     "add user to domain with empty relation",
			token:    validToken,
			domainID: sdkDomain.ID,
			addUserDomainReq: sdk.UsersRelationRequest{
				UserIDs:  []string{newUser},
				Relation: "",
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMissingRelation, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AssignUsers", mock.Anything, tc.token, tc.domainID, tc.addUserDomainReq.UserIDs, tc.addUserDomainReq.Relation).Return(tc.svcErr)
			err := mgsdk.AddUserToDomain(tc.domainID, tc.addUserDomainReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AssignUsers", mock.Anything, tc.token, tc.domainID, tc.addUserDomainReq.UserIDs, tc.addUserDomainReq.Relation)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestRemoveUserFromDomain(t *testing.T) {
	ds, svc := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	removeUserID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		token    string
		domainID string
		userID   string
		svcErr   error
		err      error
	}{
		{
			desc:     "remove user from domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			userID:   removeUserID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "remove user from domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			userID:   removeUserID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove user from domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			userID:   removeUserID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove user from domain with empty domain id",
			token:    validToken,
			domainID: "",
			userID:   removeUserID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:     "remove user from domain with empty user id",
			token:    validToken,
			domainID: sdkDomain.ID,
			userID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicy, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UnassignUser", mock.Anything, tc.token, tc.domainID, tc.userID).Return(tc.svcErr)
			err := mgsdk.RemoveUserFromDomain(tc.domainID, tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UnassignUser", mock.Anything, tc.token, tc.domainID, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func generateTestDomain(t *testing.T) (auth.Domain, sdk.Domain) {
	createdAt, err := time.Parse(time.RFC3339, "2024-04-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %s", err))
	ownerID := testsutil.GenerateUUID(t)
	ad := auth.Domain{
		ID:        testsutil.GenerateUUID(t),
		Name:      "test-domain",
		Metadata:  mgclients.Metadata(validMetadata),
		Tags:      []string{"tag1", "tag2"},
		Alias:     "test-alias",
		Status:    auth.EnabledStatus,
		CreatedBy: ownerID,
		CreatedAt: createdAt,
		UpdatedBy: ownerID,
		UpdatedAt: createdAt,
	}

	sd := sdk.Domain{
		ID:        ad.ID,
		Name:      ad.Name,
		Metadata:  validMetadata,
		Tags:      ad.Tags,
		Alias:     ad.Alias,
		Status:    ad.Status.String(),
		CreatedBy: ad.CreatedBy,
		CreatedAt: ad.CreatedAt,
		UpdatedBy: ad.UpdatedBy,
		UpdatedAt: ad.UpdatedAt,
	}
	return ad, sd
}
