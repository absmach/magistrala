// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/api"
	bmocks "github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	externalId  = testsutil.GenerateUUID(&testing.T{})
	externalKey = testsutil.GenerateUUID(&testing.T{})
	clientId    = testsutil.GenerateUUID(&testing.T{})
	channel1Id  = testsutil.GenerateUUID(&testing.T{})
	channel2Id  = testsutil.GenerateUUID(&testing.T{})
	clientCert  = "newcert"
	clientKey   = "newkey"
	caCert      = "newca"
	content     = "newcontent"
	bsName      = "test"
	encKey      = []byte("1234567891011121")

	bootstrapConfig = bootstrap.Config{
		ID:          clientId,
		Name:        bsName,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
		CACert:      caCert,
		ExternalID:  externalId,
		ExternalKey: externalKey,
		Content:     content,
		Status:      bootstrap.Inactive,
	}

	sdkBootstrapConfig = sdk.BootstrapConfig{
		ExternalID:  externalId,
		ExternalKey: externalKey,
		ID:          clientId,
		Name:        bsName,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
		CACert:      caCert,
		Content:     content,
		Status:      sdk.BootstrapDisabledStatus,
	}

	sdkBootstrapListRes = sdk.BootstrapConfig{
		ID:         clientId,
		ExternalID: externalId,
		Name:       bsName,
		Content:    content,
		Status:     sdk.BootstrapDisabledStatus,
	}

	sdkBootstrapCertRes = sdk.BootstrapConfig{
		ID:         clientId,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
	}

	sdkBootstrapReadRes = sdk.BootstrapConfig{
		ID:         clientId,
		Content:    content,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
	}

	readConfigResponse = struct {
		ID         string `json:"id"`
		Content    string `json:"content,omitempty"`
		ClientCert string `json:"client_cert,omitempty"`
		ClientKey  string `json:"client_key,omitempty"`
		CACert     string `json:"ca_cert,omitempty"`
	}{
		ID:         clientId,
		Content:    content,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
	}
)

var (
	errMarshalChan = errors.New("json: unsupported type: chan int")
	errJSONEOF     = errors.New("unexpected end of JSON input")
)

func setupBootstrap() (*httptest.Server, *bmocks.Service, *bmocks.ConfigReader, *authnmocks.Authentication) {
	bsvc := new(bmocks.Service)
	reader := new(bmocks.ConfigReader)
	logger := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))

	mux := api.MakeHandler(bsvc, am, reader, logger, "")

	return httptest.NewServer(mux), bsvc, reader, authn
}

func bootstrapSession() smqauthn.Session {
	return smqauthn.Session{
		DomainUserID: domainID + "_" + validID,
		UserID:       validID,
		DomainID:     domainID,
	}
}

func TestAddBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	createCfg := sdkBootstrapConfig
	createCfg.ID = ""

	svcReq := bootstrap.Config{
		ExternalID:  externalId,
		ExternalKey: externalKey,
		Name:        bsName,
		ClientCert:  clientCert,
		ClientKey:   clientKey,
		CACert:      caCert,
		Content:     content,
	}

	cases := []struct {
		desc           string
		token          string
		cfg            sdk.BootstrapConfig
		svcReq         bootstrap.Config
		svcRes         bootstrap.Config
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedID     string
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "add successfully",
			token:         validToken,
			cfg:           createCfg,
			svcReq:        svcReq,
			svcRes:        bootstrapConfig,
			expectSvcCall: true,
			expectedID:    clientId,
		},
		{
			desc:           "add with invalid token",
			token:          invalidToken,
			cfg:            createCfg,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "add with config that cannot be marshalled",
			token: validToken,
			cfg: sdk.BootstrapConfig{
				RenderContext: map[string]any{
					"broken": make(chan int),
				},
				ExternalID:  externalId,
				ExternalKey: externalKey,
			},
			expectedSDKErr: errors.NewSDKError(errMarshalChan),
		},
		{
			desc:           "add with missing required fields",
			token:          validToken,
			cfg:            sdk.BootstrapConfig{},
			expectedSDKErr: errors.NewSDKErrorWithStatus(apiutil.ErrMissingID, http.StatusBadRequest),
		},
		{
			desc:           "add with service failure",
			token:          validToken,
			cfg:            createCfg,
			svcReq:         svcReq,
			svcRes:         bootstrap.Config{},
			svcErr:         svcerr.ErrNotFound,
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}

			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("Add", mock.Anything, session, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			}

			resp, err := mgsdk.AddBootstrap(context.Background(), tc.cfg, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedID, resp)
			if tc.expectSvcCall {
				svcCall.Unset()
			}
			authCall.Unset()
		})
	}
}

