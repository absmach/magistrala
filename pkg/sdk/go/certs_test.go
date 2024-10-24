// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/certs"
	httpapi "github.com/absmach/magistrala/certs/api"
	"github.com/absmach/magistrala/certs/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"

var (
	valid                = "valid"
	clientID             = testsutil.GenerateUUID(&testing.T{})
	OwnerID              = testsutil.GenerateUUID(&testing.T{})
	serial               = testsutil.GenerateUUID(&testing.T{})
	ttl                  = "10h"
	cert, sdkCert        = generateTestCerts(&testing.T{})
	defOffset     uint64 = 0
	defLimit      uint64 = 10
	defRevoke            = "false"
)

func generateTestCerts(t *testing.T) (certs.Cert, sdk.Cert) {
	expirationTime, err := time.Parse(time.RFC3339, "2032-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("failed to parse expiration time: %v", err))
	c := certs.Cert{
		ClientID:     clientID,
		SerialNumber: serial,
		ExpiryTime:   expirationTime,
		Certificate:  valid,
	}
	sc := sdk.Cert{
		ClientID:     clientID,
		SerialNumber: serial,
		Key:          valid,
		Certificate:  valid,
		ExpiryTime:   expirationTime,
	}

	return c, sc
}

func setupCerts() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	logger := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	mux := httpapi.MakeHandler(svc, authn, logger, instanceID)

	return httptest.NewServer(mux), svc, authn
}

