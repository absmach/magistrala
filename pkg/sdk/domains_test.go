// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	domainapi "github.com/absmach/supermq/domains/api/http"
	"github.com/absmach/supermq/domains/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	authDomain, sdkDomain = generateTestDomain(&testing.T{})
	authDomainReq         = domains.Domain{
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

func setupDomains() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	authn := new(authnmocks.Authentication)

	handler := domainapi.MakeHandler(svc, authn, mux, logger, "", idp)
	return httptest.NewServer(handler), svc, authn
}

func TestCreateDomain(t *testing.T) {
	ds, svc, auth := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		domain   sdk.Domain
		svcReq   domains.Domain
		svcRes   domains.Domain
		svcErr   error
		authnErr error
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
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create domain with empty token",
			token:    "",
			domain:   sdkDomainReq,
			svcReq:   authDomainReq,
			svcRes:   domains.Domain{},
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
			svcReq:   domains.Domain{},
			svcRes:   domains.Domain{},
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
			svcReq:   domains.Domain{},
			svcRes:   domains.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:   "create domain with response that cannot be unmarshalled",
			token:  validToken,
			domain: sdkDomainReq,
			svcReq: authDomainReq,
			svcRes: domains.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: domains.Metadata{
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
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("CreateDomain", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, []roles.RoleProvision{}, tc.svcErr)
			resp, err := mgsdk.CreateDomain(tc.domain, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateDomain", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	ds, svc, authn := setupDomains()
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
		session  smqauthn.Session
		domainID string
		domain   sdk.Domain
		svcRes   domains.Domain
		svcErr   error
		authnErr error
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
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
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
			svcRes:   domains.Domain{},
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
			svcRes:   domains.Domain{},
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
			svcRes:   domains.Domain{},
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
			svcRes:   domains.Domain{},
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
			svcRes: domains.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: domains.Metadata{
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
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateDomain", mock.Anything, tc.session, tc.domainID, mock.Anything).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateDomain(tc.domain, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateDomain", mock.Anything, tc.session, tc.domainID, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewDomain(t *testing.T) {
	ds, svc, authn := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		authnErr error
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
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view domain with invalid domain ID",
			token:    validToken,
			domainID: wrongID,
			svcRes:   domains.Domain{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Domain{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view domain with empty id",
			token:    validToken,
			domainID: "",
			svcRes:   domains.Domain{},
			svcErr:   nil,
			response: sdk.Domain{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "view domain with response that cannot be unmarshalled",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes: domains.Domain{
				ID:   authDomain.ID,
				Name: authDomain.Name,
				Metadata: domains.Metadata{
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
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RetrieveDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Domain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveDomain", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListDomians(t *testing.T) {
	ds, svc, authn := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		pageMeta sdk.PageMetadata
		svcReq   domains.Page
		svcRes   domains.DomainsPage
		svcErr   error
		authnErr error
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
			svcReq: domains.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			svcRes: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{authDomain},
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
			svcReq: domains.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			svcRes:   domains.DomainsPage{},
			authnErr: svcerr.ErrAuthentication,
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
			svcReq:   domains.Page{},
			svcRes:   domains.DomainsPage{},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
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
			svcReq:   domains.Page{},
			svcRes:   domains.DomainsPage{},
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
			svcReq: domains.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			svcRes: domains.DomainsPage{
				Total: 1,
				Domains: []domains.Domain{{
					Name:     authDomain.Name,
					Metadata: domains.Metadata{"key": make(chan int)},
				}},
			},
			svcErr:   nil,
			response: sdk.DomainsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListDomains", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Domains(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListDomains", mock.Anything, tc.session, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableDomain(t *testing.T) {
	ds, svc, authn := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		authnErr error
		err      error
	}{
		{
			desc:     "enable domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes:   authDomain,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "enable domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "enable domain with empty domain id",
			token:    validToken,
			domainID: "",
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			err := mgsdk.EnableDomain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableDomain", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	ds, svc, authn := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		authnErr error
		err      error
	}{
		{
			desc:     "disable domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes:   authDomain,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "disable domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "disable domain with empty domain id",
			token:    validToken,
			domainID: "",
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			err := mgsdk.DisableDomain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableDomain", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestFreezeDomain(t *testing.T) {
	ds, svc, authn := setupDomains()
	defer ds.Close()

	sdkConf := sdk.Config{
		DomainsURL:     ds.URL,
		MsgContentType: contentType,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		token    string
		session  smqauthn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		authnErr error
		err      error
	}{
		{
			desc:     "freeze domain successfully",
			token:    validToken,
			domainID: sdkDomain.ID,
			svcRes:   authDomain,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "freeze domain with invalid token",
			token:    invalidToken,
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			authnErr: svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "freeze domain with empty token",
			token:    "",
			domainID: sdkDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "freeze domain with empty domain id",
			token:    validToken,
			domainID: "",
			svcRes:   domains.Domain{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("FreezeDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			err := mgsdk.FreezeDomain(tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "FreezeDomain", mock.Anything, tc.session, tc.domainID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestCreateDomainRole(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
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
	now := time.Now().UTC()
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      rReq.RoleName,
		EntityID:  domainID,
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
		roleReq         sdk.RoleReq
		svcRes          roles.RoleProvision
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "create domain role successfully",
			token:    validToken,
			domainID: domainID,
			roleReq:  rReq,
			svcRes:   roleProvision,
			svcErr:   nil,
			response: convertRoleProvision(roleProvision),
			err:      nil,
		},
		{
			desc:            "create domain role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleReq:         rReq,
			svcRes:          roles.RoleProvision{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create domain role with empty token",
			token:    "",
			domainID: domainID,
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "create domain role with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "create domain role with empty domain id",
			token:    validToken,
			domainID: "",
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "create domain role with empty role name",
			token:    validToken,
			domainID: domainID,
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
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("AddRole", mock.Anything, tc.session, tc.domainID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateDomainRole(tc.domainID, tc.roleReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AddRole", mock.Anything, tc.session, tc.domainID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListDomainRoles(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  domainID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		pageMeta        sdk.PageMetadata
		svcRes          roles.RolePage
		svcErr          error
		authenticateErr error
		response        sdk.RolesPage
		err             errors.SDKError
	}{
		{
			desc:     "list domain roles successfully",
			token:    validToken,
			domainID: domainID,
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
			desc:     "list domain roles with invalid token",
			token:    invalidToken,
			domainID: domainID,
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
			desc:     "list domain roles with empty token",
			token:    "",
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   roles.RolePage{},
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list domain roles with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
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
			desc:  "list domain roles with empty domain id",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			domainID: "",
			svcRes:   roles.RolePage{},
			svcErr:   nil,
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RetrieveAllRoles", mock.Anything, tc.session, tc.domainID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DomainRoles(tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveAllRoles", mock.Anything, tc.session, tc.domainID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewClietRole(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  domainID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "view domain role successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   role.ID,
			svcRes:   role,
			svcErr:   nil,
			response: convertRole(role),
			err:      nil,
		},
		{
			desc:            "view domain role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          role.ID,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view domain role with empty token",
			token:    "",
			domainID: domainID,
			roleID:   role.ID,
			svcRes:   roles.Role{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view domain role with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view domain role with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "view domain role with invalid role id",
			token:    validToken,
			domainID: domainID,
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
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RetrieveRole", mock.Anything, tc.session, tc.domainID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DomainRole(tc.domainID, tc.roleID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveRole", mock.Anything, tc.session, tc.domainID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateDomainRole(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	newRoleName := "newTest"
	userID := testsutil.GenerateUUID(t)
	createdAt := time.Now().UTC().Add(-time.Hour)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      newRoleName,
		EntityID:  domainID,
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
		roleID          string
		newRoleName     string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:        "update domain role successfully",
			token:       validToken,
			domainID:    domainID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      role,
			svcErr:      nil,
			response:    convertRole(role),
			err:         nil,
		},
		{
			desc:            "update domain role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			newRoleName:     newRoleName,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "update domain role with empty token",
			token:       "",
			domainID:    domainID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "update domain role with invalid domain id",
			token:       validToken,
			domainID:    testsutil.GenerateUUID(t),
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			svcErr:      svcerr.ErrAuthorization,
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:        "update domain role with empty domain id",
			token:       validToken,
			domainID:    "",
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			svcErr:      nil,
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("UpdateRoleName", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.newRoleName).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateDomainRole(tc.domainID, tc.roleID, tc.newRoleName, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateRoleName", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.newRoleName)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteDomainRole(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete domain role successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "delete domain role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "delete domain role with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete domain role with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "delete domain role with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "delete domain role with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RemoveRole", mock.Anything, tc.session, tc.domainID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.DeleteDomainRole(tc.domainID, tc.roleID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveRole", mock.Anything, tc.session, tc.domainID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddDomainRoleActions(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		actions         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add domain role actions successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			actions:  actions,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "add domain role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add domain role actions with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add domain role actions with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add domain role actions with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "add domain role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add domain role actions with empty actions",
			token:    validToken,
			domainID: domainID,
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
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleAddActions", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.actions).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddDomainRoleActions(tc.domainID, tc.roleID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddActions", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListDomainRoleActions(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "list domain role actions successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "list domain role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list domain role actions with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list domain role actions with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list domain role actions with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "list domain role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list domain role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleListActions", mock.Anything, tc.session, tc.domainID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DomainRoleActions(tc.domainID, tc.roleID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListActions", mock.Anything, tc.session, tc.domainID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveDomainRoleActions(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		actions         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove domain role actions successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			actions:  actions,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove domain role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove domain role actions with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove domain role actions with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove domain role actions with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "remove domain role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove domain role actions with empty actions",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			actions:  []string{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyEntityType), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveActions", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.actions).Return(tc.svcErr)
			err := mgsdk.RemoveDomainRoleActions(tc.domainID, tc.roleID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveActions", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllDomainRoleActions(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all domain role actions successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all domain role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all domain role actions with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all domain role actions with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all domain role actions with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "remove all domain role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all domain role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveAllActions", mock.Anything, tc.session, tc.domainID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllDomainRoleActions(tc.domainID, tc.roleID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllActions", mock.Anything, tc.session, tc.domainID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddDomainRoleMembers(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		members         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add domain role members successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			members:  members,
			svcRes:   members,
			svcErr:   nil,
			response: members,
			err:      nil,
		},
		{
			desc:            "add domain role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add domain role members with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add domain role members with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add domain role members with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "add domain role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add domain role members with empty members",
			token:    validToken,
			domainID: domainID,
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
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleAddMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.members).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddDomainRoleMembers(tc.domainID, tc.roleID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListDomainRoleMembers(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		pageMeta        sdk.PageMetadata
		svcRes          roles.MembersPage
		svcErr          error
		authenticateErr error
		response        sdk.RoleMembersPage
		err             errors.SDKError
	}{
		{
			desc:     "list domain role members successfully",
			token:    validToken,
			domainID: domainID,
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
			desc:     "list domain role members with invalid token",
			token:    invalidToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list domain role members with empty token",
			token:    "",
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list domain role members with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:  "list domain role members with empty domain id",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			domainID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "list domain role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: invalid,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list domain role members with empty role id",
			token:    validToken,
			domainID: domainID,
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
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleListMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DomainRoleMembers(tc.domainID, tc.roleID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveDomainRoleMembers(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		members         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove domain role members successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			members:  members,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove domain role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove domain role members with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove domain role members with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove domain role members with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "remove domain role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove domain role members with empty members",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			members:  []string{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleMembers), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.members).Return(tc.svcErr)
			err := mgsdk.RemoveDomainRoleMembers(tc.domainID, tc.roleID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveMembers", mock.Anything, tc.session, tc.domainID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllDomainRoleMembers(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all domain role members successfully",
			token:    validToken,
			domainID: domainID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all domain role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all domain role members with empty token",
			token:    "",
			domainID: domainID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all domain role members with invalid domain id",
			token:    validToken,
			domainID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all domain role members with empty domain id",
			token:    validToken,
			domainID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrMissingDomainID, http.StatusBadRequest),
		},
		{
			desc:     "remove all domain role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all domain role members with empty role id",
			token:    validToken,
			domainID: domainID,
			roleID:   "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRoleID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: tc.domainID + "_" + validID, UserID: validID, DomainID: tc.domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("RoleRemoveAllMembers", mock.Anything, tc.session, tc.domainID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllDomainRoleMembers(tc.domainID, tc.roleID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllMembers", mock.Anything, tc.session, tc.domainID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListAvailableDomainRoleActions(t *testing.T) {
	ts, csvc, auth := setupDomains()
	defer ts.Close()

	conf := sdk.Config{
		DomainsURL: ts.URL,
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
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "list available role actions with invalid token",
			token:           invalidToken,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list available role actions with empty token",
			token: "",
			err:   errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("ListAvailableActions", mock.Anything, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AvailableDomainRoleActions(tc.token)
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

func generateTestDomain(t *testing.T) (domains.Domain, sdk.Domain) {
	createdAt, err := time.Parse(time.RFC3339, "2024-04-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %s", err))
	ownerID := testsutil.GenerateUUID(t)
	ad := domains.Domain{
		ID:        testsutil.GenerateUUID(t),
		Name:      "test-domain",
		Metadata:  domains.Metadata(validMetadata),
		Tags:      []string{"tag1", "tag2"},
		Alias:     "test-alias",
		Status:    domains.EnabledStatus,
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