func TestListBootstraps(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		pm             sdk.PageMetadata
		svcResp        bootstrap.ConfigsPage
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedResp   sdk.BootstrapPage
		expectedSDKErr errors.SDKError
	}{
		{
			desc:  "list successfully",
			token: validToken,
			pm: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcResp: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Configs: []bootstrap.Config{bootstrapConfig},
			},
			expectSvcCall: true,
			expectedResp: sdk.BootstrapPage{
				PageRes: sdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Configs: []sdk.BootstrapConfig{sdkBootstrapListRes},
			},
		},
		{
			desc:  "list with invalid token",
			token: invalidToken,
			pm: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list with invalid query params",
			token: validToken,
			pm: sdk.PageMetadata{
				Metadata: map[string]any{
					"test": make(chan int),
				},
			},
			expectedSDKErr: errors.NewSDKError(errMarshalChan),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.expectedSDKErr == nil || tc.authErr != nil {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("List", mock.Anything, session, mock.Anything, tc.pm.Offset, tc.pm.Limit).Return(tc.svcResp, tc.svcErr)
			}

			resp, err := mgsdk.Bootstraps(context.Background(), tc.pm, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedResp, resp)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestWhitelist(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		clientID       string
		status         sdk.BootstrapStatus
		method         string
		svcResp        bootstrap.Config
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "enable bootstrap successfully",
			token:         validToken,
			clientID:      clientId,
			status:        sdk.BootstrapEnabledStatus,
			method:        "EnableConfig",
			svcResp:       bootstrap.Config{ID: clientId, Status: bootstrap.Active},
			expectSvcCall: true,
		},
		{
			desc:          "disable bootstrap successfully",
			token:         validToken,
			clientID:      clientId,
			status:        sdk.BootstrapDisabledStatus,
			method:        "DisableConfig",
			svcResp:       bootstrap.Config{ID: clientId, Status: bootstrap.Inactive},
			expectSvcCall: true,
		},
		{
			desc:           "whitelist with invalid token",
			token:          invalidToken,
			clientID:       clientId,
			status:         sdk.BootstrapEnabledStatus,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "whitelist with invalid status",
			token:          validToken,
			clientID:       clientId,
			status:         sdk.BootstrapStatus("invalid"),
			expectedSDKErr: errors.NewSDKErrorWithStatus(errors.New("invalid bootstrap status"), http.StatusBadRequest),
		},
		{
			desc:           "whitelist with empty client id",
			token:          validToken,
			clientID:       "",
			status:         sdk.BootstrapEnabledStatus,
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.clientID != "" && (tc.status == sdk.BootstrapDisabledStatus || tc.status == sdk.BootstrapEnabledStatus) {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On(tc.method, mock.Anything, session, tc.clientID).Return(tc.svcResp, tc.svcErr)
			}

			err := mgsdk.Whitelist(context.Background(), tc.clientID, tc.status, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestViewBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		id             string
		svcResp        bootstrap.Config
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedResp   sdk.BootstrapConfig
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "view successfully",
			token:         validToken,
			id:            clientId,
			svcResp:       bootstrapConfig,
			expectSvcCall: true,
			expectedResp:  sdkBootstrapListRes,
		},
		{
			desc:           "view with invalid token",
			token:          invalidToken,
			id:             clientId,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "view with non-existent client id",
			token:          validToken,
			id:             invalid,
			svcResp:        bootstrap.Config{},
			svcErr:         svcerr.ErrNotFound,
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:           "view with empty client id",
			token:          validToken,
			id:             "",
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.id != "" {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("View", mock.Anything, session, tc.id).Return(tc.svcResp, tc.svcErr)
			}

			resp, err := mgsdk.ViewBootstrap(context.Background(), tc.id, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedResp, resp)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestUpdateBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		cfg            sdk.BootstrapConfig
		svcReq         bootstrap.Config
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedSDKErr errors.SDKError
	}{
		{
			desc:  "update successfully",
			token: validToken,
			cfg:   sdkBootstrapConfig,
			svcReq: bootstrap.Config{
				ID:      clientId,
				Name:    bsName,
				Content: content,
			},
			expectSvcCall: true,
		},
		{
			desc:           "update with invalid token",
			token:          invalidToken,
			cfg:            sdkBootstrapConfig,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "update with empty client id",
			token:          validToken,
			cfg:            sdk.BootstrapConfig{},
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:  "update with config that cannot be marshalled",
			token: validToken,
			cfg: sdk.BootstrapConfig{
				ID: clientId,
				RenderContext: map[string]any{
					"broken": make(chan int),
				},
			},
			expectedSDKErr: errors.NewSDKError(errMarshalChan),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.cfg.ID != "" {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("Update", mock.Anything, session, tc.svcReq).Return(tc.svcErr)
			}

			err := mgsdk.UpdateBootstrap(context.Background(), tc.cfg, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestUpdateBootstrapCerts(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		id             string
		cert           string
		key            string
		ca             string
		svcResp        bootstrap.Config
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedResp   sdk.BootstrapConfig
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "update certs successfully",
			token:         validToken,
			id:            clientId,
			cert:          clientCert,
			key:           clientKey,
			ca:            caCert,
			svcResp:       bootstrapConfig,
			expectSvcCall: true,
			expectedResp:  sdkBootstrapCertRes,
		},
		{
			desc:           "update certs with invalid token",
			token:          invalidToken,
			id:             clientId,
			cert:           clientCert,
			key:            clientKey,
			ca:             caCert,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "update certs with empty id",
			token:          validToken,
			id:             "",
			cert:           clientCert,
			key:            clientKey,
			ca:             caCert,
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.id != "" {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("UpdateCert", mock.Anything, session, tc.id, tc.cert, tc.key, tc.ca).Return(tc.svcResp, tc.svcErr)
			}

			resp, err := mgsdk.UpdateBootstrapCerts(context.Background(), tc.id, tc.cert, tc.key, tc.ca, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedResp, resp)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestUpdateBootstrapConnection(t *testing.T) {
	mgsdk := sdk.NewSDK(sdk.Config{})

	err := mgsdk.UpdateBootstrapConnection(context.Background(), clientId, []string{channel1Id, channel2Id}, domainID, validToken)
	assert.Equal(t, errors.NewSDKError(errors.New("bootstrap connection updates are no longer supported")), err)

	err = mgsdk.UpdateBootstrapConnection(context.Background(), "", []string{channel1Id}, domainID, validToken)
	assert.Equal(t, errors.NewSDKError(apiutil.ErrMissingID), err)
}

func TestRemoveBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		token          string
		id             string
		svcErr         error
		authErr        error
		expectSvcCall  bool
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "remove successfully",
			token:         validToken,
			id:            clientId,
			expectSvcCall: true,
		},
		{
			desc:           "remove with invalid token",
			token:          invalidToken,
			id:             clientId,
			authErr:        svcerr.ErrAuthentication,
			expectedSDKErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "remove with empty id",
			token:          validToken,
			id:             "",
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authCall *mock.Call
			session := smqauthn.Session{}
			if tc.token == validToken {
				session = bootstrapSession()
			}
			if tc.id != "" {
				authCall = auth.On("Authenticate", mock.Anything, tc.token).Return(session, tc.authErr)
			}

			var svcCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("Remove", mock.Anything, session, tc.id).Return(tc.svcErr)
			}

			err := mgsdk.RemoveBootstrap(context.Background(), tc.id, domainID, tc.token)

			assert.Equal(t, tc.expectedSDKErr, err)
			if svcCall != nil {
				svcCall.Unset()
			}
			if authCall != nil {
				authCall.Unset()
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	bs, bsvc, reader, _ := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	cases := []struct {
		desc           string
		externalID     string
		externalKey    string
		svcResp        bootstrap.Config
		svcErr         error
		readerResp     any
		readerErr      error
		expectSvcCall  bool
		expectedResp   sdk.BootstrapConfig
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "bootstrap successfully",
			externalID:    externalId,
			externalKey:   externalKey,
			svcResp:       bootstrapConfig,
			readerResp:    readConfigResponse,
			expectSvcCall: true,
			expectedResp:  sdkBootstrapReadRes,
		},
		{
			desc:           "bootstrap with reader error",
			externalID:     externalId,
			externalKey:    externalKey,
			svcResp:        bootstrapConfig,
			readerErr:      errJSONEOF,
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKErrorWithStatus(errJSONEOF, http.StatusInternalServerError),
		},
		{
			desc:           "bootstrap with malformed response",
			externalID:     externalId,
			externalKey:    externalKey,
			svcResp:        bootstrapConfig,
			readerResp:     []byte{0},
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKError(errors.New("json: cannot unmarshal string into Go value of type sdk.BootstrapConfig")),
		},
		{
			desc:           "bootstrap with empty id",
			externalID:     "",
			externalKey:    externalKey,
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:           "bootstrap with empty key",
			externalID:     externalId,
			externalKey:    "",
			expectedSDKErr: errors.NewSDKErrorWithStatus(apiutil.ErrBearerKey, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var svcCall *mock.Call
			var readerCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("Bootstrap", mock.Anything, tc.externalKey, tc.externalID, false).Return(tc.svcResp, tc.svcErr)
				readerCall = reader.On("ReadConfig", tc.svcResp, false).Return(tc.readerResp, tc.readerErr)
			}

			resp, err := mgsdk.Bootstrap(context.Background(), tc.externalID, tc.externalKey)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedResp, resp)
			if svcCall != nil {
				svcCall.Unset()
			}
			if readerCall != nil {
				readerCall.Unset()
			}
		})
	}
}

func TestBootstrapSecure(t *testing.T) {
	bs, bsvc, reader, _ := setupBootstrap()
	defer bs.Close()

	mgsdk := sdk.NewSDK(sdk.Config{BootstrapURL: bs.URL})

	body, err := json.Marshal(readConfigResponse)
	assert.Nil(t, err, fmt.Sprintf("Marshalling bootstrap response expected to succeed: %s.\n", err))

	encResponse, err := encrypt(body, encKey)
	assert.Nil(t, err, fmt.Sprintf("Encrypting bootstrap response expected to succeed: %s.\n", err))

	cases := []struct {
		desc           string
		externalID     string
		externalKey    string
		cryptoKey      string
		svcResp        bootstrap.Config
		svcErr         error
		readerResp     []byte
		readerErr      error
		expectSvcCall  bool
		expectedResp   sdk.BootstrapConfig
		expectedSDKErr errors.SDKError
	}{
		{
			desc:          "secure bootstrap successfully",
			externalID:    externalId,
			externalKey:   externalKey,
			cryptoKey:     string(encKey),
			svcResp:       bootstrapConfig,
			readerResp:    encResponse,
			expectSvcCall: true,
			expectedResp:  sdkBootstrapReadRes,
		},
		{
			desc:           "secure bootstrap with invalid crypto key",
			externalID:     externalId,
			externalKey:    externalKey,
			cryptoKey:      invalid,
			expectedSDKErr: errors.NewSDKError(errors.New("crypto/aes: invalid key size 7")),
		},
		{
			desc:           "secure bootstrap with reader error",
			externalID:     externalId,
			externalKey:    externalKey,
			cryptoKey:      string(encKey),
			svcResp:        bootstrapConfig,
			readerErr:      errJSONEOF,
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKErrorWithStatus(errJSONEOF, http.StatusInternalServerError),
		},
		{
			desc:           "secure bootstrap with malformed response",
			externalID:     externalId,
			externalKey:    externalKey,
			cryptoKey:      string(encKey),
			svcResp:        bootstrapConfig,
			readerResp:     []byte{0},
			expectSvcCall:  true,
			expectedSDKErr: errors.NewSDKError(errJSONEOF),
		},
		{
			desc:           "secure bootstrap with empty id",
			externalID:     "",
			externalKey:    externalKey,
			cryptoKey:      string(encKey),
			expectedSDKErr: errors.NewSDKError(apiutil.ErrMissingID),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var svcCall *mock.Call
			var readerCall *mock.Call
			if tc.expectSvcCall {
				svcCall = bsvc.On("Bootstrap", mock.Anything, mock.Anything, tc.externalID, true).Return(tc.svcResp, tc.svcErr)
				readerCall = reader.On("ReadConfig", tc.svcResp, true).Return(tc.readerResp, tc.readerErr)
			}

			resp, err := mgsdk.BootstrapSecure(context.Background(), tc.externalID, tc.externalKey, tc.cryptoKey)

			assert.Equal(t, tc.expectedSDKErr, err)
			assert.Equal(t, tc.expectedResp, resp)
			if svcCall != nil {
				svcCall.Unset()
			}
			if readerCall != nil {
				readerCall.Unset()
			}
		})
	}
}

func encrypt(in, encKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return ciphertext, nil
}