func TestIssueCert(t *testing.T) {
	ts, svc, auth := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc            string
		clientID        string
		duration        string
		domainID        string
		token           string
		session         mgauthn.Session
		authenticateErr error
		svcRes          certs.Cert
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "create new cert with client id and duration",
			clientID: clientID,
			duration: ttl,
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{SerialNumber: serial},
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "create new cert with empty client id and duration",
			clientID: "",
			duration: ttl,
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrMissingID),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "create new cert with invalid client id and duration",
			clientID: invalid,
			duration: ttl,
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrValidation),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, certs.ErrFailedCertCreation), http.StatusBadRequest),
		},
		{
			desc:     "create new cert with client id and empty duration",
			clientID: clientID,
			duration: "",
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrMissingCertData),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingCertData), http.StatusBadRequest),
		},
		{
			desc:     "create new cert with client id and malformed duration",
			clientID: clientID,
			duration: invalid,
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrInvalidCertData),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidCertData), http.StatusBadRequest),
		},
		{
			desc:     "create new cert with empty token",
			clientID: clientID,
			duration: ttl,
			domainID: validID,
			token:    "",
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, svcerr.ErrAuthentication),
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:            "create new cert with invalid token",
			clientID:        clientID,
			domainID:        domainID,
			duration:        ttl,
			token:           invalidToken,
			svcRes:          certs.Cert{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create new empty cert",
			clientID: "",
			duration: "",
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(certs.ErrFailedCertCreation, certs.ErrFailedCertCreation),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("IssueCert", mock.Anything, tc.domainID, tc.token, tc.clientID, tc.duration).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.IssueCert(tc.clientID, tc.duration, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				assert.Equal(t, tc.svcRes.SerialNumber, resp.SerialNumber)
				ok := svcCall.Parent.AssertCalled(t, "IssueCert", mock.Anything, tc.domainID, tc.token, tc.clientID, tc.duration)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewCert(t *testing.T) {
	ts, svc, auth := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	viewCertRes := sdkCert
	viewCertRes.Key = ""

	cases := []struct {
		desc            string
		certID          string
		domainID        string
		token           string
		session         mgauthn.Session
		authenticateErr error
		svcRes          certs.Cert
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "view existing cert",
			certID:   validID,
			domainID: validID,
			token:    validToken,
			svcRes:   cert,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "view non-existent cert",
			certID:   invalid,
			domainID: validID,
			token:    validToken,
			svcRes:   certs.Cert{},
			svcErr:   errors.Wrap(svcerr.ErrNotFound, repoerr.ErrNotFound),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, svcerr.ErrNotFound), http.StatusNotFound),
		},
		{
			desc:            "view cert with invalid token",
			certID:          validID,
			domainID:        domainID,
			token:           invalidToken,
			svcRes:          certs.Cert{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view cert with empty token",
			certID:   validID,
			domainID: domainID,
			token:    "",
			svcRes:   certs.Cert{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ViewCert", mock.Anything, tc.certID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ViewCert(tc.certID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, viewCertRes, resp)
				ok := svcCall.Parent.AssertCalled(t, "ViewCert", mock.Anything, tc.certID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewCertByClient(t *testing.T) {
	ts, svc, auth := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	viewCertClientRes := sdk.CertSerials{
		Certs: []sdk.Cert{{
			SerialNumber: serial,
		}},
	}
	cases := []struct {
		desc            string
		clientID        string
		domainID        string
		token           string
		session         mgauthn.Session
		authenticateErr error
		svcRes          certs.CertPage
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "view existing cert",
			clientID: clientID,
			domainID: domainID,
			token:    validToken,
			svcRes:   certs.CertPage{Certificates: []certs.Cert{{SerialNumber: serial}}},
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "view non-existent cert",
			clientID: invalid,
			domainID: domainID,
			token:    validToken,
			svcRes:   certs.CertPage{Certificates: []certs.Cert{}},
			svcErr:   errors.Wrap(svcerr.ErrNotFound, repoerr.ErrNotFound),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, svcerr.ErrNotFound), http.StatusNotFound),
		},
		{
			desc:            "view cert with invalid token",
			clientID:        clientID,
			domainID:        domainID,
			token:           invalidToken,
			svcRes:          certs.CertPage{Certificates: []certs.Cert{}},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view cert with empty token",
			clientID: clientID,
			domainID: domainID,
			token:    "",
			svcRes:   certs.CertPage{Certificates: []certs.Cert{}},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view cert with empty client id",
			clientID: "",
			domainID: domainID,
			token:    validToken,
			svcRes:   certs.CertPage{Certificates: []certs.Cert{}},
			svcErr:   nil,
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ListSerials", mock.Anything, tc.clientID, certs.PageMetadata{Revoked: defRevoke, Offset: defOffset, Limit: defLimit}).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ViewCertByClient(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				assert.Equal(t, viewCertClientRes, resp)
				ok := svcCall.Parent.AssertCalled(t, "ListSerials", mock.Anything, tc.clientID, certs.PageMetadata{Revoked: defRevoke, Offset: defOffset, Limit: defLimit})
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRevokeCert(t *testing.T) {
	ts, svc, auth := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc            string
		clientID        string
		domainID        string
		token           string
		session         mgauthn.Session
		svcResp         certs.Revoke
		authenticateErr error
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "revoke cert successfully",
			clientID: clientID,
			domainID: validID,
			token:    validToken,
			svcResp:  certs.Revoke{RevocationTime: time.Now()},
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "revoke cert with invalid token",
			clientID:        clientID,
			domainID:        validID,
			token:           invalidToken,
			svcResp:         certs.Revoke{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "revoke non-existing cert",
			clientID: invalid,
			domainID: validID,
			token:    validToken,
			svcResp:  certs.Revoke{},
			svcErr:   errors.Wrap(certs.ErrFailedCertRevocation, svcerr.ErrNotFound),
			err:      errors.NewSDKErrorWithStatus(certs.ErrFailedCertRevocation, http.StatusNotFound),
		},
		{
			desc:     "revoke cert with empty token",
			clientID: clientID,
			domainID: validID,
			token:    "",
			svcResp:  certs.Revoke{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "revoke deleted cert",
			clientID: clientID,
			domainID: validID,
			token:    validToken,
			svcResp:  certs.Revoke{},
			svcErr:   errors.Wrap(certs.ErrFailedToRemoveCertFromDB, svcerr.ErrNotFound),
			err:      errors.NewSDKErrorWithStatus(certs.ErrFailedToRemoveCertFromDB, http.StatusNotFound),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == valid {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("RevokeCert", mock.Anything, tc.domainID, tc.token, tc.clientID).Return(tc.svcResp, tc.svcErr)
			resp, err := mgsdk.RevokeCert(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.NotEmpty(t, resp)
				ok := svcCall.Parent.AssertCalled(t, "RevokeCert", mock.Anything, tc.domainID, tc.token, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}
