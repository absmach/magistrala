// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/certs"
	httpapi "github.com/absmach/magistrala/certs/api"
	"github.com/absmach/magistrala/certs/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"

var thingID = "1"

var c = certs.Cert{
	OwnerID:        "",
	ThingID:        thingID,
	ClientCert:     "",
	IssuingCA:      "",
	CAChain:        []string{},
	ClientKey:      "",
	PrivateKeyType: "",
	Serial:         "",
	Expire:         time.Time{},
}

func setupCerts() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)
	logger := mglog.NewMock()
	mux := httpapi.MakeHandler(svc, logger, instanceID)

	return httptest.NewServer(mux), svc
}

func TestIssueCert(t *testing.T) {
	ts, svc := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		thingID  string
		duration string
		token    string
		cRes     certs.Cert
		err      errors.SDKError
		svcerr   error
	}{
		{
			desc:     "create new cert with thing id and duration",
			thingID:  thingID,
			duration: "10h",
			token:    validToken,
			cRes:     c,
		},
		{
			desc:     "create new cert with empty thing id and duration",
			thingID:  "",
			duration: "10h",
			token:    validToken,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrMissingID),
		},
		{
			desc:     "create new cert with invalid thing id and duration",
			thingID:  "ah",
			duration: "10h",
			token:    validToken,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, certs.ErrFailedCertCreation), http.StatusBadRequest),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrValidation),
		},
		{
			desc:     "create new cert with thing id and empty duration",
			thingID:  thingID,
			duration: "",
			token:    exampleUser1,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingCertData), http.StatusBadRequest),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrMissingCertData),
		},
		{
			desc:     "create new cert with thing id and malformed duration",
			thingID:  thingID,
			duration: "10g",
			token:    exampleUser1,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidCertData), http.StatusBadRequest),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, apiutil.ErrInvalidCertData),
		},
		{
			desc:     "create new cert with empty token",
			thingID:  thingID,
			duration: "10h",
			token:    "",
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, svcerr.ErrAuthentication),
		},
		{
			desc:     "create new cert with invalid token",
			thingID:  thingID,
			duration: "10h",
			token:    authmocks.InvalidValue,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, certs.ErrFailedCertCreation), http.StatusUnauthorized),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, svcerr.ErrAuthentication),
		},
		{
			desc:     "create new empty cert",
			thingID:  "",
			duration: "",
			token:    validToken,
			cRes:     c,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
			svcerr:   errors.Wrap(certs.ErrFailedCertCreation, certs.ErrFailedCertCreation),
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("IssueCert", mock.Anything, tc.token, tc.thingID, tc.duration).Return(tc.cRes, tc.svcerr)

		_, err := mgsdk.IssueCert(tc.thingID, tc.duration, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestViewCert(t *testing.T) {
	ts, svc := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc   string
		certID string
		token  string
		err    errors.SDKError
		svcerr error
		cRes   certs.Cert
	}{
		{
			desc:   "get existing cert",
			certID: validID,
			token:  token,
			cRes:   c,
			err:    nil,
			svcerr: nil,
		},
		{
			desc:   "get non-existent cert",
			certID: "43",
			token:  token,
			cRes:   c,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, svcerr.ErrNotFound), http.StatusNotFound),
			svcerr: errors.Wrap(svcerr.ErrNotFound, repoerr.ErrNotFound),
		},
		{
			desc:   "get cert with invalid token",
			certID: validID,
			token:  "",
			cRes:   c,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			svcerr: errors.Wrap(svcerr.ErrAuthentication, apiutil.ErrBearerToken),
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("ViewCert", mock.Anything, tc.token, tc.certID).Return(tc.cRes, tc.svcerr)

		cert, err := mgsdk.ViewCert(tc.certID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, cert, fmt.Sprintf("%s: got empty cert", tc.desc))
		}
		svcCall.Unset()
	}
}

