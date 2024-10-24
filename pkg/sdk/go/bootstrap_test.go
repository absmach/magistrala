// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/api"
	bmocks "github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	externalId      = testsutil.GenerateUUID(&testing.T{})
	externalKey     = testsutil.GenerateUUID(&testing.T{})
	clientId        = testsutil.GenerateUUID(&testing.T{})
	clientSecret    = testsutil.GenerateUUID(&testing.T{})
	channel1Id      = testsutil.GenerateUUID(&testing.T{})
	channel2Id      = testsutil.GenerateUUID(&testing.T{})
	clientCert      = "newcert"
	clientKey       = "newkey"
	caCert          = "newca"
	content         = "newcontent"
	state           = 1
	bsName          = "test"
	encKey          = []byte("1234567891011121")
	bootstrapConfig = bootstrap.Config{
		ClientID:   clientId,
		Name:       "test",
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
		Channels: []bootstrap.Channel{
			{
				ID: channel1Id,
			},
			{
				ID: channel2Id,
			},
		},
		ExternalID:  externalId,
		ExternalKey: externalKey,
		Content:     content,
		State:       bootstrap.Inactive,
	}
	sdkBootstrapConfig = sdk.BootstrapConfig{
		Channels:     []string{channel1Id, channel2Id},
		ExternalID:   externalId,
		ExternalKey:  externalKey,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Name:         bsName,
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		CACert:       caCert,
		Content:      content,
		State:        state,
	}
	sdkBootsrapConfigRes = sdk.BootstrapConfig{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Channels: []sdk.Channel{
			{
				ID: channel1Id,
			},
			{
				ID: channel2Id,
			},
		},
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
	}
	readConfigResponse = struct {
		ClientID     string             `json:"client_id"`
		ClientSecret string             `json:"client_secret"`
		Channels     []readerChannelRes `json:"channels"`
		Content      string             `json:"content,omitempty"`
		ClientCert   string             `json:"client_cert,omitempty"`
		ClientKey    string             `json:"client_key,omitempty"`
		CACert       string             `json:"ca_cert,omitempty"`
	}{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Channels: []readerChannelRes{
			{
				ID: channel1Id,
			},
			{
				ID: channel2Id,
			},
		},
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     caCert,
	}
)

var (
	errMarshalChan = errors.New("json: unsupported type: chan int")
	errJsonEOF     = errors.New("unexpected end of JSON input")
)

type readerChannelRes struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

func setupBootstrap() (*httptest.Server, *bmocks.Service, *bmocks.ConfigReader, *authnmocks.Authentication) {
	bsvc := new(bmocks.Service)
	reader := new(bmocks.ConfigReader)
	logger := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	mux := api.MakeHandler(bsvc, authn, reader, logger, "")

	return httptest.NewServer(mux), bsvc, reader, authn
}

func TestAddBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	neID := sdkBootstrapConfig
	neID.ClientID = "non-existent"

	neReqId := bootstrapConfig
	neReqId.ClientID = "non-existent"

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		cfg             sdk.BootstrapConfig
		svcReq          bootstrap.Config
		svcRes          bootstrap.Config
		svcErr          error
		authenticateErr error
		response        string
		err             errors.SDKError
	}{
		{
			desc:     "add successfully",
			domainID: domainID,
			token:    validToken,
			cfg:      sdkBootstrapConfig,
			svcReq:   bootstrapConfig,
			svcRes:   bootstrapConfig,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "add with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			cfg:             sdkBootstrapConfig,
			svcReq:          bootstrapConfig,
			svcRes:          bootstrap.Config{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add with config that cannot be marshalled",
			domainID: domainID,
			token:    validToken,
			cfg: sdk.BootstrapConfig{
				Channels: map[string]interface{}{
					"channel1": make(chan int),
				},
				ExternalID:   externalId,
				ExternalKey:  externalKey,
				ClientID:     clientId,
				ClientSecret: clientSecret,
				Name:         bsName,
				ClientCert:   clientCert,
				ClientKey:    clientKey,
				CACert:       caCert,
				Content:      content,
			},
			svcReq: bootstrap.Config{},
			svcRes: bootstrap.Config{},
			svcErr: nil,
			err:    errors.NewSDKError(errMarshalChan),
		},
		{
			desc:     "add an existing config",
			domainID: domainID,
			token:    validToken,
			cfg:      sdkBootstrapConfig,
			svcReq:   bootstrapConfig,
			svcRes:   bootstrap.Config{},
			svcErr:   svcerr.ErrConflict,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrConflict, http.StatusConflict),
		},
		{
			desc:     "add empty config",
			domainID: domainID,
			token:    validToken,
			cfg:      sdk.BootstrapConfig{},
			svcReq:   bootstrap.Config{},
			svcRes:   bootstrap.Config{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add with non-existent client Id",
			domainID: domainID,
			token:    validToken,
			cfg:      neID,
			svcReq:   neReqId,
			svcRes:   bootstrap.Config{},
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := bsvc.On("Add", mock.Anything, tc.session, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddBootstrap(tc.cfg, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, bootstrapConfig.ClientID, resp)
				ok := svcCall.Parent.AssertCalled(t, "Add", mock.Anything, tc.session, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListBootstraps(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	configRes := sdk.BootstrapConfig{
		Channels: []sdk.Channel{
			{
				ID: channel1Id,
			},
			{
				ID: channel2Id,
			},
		},
		ClientID:    clientId,
		Name:        bsName,
		ExternalID:  externalId,
		ExternalKey: externalKey,
		Content:     content,
	}
	unmarshalableConfig := bootstrapConfig
	unmarshalableConfig.Channels = []bootstrap.Channel{
		{
			ID: channel1Id,
			Metadata: map[string]interface{}{
				"test": make(chan int),
			},
		},
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		pageMeta        sdk.PageMetadata
		svcResp         bootstrap.ConfigsPage
		svcErr          error
		authenticateErr error
		response        sdk.BootstrapPage
		err             errors.SDKError
	}{
		{
			desc:     "list successfully",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcResp: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Configs: []bootstrap.Config{bootstrapConfig},
			},
			response: sdk.BootstrapPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Configs: []sdk.BootstrapConfig{configRes},
			},
			err: nil,
		},
		{
			desc:     "list with invalid token",
			domainID: domainID,
			token:    invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcResp:         bootstrap.ConfigsPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.BootstrapPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list with empty token",
			domainID: domainID,
			token:    "",
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcResp:  bootstrap.ConfigsPage{},
			svcErr:   nil,
			response: sdk.BootstrapPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list with invalid query params",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 1,
				Limit:  10,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcResp:  bootstrap.ConfigsPage{},
			svcErr:   nil,
			response: sdk.BootstrapPage{},
			err:      errors.NewSDKError(errMarshalChan),
		},
		{
			desc:     "list with response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcResp: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Configs: []bootstrap.Config{unmarshalableConfig},
			},
			svcErr:   nil,
			response: sdk.BootstrapPage{},
			err:      errors.NewSDKError(errJsonEOF),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := bsvc.On("List", mock.Anything, tc.session, mock.Anything, tc.pageMeta.Offset, tc.pageMeta.Limit).Return(tc.svcResp, tc.svcErr)
			resp, err := mgsdk.Bootstraps(tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if err == nil {
				ok := svcCall.Parent.AssertCalled(t, "List", mock.Anything, tc.session, mock.Anything, tc.pageMeta.Offset, tc.pageMeta.Limit)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestWhiteList(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	active := 1
	inactive := 0

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		clientID        string
		state           int
		svcReq          bootstrap.State
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "whitelist to active state successfully",
			domainID: domainID,
			token:    validToken,
			clientID: clientId,
			state:    active,
			svcReq:   bootstrap.Active,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "whitelist to inactive state successfully",
			domainID: domainID,
			token:    validToken,
			clientID: clientId,
			state:    inactive,
			svcReq:   bootstrap.Inactive,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "whitelist with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        clientId,
			state:           active,
			svcReq:          bootstrap.Active,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "whitelist with empty token",
			domainID: domainID,
			token:    "",
			clientID: clientId,
			state:    active,
			svcReq:   bootstrap.Active,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "whitelist with invalid state",
			domainID: domainID,
			token:    validToken,
			clientID: clientId,
			state:    -1,
			svcReq:   bootstrap.Active,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBootstrapState), http.StatusBadRequest),
		},
		{
			desc:     "whitelist with empty client Id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			state:    1,
			svcReq:   bootstrap.Active,
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
			svcCall := bsvc.On("ChangeState", mock.Anything, tc.session, tc.token, tc.clientID, tc.svcReq).Return(tc.svcErr)
			err := mgsdk.Whitelist(tc.clientID, tc.state, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ChangeState", mock.Anything, tc.session, tc.token, tc.clientID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	viewBoostrapRes := sdk.BootstrapConfig{
		ClientID:    clientId,
		Channels:    sdkBootsrapConfigRes.Channels,
		ExternalID:  externalId,
		ExternalKey: externalKey,
		Name:        bsName,
		Content:     content,
		State:       0,
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		id              string
		svcResp         bootstrap.Config
		svcErr          error
		authenticateErr error
		response        sdk.BootstrapConfig
		err             errors.SDKError
	}{
		{
			desc:     "view successfully",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			svcResp:  bootstrapConfig,
			svcErr:   nil,
			response: viewBoostrapRes,
			err:      nil,
		},
		{
			desc:            "view with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			id:              clientId,
			svcResp:         bootstrap.Config{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.BootstrapConfig{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view with empty token",
			domainID: domainID,
			token:    "",
			id:       clientId,
			svcResp:  bootstrap.Config{},
			svcErr:   nil,
			response: sdk.BootstrapConfig{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view with non-existent client Id",
			domainID: domainID,
			token:    validToken,
			id:       invalid,
			svcResp:  bootstrap.Config{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.BootstrapConfig{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view with response that cannot be unmarshalled",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			svcResp: bootstrap.Config{
				ClientID: clientId,
				Channels: []bootstrap.Channel{
					{
						ID: channel1Id,
						Metadata: map[string]interface{}{
							"test": make(chan int),
						},
					},
				},
			},
			svcErr:   nil,
			response: sdk.BootstrapConfig{},
			err:      errors.NewSDKError(errJsonEOF),
		},
		{
			desc:     "view with empty client Id",
			domainID: domainID,
			token:    validToken,
			id:       "",
			svcResp:  bootstrap.Config{},
			svcErr:   nil,
			response: sdk.BootstrapConfig{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := bsvc.On("View", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
			resp, err := mgsdk.ViewBootstrap(tc.id, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if err == nil {
				ok := svcCall.Parent.AssertCalled(t, "View", mock.Anything, tc.session, tc.id)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc              string
		domainID          string
		token             string
		session           mgauthn.Session
		cfg               sdk.BootstrapConfig
		svcReq            bootstrap.Config
		svcErr            error
		authenticationErr error
		err               errors.SDKError
	}{
		{
			desc:     "update successfully",
			domainID: domainID,
			token:    validToken,
			cfg:      sdkBootstrapConfig,
			svcReq: bootstrap.Config{
				ClientID: clientId,
				Name:     bsName,
				Content:  content,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "update with invalid token",
			domainID: domainID,
			token:    invalidToken,
			cfg:      sdkBootstrapConfig,
			svcReq: bootstrap.Config{
				ClientID: clientId,
				Name:     bsName,
				Content:  content,
			},
			authenticationErr: svcerr.ErrAuthentication,
			err:               errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update with empty token",
			domainID: domainID,
			token:    "",
			cfg:      sdkBootstrapConfig,
			svcReq:   bootstrap.Config{},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update with config that cannot be marshalled",
			domainID: domainID,
			token:    validToken,
			cfg: sdk.BootstrapConfig{
				Channels: map[string]interface{}{
					"channel1": make(chan int),
				},
				ExternalID:   externalId,
				ExternalKey:  externalKey,
				ClientID:     clientId,
				ClientSecret: clientSecret,
				Name:         bsName,
				ClientCert:   clientCert,
				ClientKey:    clientKey,
				CACert:       caCert,
				Content:      content,
			},
			svcReq: bootstrap.Config{
				ClientID: clientId,
				Name:     bsName,
				Content:  content,
			},
			svcErr: nil,
			err:    errors.NewSDKError(errMarshalChan),
		},
		{
			desc:     "update with non-existent client Id",
			domainID: domainID,
			token:    validToken,
			cfg: sdk.BootstrapConfig{
				ClientID: invalid,
				Channels: []sdk.Channel{
					{
						ID: channel1Id,
					},
				},
				ExternalID:  externalId,
				ExternalKey: externalKey,
				Content:     content,
				Name:        bsName,
			},
			svcReq: bootstrap.Config{
				ClientID: invalid,
				Name:     bsName,
				Content:  content,
			},
			svcErr: svcerr.ErrNotFound,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update with empty client Id",
			domainID: domainID,
			token:    validToken,
			cfg: sdk.BootstrapConfig{
				ClientID: "",
				Channels: []sdk.Channel{
					{
						ID: channel1Id,
					},
				},
				ExternalID:  externalId,
				ExternalKey: externalKey,
				Content:     content,
				Name:        bsName,
			},
			svcReq: bootstrap.Config{
				ClientID: "",
				Name:     bsName,
				Content:  content,
			},
			svcErr: nil,
			err:    errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "update with config with only client Id",
			domainID: domainID,
			token:    validToken,
			cfg: sdk.BootstrapConfig{
				ClientID: clientId,
			},
			svcReq: bootstrap.Config{
				ClientID: clientId,
			},
			svcErr: nil,
			err:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticationErr)
			svcCall := bsvc.On("Update", mock.Anything, tc.session, tc.svcReq).Return(tc.svcErr)
			err := mgsdk.UpdateBootstrap(tc.cfg, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Update", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateBootstrapCerts(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updateconfigRes := sdk.BootstrapConfig{
		ClientID:   clientId,
		ClientCert: clientCert,
		CACert:     caCert,
		ClientKey:  clientKey,
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		id              string
		clientCert      string
		clientKey       string
		caCert          string
		svcResp         bootstrap.Config
		svcErr          error
		authenticateErr error
		response        sdk.BootstrapConfig
		err             errors.SDKError
	}{
		{
			desc:       "update certs successfully",
			domainID:   domainID,
			token:      validToken,
			id:         clientId,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			svcResp:    bootstrapConfig,
			svcErr:     nil,
			response:   updateconfigRes,
			err:        nil,
		},
		{
			desc:            "update certs with invalid token",
			domainID:        domainID,
			token:           validToken,
			id:              clientId,
			clientCert:      clientCert,
			clientKey:       clientKey,
			caCert:          caCert,
			svcResp:         bootstrap.Config{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:       "update certs with empty token",
			domainID:   domainID,
			token:      "",
			id:         clientId,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			svcResp:    bootstrap.Config{},
			svcErr:     nil,
			err:        errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:       "update certs with non-existent client Id",
			domainID:   domainID,
			token:      validToken,
			id:         invalid,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			svcResp:    bootstrap.Config{},
			svcErr:     svcerr.ErrNotFound,
			err:        errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:       "update certs with empty certs",
			domainID:   domainID,
			token:      validToken,
			id:         clientId,
			clientCert: "",
			clientKey:  "",
			caCert:     "",
			svcResp:    bootstrap.Config{},
			svcErr:     nil,
			err:        nil,
		},
		{
			desc:       "update certs with empty id",
			domainID:   domainID,
			token:      validToken,
			id:         "",
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			svcResp:    bootstrap.Config{},
			svcErr:     nil,
			err:        errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := bsvc.On("UpdateCert", mock.Anything, tc.session, tc.id, tc.clientCert, tc.clientKey, tc.caCert).Return(tc.svcResp, tc.svcErr)
			resp, err := mgsdk.UpdateBootstrapCerts(tc.id, tc.clientCert, tc.clientKey, tc.caCert, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, tc.response, resp)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateBootstrapConnection(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		id              string
		channels        []string
		svcRes          bootstrap.Config
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "update connection successfully",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			channels: []string{channel1Id, channel2Id},
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "update connection with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			id:              clientId,
			channels:        []string{channel1Id, channel2Id},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update connection with empty token",
			domainID: domainID,
			token:    "",
			id:       clientId,
			channels: []string{channel1Id, channel2Id},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update connection with non-existent client Id",
			domainID: domainID,
			token:    validToken,
			id:       invalid,
			channels: []string{channel1Id, channel2Id},
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update connection with non-existent channel Id",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			channels: []string{invalid},
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update connection with empty channels",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			channels: []string{},
			svcErr:   svcerr.ErrUpdateEntity,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "update connection with empty id",
			domainID: domainID,
			token:    validToken,
			id:       "",
			channels: []string{channel1Id, channel2Id},
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
			svcCall := bsvc.On("UpdateConnections", mock.Anything, tc.session, tc.token, tc.id, tc.channels).Return(tc.svcErr)
			err := mgsdk.UpdateBootstrapConnection(tc.id, tc.channels, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateConnections", mock.Anything, tc.session, tc.token, tc.id, tc.channels)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveBootstrap(t *testing.T) {
	bs, bsvc, _, auth := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		id              string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove successfully",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			id:              clientId,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove with non-existent client Id",
			domainID: domainID,
			token:    validToken,
			id:       invalid,
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "remove removed bootstrap",
			domainID: domainID,
			token:    validToken,
			id:       clientId,
			svcErr:   svcerr.ErrNotFound,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "remove with empty token",
			domainID: domainID,
			token:    "",
			id:       clientId,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove with empty id",
			domainID: domainID,
			token:    validToken,
			id:       "",
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
			svcCall := bsvc.On("Remove", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			err := mgsdk.RemoveBootstrap(tc.id, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Remove", mock.Anything, tc.session, tc.id)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestBoostrap(t *testing.T) {
	bs, bsvc, reader, _ := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc        string
		token       string
		externalID  string
		externalKey string
		svcResp     bootstrap.Config
		svcErr      error
		readerResp  interface{}
		readerErr   error
		response    sdk.BootstrapConfig
		err         errors.SDKError
	}{
		{
			desc:        "bootstrap successfully",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  readConfigResponse,
			readerErr:   nil,
			response:    sdkBootsrapConfigRes,
			err:         nil,
		},
		{
			desc:        "bootstrap with invalid token",
			token:       invalidToken,
			externalID:  externalId,
			externalKey: externalKey,
			svcResp:     bootstrap.Config{},
			svcErr:      svcerr.ErrAuthentication,
			readerResp:  bootstrap.Config{},
			readerErr:   nil,
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "bootstrap with error in reader",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   errJsonEOF,
			err:         errors.NewSDKErrorWithStatus(errJsonEOF, http.StatusInternalServerError),
		},
		{
			desc:        "boostrap with response that cannot be unmarshalled",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   nil,
			err:         errors.NewSDKError(errors.New("json: cannot unmarshal string into Go value of type map[string]json.RawMessage")),
		},
		{
			desc:        "bootstrap with empty id",
			token:       validToken,
			externalID:  "",
			externalKey: externalKey,
			svcResp:     bootstrap.Config{},
			svcErr:      nil,
			readerResp:  bootstrap.Config{},
			readerErr:   nil,
			err:         errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:        "boostrap with empty key",
			token:       validToken,
			externalID:  externalId,
			externalKey: "",
			svcResp:     bootstrap.Config{},
			svcErr:      nil,
			readerResp:  bootstrap.Config{},
			readerErr:   nil,
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := bsvc.On("Bootstrap", mock.Anything, tc.externalKey, tc.externalID, false).Return(tc.svcResp, tc.svcErr)
			readerCall := reader.On("ReadConfig", tc.svcResp, false).Return(tc.readerResp, tc.readerErr)
			resp, err := mgsdk.Bootstrap(tc.externalID, tc.externalKey)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, tc.response, resp)
				ok := svcCall.Parent.AssertCalled(t, "Bootstrap", mock.Anything, tc.externalKey, tc.externalID, false)
				assert.True(t, ok)
			}
			svcCall.Unset()
			readerCall.Unset()
		})
	}
}

func TestBootstrapSecure(t *testing.T) {
	bs, bsvc, reader, _ := setupBootstrap()
	defer bs.Close()

	conf := sdk.Config{
		BootstrapURL: bs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	b, err := json.Marshal(readConfigResponse)
	assert.Nil(t, err, fmt.Sprintf("Marshalling bootstrap response expected to succeed: %s.\n", err))
	encResponse, err := encrypt(b, encKey)
	assert.Nil(t, err, fmt.Sprintf("Encrypting bootstrap response expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		token       string
		externalID  string
		externalKey string
		cryptoKey   string
		svcResp     bootstrap.Config
		svcErr      error
		readerResp  []byte
		readerErr   error
		response    sdk.BootstrapConfig
		err         errors.SDKError
	}{
		{
			desc:        "bootstrap successfully",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			cryptoKey:   string(encKey),
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  encResponse,
			readerErr:   nil,
			response:    sdkBootsrapConfigRes,
			err:         nil,
		},
		{
			desc:        "bootstrap with invalid token",
			token:       invalidToken,
			externalID:  externalId,
			externalKey: externalKey,
			cryptoKey:   string(encKey),
			svcResp:     bootstrap.Config{},
			svcErr:      svcerr.ErrAuthentication,
			readerResp:  []byte{0},
			readerErr:   nil,
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "booostrap with invalid crypto key",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			cryptoKey:   invalid,
			svcResp:     bootstrap.Config{},
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   nil,
			err:         errors.NewSDKError(errors.New("crypto/aes: invalid key size 7")),
		},
		{
			desc:        "bootstrap with error in reader",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			cryptoKey:   string(encKey),
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   errJsonEOF,
			err:         errors.NewSDKErrorWithStatus(errJsonEOF, http.StatusInternalServerError),
		},
		{
			desc:        "bootstrap with response that cannot be unmarshalled",
			token:       validToken,
			externalID:  externalId,
			externalKey: externalKey,
			cryptoKey:   string(encKey),
			svcResp:     bootstrapConfig,
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   nil,
			err:         errors.NewSDKError(errJsonEOF),
		},
		{
			desc:        "bootstrap with empty id",
			token:       validToken,
			externalID:  "",
			externalKey: externalKey,
			svcResp:     bootstrap.Config{},
			svcErr:      nil,
			readerResp:  []byte{0},
			readerErr:   nil,
			err:         errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := bsvc.On("Bootstrap", mock.Anything, mock.Anything, tc.externalID, true).Return(tc.svcResp, tc.svcErr)
			readerCall := reader.On("ReadConfig", tc.svcResp, true).Return(tc.readerResp, tc.readerErr)
			resp, err := mgsdk.BootstrapSecure(tc.externalID, tc.externalKey, tc.cryptoKey)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, sdkBootsrapConfigRes, resp)
				ok := svcCall.Parent.AssertCalled(t, "Bootstrap", mock.Anything, mock.Anything, tc.externalID, true)
				assert.True(t, ok)
			}
			svcCall.Unset()
			readerCall.Unset()
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
