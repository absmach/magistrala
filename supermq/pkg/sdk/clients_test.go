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
	"github.com/absmach/supermq/clients"
	api "github.com/absmach/supermq/clients/api/http"
	"github.com/absmach/supermq/clients/mocks"
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

func setupClients() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	tsvc := new(mocks.Service)

	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	authn := new(authnmocks.Authentication)
	api.MakeHandler(tsvc, authn, mux, logger, "", idp)

	return httptest.NewServer(mux), tsvc, authn
}

func TestCreateClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	client := generateTestClient(t)
	createClientReq := sdk.Client{
		Name:        client.Name,
		Tags:        client.Tags,
		Credentials: client.Credentials,
		Metadata:    client.Metadata,
		Status:      client.Status,
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}

	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		createClientReq sdk.Client
		svcReq          clients.Client
		svcRes          []clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:            "create new client successfully",
			domainID:        domainID,
			token:           validToken,
			createClientReq: createClientReq,
			svcReq:          convertClient(createClientReq),
			svcRes:          []clients.Client{convertClient(client)},
			svcErr:          nil,
			response:        client,
			err:             nil,
		},
		{
			desc:            "create new client with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			createClientReq: createClientReq,
			svcReq:          convertClient(createClientReq),
			svcRes:          []clients.Client{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:            "create new client with empty token",
			domainID:        domainID,
			token:           "",
			createClientReq: createClientReq,
			svcReq:          convertClient(createClientReq),
			svcRes:          []clients.Client{},
			svcErr:          nil,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:            "create an existing client",
			domainID:        domainID,
			token:           validToken,
			createClientReq: createClientReq,
			svcReq:          convertClient(createClientReq),
			svcRes:          []clients.Client{},
			svcErr:          svcerr.ErrCreateEntity,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "create a client with name too long",
			domainID: domainID,
			token:    validToken,
			createClientReq: sdk.Client{
				Name:        strings.Repeat("a", 1025),
				Tags:        client.Tags,
				Credentials: client.Credentials,
				Metadata:    client.Metadata,
				Status:      client.Status,
			},
			svcReq:   clients.Client{},
			svcRes:   []clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:     "create a client with invalid id",
			domainID: domainID,
			token:    validToken,
			createClientReq: sdk.Client{
				ID:          "123456789",
				Name:        client.Name,
				Tags:        client.Tags,
				Credentials: client.Credentials,
				Metadata:    client.Metadata,
				Status:      client.Status,
			},
			svcReq:   clients.Client{},
			svcRes:   []clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidIDFormat), http.StatusBadRequest),
		},
		{
			desc:     "create a client with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,
			createClientReq: sdk.Client{
				Name: valid,
				Metadata: map[string]interface{}{
					valid: make(chan int),
				},
			},
			svcReq:   clients.Client{},
			svcRes:   []clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:            "create a client with a response that can't be unmarshalled",
			domainID:        domainID,
			token:           validToken,
			createClientReq: createClientReq,
			svcReq:          convertClient(createClientReq),
			svcRes: []clients.Client{{
				Name:        client.Name,
				Tags:        client.Tags,
				Credentials: clients.Credentials(client.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			}},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("CreateClients", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, []roles.RoleProvision{}, tc.svcErr)
			resp, err := mgsdk.CreateClient(tc.createClientReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateClients", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestCreateClients(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	sdkClients := []sdk.Client{}
	for i := 0; i < 3; i++ {
		client := generateTestClient(t)
		sdkClients = append(sdkClients, client)
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc                 string
		domainID             string
		token                string
		session              smqauthn.Session
		createClientsRequest []sdk.Client
		svcReq               []clients.Client
		svcRes               []clients.Client
		svcErr               error
		authenticateErr      error
		response             []sdk.Client
		err                  errors.SDKError
	}{
		{
			desc:                 "create new clients successfully",
			domainID:             domainID,
			token:                validToken,
			createClientsRequest: sdkClients,
			svcReq:               convertClients(sdkClients...),
			svcRes:               convertClients(sdkClients...),
			svcErr:               nil,
			response:             sdkClients,
			err:                  nil,
		},
		{
			desc:                 "create new clients with invalid token",
			domainID:             domainID,
			token:                invalidToken,
			createClientsRequest: sdkClients,
			svcReq:               convertClients(sdkClients...),
			svcRes:               []clients.Client{},
			authenticateErr:      svcerr.ErrAuthentication,
			response:             []sdk.Client{},
			err:                  errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:                 "create new clients with empty token",
			domainID:             domainID,
			token:                "",
			createClientsRequest: sdkClients,
			svcReq:               convertClients(sdkClients...),
			svcRes:               []clients.Client{},
			svcErr:               nil,
			response:             []sdk.Client{},
			err:                  errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:                 "create new clients with a request that can't be marshalled",
			domainID:             domainID,
			token:                validToken,
			createClientsRequest: []sdk.Client{{Name: "test", Metadata: map[string]interface{}{"test": make(chan int)}}},
			svcReq:               convertClients(sdkClients...),
			svcRes:               []clients.Client{},
			svcErr:               nil,
			response:             []sdk.Client{},
			err:                  errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:                 "create new clients with a response that can't be unmarshalled",
			domainID:             domainID,
			token:                validToken,
			createClientsRequest: sdkClients,
			svcReq:               convertClients(sdkClients...),
			svcRes: []clients.Client{{
				Name:        sdkClients[0].Name,
				Tags:        sdkClients[0].Tags,
				Credentials: clients.Credentials(sdkClients[0].Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			}},
			svcErr:   nil,
			response: []sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("CreateClients", mock.Anything, tc.session, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2]).Return(tc.svcRes, []roles.RoleProvision{}, tc.svcErr)
			resp, err := mgsdk.CreateClients(tc.createClientsRequest, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateClients", mock.Anything, tc.session, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2])
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListClients(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	var sdkClients []sdk.Client
	for i := 10; i < 100; i++ {
		c := generateTestClient(t)
		if i == 50 {
			c.Status = clients.DisabledStatus.String()
			c.Tags = []string{"tag1", "tag2"}
		}
		sdkClients = append(sdkClients, c)
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		domainID        string
		session         smqauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          clients.Page
		svcRes          clients.ClientsPage
		svcErr          error
		authenticateErr error
		response        sdk.ClientsPage
		err             errors.SDKError
	}{
		{
			desc:     "list all clients successfully",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
				Offset:  0,
				Limit:   100,
			},
			svcRes: clients.ClientsPage{
				Page: clients.Page{
					Offset: 0,
					Limit:  100,
					Total:  uint64(len(sdkClients)),
				},
				Clients: convertClients(sdkClients...),
			},
			svcErr: nil,
			response: sdk.ClientsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: uint64(len(sdkClients)),
				},
				Clients: sdkClients,
			},
		},
		{
			desc:     "list all clients with an invalid token",
			domainID: domainID,
			token:    invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
				Offset:  0,
				Limit:   100,
			},
			svcRes:          clients.ClientsPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ClientsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list all clients with limit greater than max",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  1000,
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
			},
			svcRes:   clients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ClientsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:     "list all clients with name size greater than max",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Name:   strings.Repeat("a", 1025),
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
			},
			svcRes:   clients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ClientsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:     "list all clients with status",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Status: clients.DisabledStatus.String(),
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
				Offset:  0,
				Limit:   100,
				Status:  clients.DisabledStatus,
			},
			svcRes: clients.ClientsPage{
				Page: clients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertClients(sdkClients[50]),
			},
			svcErr: nil,
			response: sdk.ClientsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Clients: []sdk.Client{sdkClients[50]},
			},
			err: nil,
		},
		{
			desc:     "list all clients with tags",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Tag:    "tag1",
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
				Offset:  0,
				Limit:   100,
				Tag:     "tag1",
			},
			svcRes: clients.ClientsPage{
				Page: clients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertClients(sdkClients[50]),
			},
			svcErr: nil,
			response: sdk.ClientsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Clients: []sdk.Client{sdkClients[50]},
			},
			err: nil,
		},
		{
			desc:     "list all clients with invalid metadata",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
			},
			svcRes:   clients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ClientsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "list all clients with response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: clients.Page{
				Actions: []string{},
				Order:   "updated_at",
				Dir:     "asc",
				Offset:  0,
				Limit:   100,
			},
			svcRes: clients.ClientsPage{
				Page: clients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: []clients.Client{{
					Name:        sdkClients[0].Name,
					Tags:        sdkClients[0].Tags,
					Credentials: clients.Credentials(sdkClients[0].Credentials),
					Metadata: clients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ClientsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ListClients", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Clients(tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	sdkClient := generateTestClient(t)
	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:     "view client successfully",
			domainID: domainID,
			token:    validToken,
			clientID: sdkClient.ID,
			svcRes:   convertClient(sdkClient),
			svcErr:   nil,
			response: sdkClient,
			err:      nil,
		},
		{
			desc:            "view client with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        sdkClient.ID,
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view client with empty token",
			domainID: domainID,
			token:    "",
			clientID: sdkClient.ID,
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view client with an invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view client with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "view client with response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			clientID: sdkClient.ID,
			svcRes: clients.Client{
				Name:        sdkClient.Name,
				Tags:        sdkClient.Tags,
				Credentials: clients.Credentials(sdkClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("View", mock.Anything, tc.session, tc.clientID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Client(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "View", mock.Anything, tc.session, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	sdkClient := generateTestClient(t)
	updatedClient := sdkClient
	updatedClient.Name = "newName"
	updatedClient.Metadata = map[string]interface{}{
		"newKey": "newValue",
	}
	updateClientReq := sdk.Client{
		ID:       sdkClient.ID,
		Name:     updatedClient.Name,
		Metadata: updatedClient.Metadata,
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		updateClientReq sdk.Client
		svcReq          clients.Client
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:            "update client successfully",
			domainID:        domainID,
			token:           validToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          convertClient(updatedClient),
			svcErr:          nil,
			response:        updatedClient,
			err:             nil,
		},
		{
			desc:            "update client with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:            "update client with empty token",
			domainID:        domainID,
			token:           "",
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          clients.Client{},
			svcErr:          nil,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update client with an invalid client id",
			domainID: domainID,
			token:    validToken,
			updateClientReq: sdk.Client{
				ID:   wrongID,
				Name: updatedClient.Name,
			},
			svcReq: convertClient(sdk.Client{
				ID:   wrongID,
				Name: updatedClient.Name,
			}),
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "update client with empty client id",
			domainID: domainID,
			token:    validToken,

			updateClientReq: sdk.Client{
				ID:   "",
				Name: updatedClient.Name,
			},
			svcReq: convertClient(sdk.Client{
				ID:   "",
				Name: updatedClient.Name,
			}),
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "update client with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,

			updateClientReq: sdk.Client{
				ID: valid,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   clients.Client{},
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:            "update client with a response that can't be unmarshalled",
			domainID:        domainID,
			token:           validToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes: clients.Client{
				Name:        updatedClient.Name,
				Tags:        updatedClient.Tags,
				Credentials: clients.Credentials(updatedClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Update", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateClient(tc.updateClientReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Update", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateClientTags(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	sdkClient := generateTestClient(t)
	updatedClient := sdkClient
	updatedClient.Tags = []string{"newTag1", "newTag2"}
	updateClientReq := sdk.Client{
		ID:   sdkClient.ID,
		Tags: updatedClient.Tags,
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		updateClientReq sdk.Client
		svcReq          clients.Client
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:            "update client tags successfully",
			domainID:        domainID,
			token:           validToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          convertClient(updatedClient),
			svcErr:          nil,
			response:        updatedClient,
			err:             nil,
		},
		{
			desc:            "update client tags with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:            "update client tags with empty token",
			domainID:        domainID,
			token:           "",
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes:          clients.Client{},
			svcErr:          nil,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update client tags with an invalid client id",
			domainID: domainID,
			token:    validToken,
			updateClientReq: sdk.Client{
				ID:   wrongID,
				Tags: updatedClient.Tags,
			},
			svcReq: convertClient(sdk.Client{
				ID:   wrongID,
				Tags: updatedClient.Tags,
			}),
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "update client tags with empty client id",
			domainID: domainID,
			token:    validToken,
			updateClientReq: sdk.Client{
				ID:   "",
				Tags: updatedClient.Tags,
			},
			svcReq: convertClient(sdk.Client{
				ID:   "",
				Tags: updatedClient.Tags,
			}),
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "update client tags with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,
			updateClientReq: sdk.Client{
				ID: valid,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   clients.Client{},
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:            "update client tags with a response that can't be unmarshalled",
			domainID:        domainID,
			token:           validToken,
			updateClientReq: updateClientReq,
			svcReq:          convertClient(updateClientReq),
			svcRes: clients.Client{
				Name:        updatedClient.Name,
				Tags:        updatedClient.Tags,
				Credentials: clients.Credentials(updatedClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("UpdateTags", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateClientTags(tc.updateClientReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateTags", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	sdkClient := generateTestClient(t)
	newSecret := generateUUID(t)
	updatedClient := sdkClient
	updatedClient.Credentials.Secret = newSecret

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		newSecret       string
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:      "update client secret successfully",
			domainID:  domainID,
			token:     validToken,
			clientID:  sdkClient.ID,
			newSecret: newSecret,
			svcRes:    convertClient(updatedClient),
			svcErr:    nil,
			response:  updatedClient,
			err:       nil,
		},
		{
			desc:            "update client secret with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        sdkClient.ID,
			newSecret:       newSecret,
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "update client secret with empty token",
			domainID:  domainID,
			token:     "",
			clientID:  sdkClient.ID,
			newSecret: newSecret,
			svcRes:    clients.Client{},
			svcErr:    nil,
			response:  sdk.Client{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "update client secret with an invalid client id",
			domainID:  domainID,
			token:     validToken,
			clientID:  wrongID,
			newSecret: newSecret,
			svcRes:    clients.Client{},
			svcErr:    svcerr.ErrUpdateEntity,
			response:  sdk.Client{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:      "update client secret with empty client id",
			domainID:  domainID,
			token:     validToken,
			clientID:  "",
			newSecret: newSecret,
			svcRes:    clients.Client{},
			svcErr:    nil,
			response:  sdk.Client{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "update client with empty new secret",
			domainID:  domainID,
			token:     validToken,
			clientID:  sdkClient.ID,
			newSecret: "",
			svcRes:    clients.Client{},
			svcErr:    nil,
			response:  sdk.Client{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest),
		},
		{
			desc:      "update client secret with a response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			clientID:  sdkClient.ID,
			newSecret: newSecret,
			svcRes: clients.Client{
				Name:        updatedClient.Name,
				Tags:        updatedClient.Tags,
				Credentials: clients.Credentials(updatedClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("UpdateSecret", mock.Anything, tc.session, tc.clientID, tc.newSecret).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateClientSecret(tc.clientID, tc.newSecret, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, tc.session, tc.clientID, tc.newSecret)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	client := generateTestClient(t)
	enabledClient := client
	enabledClient.Status = clients.EnabledStatus.String()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:     "enable client successfully",
			domainID: domainID,
			token:    validToken,
			clientID: client.ID,
			svcRes:   convertClient(enabledClient),
			svcErr:   nil,
			response: enabledClient,
			err:      nil,
		},
		{
			desc:            "enable client with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        client.ID,
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "enable client with an invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrEnableClient,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusUnprocessableEntity),
		},
		{
			desc:     "enable client with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "enable client with a response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			clientID: client.ID,
			svcRes: clients.Client{
				Name:        enabledClient.Name,
				Tags:        enabledClient.Tags,
				Credentials: clients.Credentials(enabledClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Enable", mock.Anything, tc.session, tc.clientID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableClient(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Enable", mock.Anything, tc.session, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	client := generateTestClient(t)
	disabledClient := client
	disabledClient.Status = clients.DisabledStatus.String()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		svcRes          clients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Client
		err             errors.SDKError
	}{
		{
			desc:     "disable client successfully",
			domainID: domainID,
			token:    validToken,
			clientID: client.ID,
			svcRes:   convertClient(disabledClient),
			svcErr:   nil,
			response: disabledClient,
			err:      nil,
		},
		{
			desc:            "disable client with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        client.ID,
			svcRes:          clients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Client{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "disable client with an invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrDisableClient,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrDisableClient, http.StatusInternalServerError),
		},
		{
			desc:     "disable client with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			svcRes:   clients.Client{},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "disable client with a response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			clientID: client.ID,
			svcRes: clients.Client{
				Name:        disabledClient.Name,
				Tags:        disabledClient.Tags,
				Credentials: clients.Credentials(disabledClient.Credentials),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Client{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Disable", mock.Anything, tc.session, tc.clientID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableClient(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Disable", mock.Anything, tc.session, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteClient(t *testing.T) {
	ts, tsvc, auth := setupClients()
	defer ts.Close()

	client := generateTestClient(t)

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete client successfully",
			domainID: domainID,
			token:    validToken,
			clientID: client.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "delete client with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        client.ID,
			authenticateErr: svcerr.ErrAuthorization,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "delete client with empty token",
			domainID: domainID,
			token:    "",
			clientID: client.ID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete client with an invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "delete client with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			svcErr:   nil,
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Delete", mock.Anything, tc.session, tc.clientID).Return(tc.svcErr)
			err := mgsdk.DeleteClient(tc.clientID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Delete", mock.Anything, tc.session, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestSetClientParent(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	parentID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		parentID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "set client parent successfully",
			domainID: domainID,
			token:    validToken,
			clientID: clientID,
			parentID: parentID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "set client parent with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        clientID,
			parentID:        parentID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "set client parent with empty token",
			domainID: domainID,
			token:    "",
			clientID: clientID,
			parentID: parentID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "set client parent with invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			parentID: parentID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "set client parent with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
			parentID: parentID,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "set client parent with empty parent id",
			domainID: domainID,
			token:    validToken,
			clientID: clientID,
			parentID: "",
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingParentGroupID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := csvc.On("SetParentGroup", mock.Anything, tc.session, tc.parentID, tc.clientID).Return(tc.svcErr)
			err := mgsdk.SetClientParent(tc.clientID, tc.domainID, tc.parentID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "SetParentGroup", mock.Anything, tc.session, tc.parentID, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveClientParent(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	parentID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         smqauthn.Session
		clientID        string
		parentID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove client parent successfully",
			domainID: domainID,
			token:    validToken,
			clientID: clientID,
			parentID: parentID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove client parent with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			clientID:        clientID,
			parentID:        parentID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove client parent with empty token",
			domainID: domainID,
			token:    "",
			clientID: clientID,
			parentID: parentID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove client parent with invalid client id",
			domainID: domainID,
			token:    validToken,
			clientID: wrongID,
			parentID: parentID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove client parent with empty client id",
			domainID: domainID,
			token:    validToken,
			clientID: "",
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
			svcCall := csvc.On("RemoveParentGroup", mock.Anything, tc.session, tc.clientID).Return(tc.svcErr)
			err := mgsdk.RemoveClientParent(tc.clientID, tc.domainID, tc.parentID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveParentGroup", mock.Anything, tc.session, tc.clientID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestCreateClientRole(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
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
		EntityID:  clientID,
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
		clientID        string
		roleReq         sdk.RoleReq
		svcRes          roles.RoleProvision
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "create client role successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleReq:  rReq,
			svcRes:   roleProvision,
			svcErr:   nil,
			response: convertRoleProvision(roleProvision),
			err:      nil,
		},
		{
			desc:            "create client role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleReq:         rReq,
			svcRes:          roles.RoleProvision{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "create client role with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "create client role with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "create client role with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleReq:  rReq,
			svcRes:   roles.RoleProvision{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidIDFormat), http.StatusBadRequest),
		},
		{
			desc:     "create client role with empty role name",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("AddRole", mock.Anything, tc.session, tc.clientID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateClientRole(tc.clientID, tc.domainID, tc.roleReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "AddRole", mock.Anything, tc.session, tc.clientID, tc.roleReq.RoleName, tc.roleReq.OptionalActions, tc.roleReq.OptionalMembers)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListClientRoles(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  clientID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		pageMeta        sdk.PageMetadata
		svcRes          roles.RolePage
		svcErr          error
		authenticateErr error
		response        sdk.RolesPage
		err             errors.SDKError
	}{
		{
			desc:     "list client roles successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			desc:     "list client roles with invalid token",
			token:    invalidToken,
			domainID: domainID,
			clientID: clientID,
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
			desc:     "list client roles with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   roles.RolePage{},
			response: sdk.RolesPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list client roles with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
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
			desc:     "list client roles with empty client id",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			clientID: "",
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
			svcCall := csvc.On("RetrieveAllRoles", mock.Anything, tc.session, tc.clientID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ClientRoles(tc.clientID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveAllRoles", mock.Anything, tc.session, tc.clientID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewClientRole(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      roleName,
		EntityID:  clientID,
		CreatedBy: testsutil.GenerateUUID(t),
		CreatedAt: time.Now().UTC(),
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:     "view client role successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   role.ID,
			svcRes:   role,
			svcErr:   nil,
			response: convertRole(role),
			err:      nil,
		},
		{
			desc:            "view client role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          role.ID,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view client role with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   role.ID,
			svcRes:   roles.Role{},
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view client role with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view client role with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   role.ID,
			svcRes:   roles.Role{},
			svcErr:   nil,
			response: sdk.Role{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "view client role with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RetrieveRole", mock.Anything, tc.session, tc.clientID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ClientRole(tc.clientID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveRole", mock.Anything, tc.session, tc.clientID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateClientRole(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	newRoleName := valid
	userID := testsutil.GenerateUUID(t)
	createdAt := time.Now().UTC().Add(-time.Hour)
	role := roles.Role{
		ID:        testsutil.GenerateUUID(t),
		Name:      newRoleName,
		EntityID:  clientID,
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
		clientID        string
		roleID          string
		newRoleName     string
		svcRes          roles.Role
		svcErr          error
		authenticateErr error
		response        sdk.Role
		err             errors.SDKError
	}{
		{
			desc:        "update client role successfully",
			token:       validToken,
			domainID:    domainID,
			clientID:    clientID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      role,
			svcErr:      nil,
			response:    convertRole(role),
			err:         nil,
		},
		{
			desc:            "update client role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			newRoleName:     newRoleName,
			svcRes:          roles.Role{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Role{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "update client role with empty token",
			token:       "",
			domainID:    domainID,
			clientID:    clientID,
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "update client role with invalid client id",
			token:       validToken,
			domainID:    domainID,
			clientID:    testsutil.GenerateUUID(t),
			roleID:      roleID,
			newRoleName: newRoleName,
			svcRes:      roles.Role{},
			svcErr:      svcerr.ErrAuthorization,
			response:    sdk.Role{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:        "update client role with empty client id",
			token:       validToken,
			domainID:    domainID,
			clientID:    "",
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
			svcCall := csvc.On("UpdateRoleName", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.newRoleName).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateClientRole(tc.clientID, tc.roleID, tc.newRoleName, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateRoleName", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.newRoleName)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteClientRole(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete client role successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "delete client role with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "delete client role with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete client role with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "delete client role with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "delete client role with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RemoveRole", mock.Anything, tc.session, tc.clientID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.DeleteClientRole(tc.clientID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveRole", mock.Anything, tc.session, tc.clientID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddClientRoleActions(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		actions         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add client role actions successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			actions:  actions,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "add client role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add client role actions with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add client role actions with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add client role actions with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			actions:  actions,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add client role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add client role actions with empty actions",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleAddActions", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.actions).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddClientRoleActions(tc.clientID, tc.roleID, tc.domainID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddActions", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListClientRoleActions(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "list client role actions successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			svcRes:   actions,
			svcErr:   nil,
			response: actions,
			err:      nil,
		},
		{
			desc:            "list client role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list client role actions with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list client role actions with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list client role actions with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "list client role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list client role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleListActions", mock.Anything, tc.session, tc.clientID, tc.roleID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ClientRoleActions(tc.clientID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListActions", mock.Anything, tc.session, tc.clientID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveClientRoleActions(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	actions := []string{"create", "update"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		actions         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove client role actions successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			actions:  actions,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove client role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			actions:         actions,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove client role actions with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove client role actions with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove client role actions with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			actions:  actions,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove client role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			actions:  actions,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove client role actions with empty actions",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleRemoveActions", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.actions).Return(tc.svcErr)
			err := mgsdk.RemoveClientRoleActions(tc.clientID, tc.roleID, tc.domainID, tc.actions, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveActions", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.actions)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllClientRoleActions(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all client role actions successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all client role actions with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all client role actions with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all client role actions with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all client role actions with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove all client role actions with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all client role actions with empty role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleRemoveAllActions", mock.Anything, tc.session, tc.clientID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllClientRoleActions(tc.clientID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllActions", mock.Anything, tc.session, tc.clientID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddClientRoleMembers(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		members         []string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        []string
		err             errors.SDKError
	}{
		{
			desc:     "add client role members successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			members:  members,
			svcRes:   members,
			svcErr:   nil,
			response: members,
			err:      nil,
		},
		{
			desc:            "add client role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			response:        []string{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "add client role members with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "add client role members with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add client role members with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			members:  members,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "add client role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			response: []string{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "add client role members with empty members",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleAddMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.members).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.AddClientRoleMembers(tc.clientID, tc.roleID, tc.domainID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleAddMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListClientRoleMembers(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		pageMeta        sdk.PageMetadata
		svcRes          roles.MembersPage
		svcErr          error
		authenticateErr error
		response        sdk.RoleMembersPage
		err             errors.SDKError
	}{
		{
			desc:     "list client role members successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			desc:     "list client role members with invalid token",
			token:    invalidToken,
			domainID: domainID,
			clientID: clientID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list client role members with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list client role members with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: roleID,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list client role members with empty client id",
			token:    validToken,
			domainID: domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			clientID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "list client role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  5,
			},
			roleID: invalid,
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list client role members with empty role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleListMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ClientRoleMembers(tc.clientID, tc.roleID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleListMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.pageMeta.Limit, tc.pageMeta.Offset)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveClientRoleMembers(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)
	members := []string{"user1", "user2"}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		members         []string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove client role members successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			members:  members,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove client role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			members:         members,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove client role members with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove client role members with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove client role members with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			members:  members,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove client role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			members:  members,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove client role members with empty members",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleRemoveMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.members).Return(tc.svcErr)
			err := mgsdk.RemoveClientRoleMembers(tc.clientID, tc.roleID, tc.domainID, tc.members, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveMembers", mock.Anything, tc.session, tc.clientID, tc.roleID, tc.members)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllClientRoleMembers(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	roleID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		domainID        string
		clientID        string
		roleID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "remove all client role members successfully",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "remove all client role members with invalid token",
			token:           invalidToken,
			domainID:        domainID,
			clientID:        clientID,
			roleID:          roleID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "remove all client role members with empty token",
			token:    "",
			domainID: domainID,
			clientID: clientID,
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "remove all client role members with invalid client id",
			token:    validToken,
			domainID: domainID,
			clientID: testsutil.GenerateUUID(t),
			roleID:   roleID,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all client role members with empty client id",
			token:    validToken,
			domainID: domainID,
			clientID: "",
			roleID:   roleID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "remove all client role members with invalid role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
			roleID:   invalid,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "remove all client role members with empty role id",
			token:    validToken,
			domainID: domainID,
			clientID: clientID,
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
			svcCall := csvc.On("RoleRemoveAllMembers", mock.Anything, tc.session, tc.clientID, tc.roleID).Return(tc.svcErr)
			err := mgsdk.RemoveAllClientRoleMembers(tc.clientID, tc.roleID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RoleRemoveAllMembers", mock.Anything, tc.session, tc.clientID, tc.roleID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListAvailableClientRoleActions(t *testing.T) {
	ts, csvc, auth := setupClients()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
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
			resp, err := mgsdk.AvailableClientRoleActions(tc.domainID, tc.token)
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

func generateTestClient(t *testing.T) sdk.Client {
	createdAt, err := time.Parse(time.RFC3339, "2023-03-03T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	updatedAt := createdAt
	return sdk.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "clientname",
		Credentials: sdk.ClientCredentials{
			Identity: "client@example.com",
			Secret:   generateUUID(t),
		},
		Tags:      []string{"tag1", "tag2"},
		Metadata:  validMetadata,
		Status:    clients.EnabledStatus.String(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