func TestViewCertByThing(t *testing.T) {
	ts, svc := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc    string
		thingID string
		token   string
		page    certs.Page
		err     errors.SDKError
		viewerr errors.SDKError
		svcerr  error
	}{
		{
			desc:    "get existing cert",
			thingID: thingID,
			token:   token,
			page:    certs.Page{Certs: []certs.Cert{c}},
		},
		{
			desc:    "get non-existent cert",
			thingID: "43",
			token:   token,
			page:    certs.Page{Certs: []certs.Cert{}},
			err:     errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, repoerr.ErrNotFound), http.StatusNotFound),
			svcerr:  errors.Wrap(svcerr.ErrNotFound, repoerr.ErrNotFound),
			viewerr: errors.NewSDKError(svcerr.ErrViewEntity),
		},
		{
			desc:    "get cert with invalid token",
			thingID: thingID,
			token:   "",
			page:    certs.Page{Certs: []certs.Cert{}},
			err:     errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			svcerr:  errors.Wrap(svcerr.ErrAuthentication, apiutil.ErrBearerToken),
		},
	}
	for _, tc := range cases {
		svcCall := svc.On("ListSerials", mock.Anything, tc.token, tc.thingID, tc.page.Offset, mock.Anything).Return(tc.page, tc.svcerr)
		svcCall1 := svc.On("ViewCertByThing", mock.Anything, tc.thingID, tc.token).Return(tc.page, tc.viewerr)

		cert, err := mgsdk.ViewCertByThing(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, cert, fmt.Sprintf("%s: got empty cert", tc.desc))
		}
		svcCall.Unset()
		svcCall1.Unset()
	}
}

func TestRevokeCert(t *testing.T) {
	ts, svc := setupCerts()
	defer ts.Close()

	sdkConf := sdk.Config{
		CertsURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc        string
		page        certs.Page
		thingID     string
		token       string
		svcResponse certs.Revoke
		err         errors.SDKError
		svcerr      error
	}{
		{
			desc:        "revoke cert with invalid token",
			thingID:     thingID,
			token:       authmocks.InvalidValue,
			svcResponse: certs.Revoke{RevocationTime: time.Now()},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			svcerr:      errors.Wrap(svcerr.ErrAuthentication, svcerr.ErrAuthentication),
		},
		{
			desc:        "revoke non-existing cert",
			thingID:     "2",
			token:       token,
			svcResponse: certs.Revoke{RevocationTime: time.Now()},
			err:         errors.NewSDKErrorWithStatus(certs.ErrFailedCertRevocation, http.StatusNotFound),
			svcerr:      errors.Wrap(certs.ErrFailedCertRevocation, svcerr.ErrNotFound),
		},
		{
			desc:        "revoke cert with empty token",
			thingID:     thingID,
			token:       "",
			svcResponse: certs.Revoke{RevocationTime: time.Now()},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			svcerr:      errors.Wrap(svcerr.ErrAuthentication, apiutil.ErrBearerToken),
		},
		{
			desc:        "revoke existing cert",
			thingID:     thingID,
			token:       token,
			svcResponse: certs.Revoke{RevocationTime: time.Now()},
		},
		{
			desc:        "revoke deleted cert",
			thingID:     thingID,
			token:       token,
			svcResponse: certs.Revoke{RevocationTime: time.Now()},
			err:         errors.NewSDKErrorWithStatus(certs.ErrFailedToRemoveCertFromDB, http.StatusNotFound),
			svcerr:      errors.Wrap(certs.ErrFailedToRemoveCertFromDB, svcerr.ErrNotFound),
		},
	}
	for _, tc := range cases {
		svcCall := svc.On("RevokeCert", mock.Anything, tc.token, tc.thingID).Return(tc.svcResponse, tc.svcerr)

		response, err := mgsdk.RevokeCert(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, response, fmt.Sprintf("%s: got empty revocation time", tc.desc))
		}
		svcCall.Unset()
	}
}
