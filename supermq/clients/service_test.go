// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"context"
	"fmt"
	"testing"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	"github.com/absmach/supermq/clients"
	climocks "github.com/absmach/supermq/clients/mocks"
	gpmocks "github.com/absmach/supermq/groups/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	policysvc "github.com/absmach/supermq/pkg/policies"
	policymocks "github.com/absmach/supermq/pkg/policies/mocks"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	secret         = "strongsecret"
	validTMetadata = clients.Metadata{"role": "client"}
	ID             = "6e5e10b3-d4df-4758-b426-4929d55ad740"
	client         = clients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: clients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validTMetadata,
		Status:      clients.EnabledStatus,
	}
	validToken = "token"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID    = testsutil.GenerateUUID(&testing.T{})
)

var (
	pService     *policymocks.Service
	cache        *climocks.Cache
	repo         *climocks.Repository
	chgRPCClient *chmocks.ChannelsServiceClient
	gpgRPCClient *gpmocks.GroupsServiceClient
)

func newService() clients.Service {
	pService = new(policymocks.Service)
	cache = new(climocks.Cache)
	idProvider := uuid.NewMock()
	sidProvider := uuid.NewMock()
	repo = new(climocks.Repository)
	chgRPCClient = new(chmocks.ChannelsServiceClient)
	gpgRPCClient = new(gpmocks.GroupsServiceClient)
	availableActions := []roles.Action{}
	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		clients.BuiltInRoleAdmin: availableActions,
	}
	tsv, _ := clients.NewService(repo, pService, cache, chgRPCClient, gpgRPCClient, idProvider, sidProvider, availableActions, builtInRoles)
	return tsv
}

func TestCreateClients(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		client          clients.Client
		token           string
		addPolicyErr    error
		deletePolicyErr error
		saveErr         error
		addRoleErr      error
		deleteErr       error
		err             error
	}{
		{
			desc:   "create a new client successfully",
			client: client,
			token:  validToken,
			err:    nil,
		},
		{
			desc:    "create an existing client",
			client:  client,
			token:   validToken,
			saveErr: repoerr.ErrConflict,
			err:     repoerr.ErrConflict,
		},
		{
			desc: "create a new client without secret",
			client: clients.Client{
				Name: "clientWithoutSecret",
				Credentials: clients.Credentials{
					Identity: "newclientwithoutsecret@example.com",
				},
				Status: clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new client without identity",
			client: clients.Client{
				Name: "clientWithoutIdentity",
				Credentials: clients.Credentials{
					Identity: "newclientwithoutsecret@example.com",
				},
				Status: clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled client with name",
			client: clients.Client{
				Name: "clientWithName",
				Credentials: clients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
				Status: clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},

		{
			desc: "create a new disabled client with name",
			client: clients.Client{
				Name: "clientWithName",
				Credentials: clients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled client with tags",
			client: clients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: clients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled client with tags",
			client: clients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: clients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: clients.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled client with metadata",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validTMetadata,
				Status:   clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled client with metadata",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validTMetadata,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled client",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new client with valid disabled status",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
				Status: clients.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new client with all fields",
			client: clients.Client{
				Name: "newclientwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: clients.Credentials{
					Identity: "newclientwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: clients.Metadata{
					"name": "newclientwithallfields",
				},
				Status: clients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new client with invalid status",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: clients.AllStatus,
			},
			token: validToken,
			err:   svcerr.ErrInvalidStatus,
		},
		{
			desc: "create a new client with failed add policies response",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: clients.EnabledStatus,
			},
			token:        validToken,
			addPolicyErr: svcerr.ErrInvalidPolicy,
			err:          svcerr.ErrInvalidPolicy,
		},
		{
			desc: "create a new client with failed delete policies response",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: clients.EnabledStatus,
			},
			token:           validToken,
			saveErr:         repoerr.ErrConflict,
			deletePolicyErr: svcerr.ErrInvalidPolicy,
			err:             repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("Save", context.Background(), mock.Anything).Return([]clients.Client{tc.client}, tc.saveErr)
		policyCall := pService.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPolicyErr)
		policyCall1 := pService.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePolicyErr)
		repoCall1 := repo.On("AddRoles", context.Background(), mock.Anything).Return([]roles.RoleProvision{}, tc.addRoleErr)
		repoCall2 := repo.On("Delete", context.Background(), mock.Anything).Return(tc.deleteErr)
		expected, _, err := svc.CreateClients(context.Background(), smqauthn.Session{}, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected[0].ID
			tc.client.CreatedAt = expected[0].CreatedAt
			tc.client.UpdatedAt = expected[0].UpdatedAt
			tc.client.Credentials.Secret = expected[0].Credentials.Secret
			tc.client.Domain = expected[0].Domain
			tc.client.UpdatedBy = expected[0].UpdatedBy
			assert.Equal(t, tc.client, expected[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected[0]))
		}
		repoCall.Unset()
		policyCall.Unset()
		policyCall1.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc        string
		clientID    string
		response    clients.Client
		retrieveErr error
		err         error
	}{
		{
			desc:     "view client successfully",
			response: client,
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:     "view client with an invalid token",
			response: clients.Client{},
			clientID: "",
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:        "view client with valid token and invalid client id",
			response:    clients.Client{},
			clientID:    wrongID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: clients.Client{},
			clientID: wrongID,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall1 := repo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.response, tc.err)
		rClient, err := svc.View(context.Background(), smqauthn.Session{}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		repoCall1.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc := newService()

	adminID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	nonAdminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                    string
		userKind                string
		session                 smqauthn.Session
		page                    clients.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     clients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                clients.ClientsPage
		id                      string
		size                    uint64
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all clients successfully as non admin",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset: 0,
				Limit:  100,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: clients.ClientsPage{
				Page: clients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []clients.Client{client, client},
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []clients.Client{client, client},
			},
			err: nil,
		},
		{
			desc:     "list all clients as non admin with failed to retrieve all",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset: 0,
				Limit:  100,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: clients.ClientsPage{},
			response:            clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed super admin",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset: 0,
				Limit:  100,
			},
			response:            clients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
		},
		{
			desc:     "list all clients as non admin with failed to list objects",
			userKind: "non-admin",
			id:       nonAdminID,
			page: clients.Page{
				Offset: 0,
				Limit:  100,
			},
			retrieveAllErr:      repoerr.ErrNotFound,
			response:            clients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		retrieveAllCall := repo.On("RetrieveAll", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		retrieveUserClientsCall := repo.On("RetrieveUserClients", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListClients(context.Background(), tc.session, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		retrieveAllCall.Unset()
		retrieveUserClientsCall.Unset()
	}

	cases2 := []struct {
		desc                    string
		userKind                string
		session                 smqauthn.Session
		page                    clients.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     clients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                clients.ClientsPage
		id                      string
		size                    uint64
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all clients as admin successfully",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: clients.ClientsPage{
				Page: clients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []clients.Client{client, client},
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []clients.Client{client, client},
			},
			err: nil,
		},
		{
			desc:     "list all clients as admin with failed to retrieve all",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list clients",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			retrieveAllResponse: clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		retrieveAllCall := repo.On("RetrieveAll", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListClients(context.Background(), tc.session, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		retrieveAllCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc := newService()

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = clients.Metadata{"role": "test"}

	cases := []struct {
		desc           string
		client         clients.Client
		session        smqauthn.Session
		updateResponse clients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client name successfully",
			client:         client1,
			session:        smqauthn.Session{UserID: validID},
			updateResponse: client1,
			err:            nil,
		},
		{
			desc:           "update client metadata with valid token",
			client:         client2,
			updateResponse: client2,
			session:        smqauthn.Session{UserID: validID},
			err:            nil,
		},
		{
			desc:           "update client with failed to update repo",
			client:         client1,
			updateResponse: clients.Client{},
			session:        smqauthn.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := repo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.Update(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall1.Unset()
	}
}

func TestUpdateTags(t *testing.T) {
	svc := newService()

	client.Tags = []string{"updated"}

	cases := []struct {
		desc           string
		client         clients.Client
		session        smqauthn.Session
		updateResponse clients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client tags successfully",
			client:         client,
			session:        smqauthn.Session{UserID: validID},
			updateResponse: client,
			err:            nil,
		},
		{
			desc:           "update client tags with failed to update repo",
			client:         client,
			updateResponse: clients.Client{},
			session:        smqauthn.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := repo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.UpdateTags(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall1.Unset()
	}
}

func TestUpdateSecret(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc                 string
		client               clients.Client
		newSecret            string
		updateSecretResponse clients.Client
		session              smqauthn.Session
		updateErr            error
		err                  error
	}{
		{
			desc:      "update client secret successfully",
			client:    client,
			newSecret: "newSecret",
			session:   smqauthn.Session{UserID: validID},
			updateSecretResponse: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: client.Credentials.Identity,
					Secret:   "newSecret",
				},
			},
			err: nil,
		},
		{
			desc:                 "update client secret with failed to update repo",
			client:               client,
			newSecret:            "newSecret",
			session:              smqauthn.Session{UserID: validID},
			updateSecretResponse: clients.Client{},
			updateErr:            repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateErr)
		updatedClient, err := svc.UpdateSecret(context.Background(), tc.session, tc.client.ID, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateSecretResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateSecretResponse, updatedClient))
		repoCall.Unset()
	}
}

func TestEnable(t *testing.T) {
	svc := newService()

	enabledClient1 := clients.Client{ID: ID, Credentials: clients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: clients.EnabledStatus}
	disabledClient1 := clients.Client{ID: ID, Credentials: clients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: clients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = clients.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              smqauthn.Session
		client               clients.Client
		changeStatusResponse clients.Client
		retrieveByIDResponse clients.Client
		changeStatusErr      error
		retrieveIDErr        error
		err                  error
	}{
		{
			desc:                 "enable disabled client",
			id:                   disabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: endisabledClient1,
			retrieveByIDResponse: disabledClient1,
			err:                  nil,
		},
		{
			desc:                 "enable disabled client with failed to update repo",
			id:                   disabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: disabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "enable enabled client",
			id:                   enabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: enabledClient1,
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable non-existing client",
			id:                   wrongID,
			session:              smqauthn.Session{UserID: validID},
			client:               clients.Client{},
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: clients.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.Enable(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisable(t *testing.T) {
	svc := newService()

	enabledClient1 := clients.Client{ID: ID, Credentials: clients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: clients.EnabledStatus}
	disabledClient1 := clients.Client{ID: ID, Credentials: clients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: clients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = clients.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              smqauthn.Session
		client               clients.Client
		changeStatusResponse clients.Client
		retrieveByIDResponse clients.Client
		changeStatusErr      error
		retrieveIDErr        error
		removeErr            error
		err                  error
	}{
		{
			desc:                 "disable enabled client",
			id:                   enabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			err:                  nil,
		},
		{
			desc:                 "disable client with failed to update repo",
			id:                   enabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "disable disabled client",
			id:                   disabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: disabledClient1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable non-existing client",
			id:                   wrongID,
			client:               clients.Client{},
			session:              smqauthn.Session{UserID: validID},
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: clients.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable client with failed to remove from cache",
			id:                   enabledClient1.ID,
			session:              smqauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			removeErr:            svcerr.ErrRemoveEntity,
			err:                  svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		repoCall2 := cache.On("Remove", mock.Anything, mock.Anything).Return(tc.removeErr)
		_, err := svc.Disable(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDelete(t *testing.T) {
	svc := newService()

	client := clients.Client{
		ID: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc                 string
		clientID             string
		checkConnectionsRes  bool
		checkConnectionsErr  error
		removeConnectionsErr error
		changeStatusErr      error
		deletePoliciesErr    error
		removeErr            error
		deleteErr            error
		err                  error
	}{
		{
			desc:     "Delete client without connections successfully",
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:                "Delete client with connections",
			clientID:            client.ID,
			checkConnectionsRes: true,
			err:                 nil,
		},
		{
			desc:                "Delete client with failed to check connections",
			clientID:            client.ID,
			checkConnectionsErr: svcerr.ErrRemoveEntity,
			err:                 svcerr.ErrRemoveEntity,
		},
		{
			desc:                 "Delete client with failed to remove connections",
			clientID:             client.ID,
			checkConnectionsRes:  true,
			removeConnectionsErr: svcerr.ErrRemoveEntity,
			err:                  svcerr.ErrRemoveEntity,
		},
		{
			desc:      "Delete cliet with failed to remove from cache",
			clientID:  client.ID,
			removeErr: svcerr.ErrRemoveEntity,
			err:       svcerr.ErrRemoveEntity,
		},
		{
			desc:            "Delete client with failed to change status",
			clientID:        client.ID,
			changeStatusErr: svcerr.ErrNotFound,
			err:             svcerr.ErrRemoveEntity,
		},
		{
			desc:              "Delete client with failed to delete policies",
			clientID:          client.ID,
			deletePoliciesErr: svcerr.ErrNotFound,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:      "Delete client with failed to delete",
			clientID:  client.ID,
			deleteErr: svcerr.ErrNotFound,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("DoesClientHaveConnections", context.Background(), mock.Anything).Return(tc.checkConnectionsRes, tc.checkConnectionsErr)
		channelsCall := chgRPCClient.On("RemoveClientConnections", context.Background(), &grpcChannelsV1.RemoveClientConnectionsReq{ClientId: tc.clientID}).Return(&grpcChannelsV1.RemoveClientConnectionsRes{}, tc.removeConnectionsErr)
		repoCall1 := cache.On("Remove", mock.Anything, tc.clientID).Return(tc.removeErr)
		repoCall2 := repo.On("ChangeStatus", context.Background(), clients.Client{ID: tc.clientID, Status: clients.DeletedStatus}).Return(client, tc.changeStatusErr)
		repoCall3 := repo.On("RetrieveEntitiesRolesActionsMembers", context.Background(), []string{tc.clientID}).Return([]roles.EntityActionRole{}, []roles.EntityMemberRole{}, nil)
		policyCall1 := pService.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
		policyCall2 := pService.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
		repoCall4 := repo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		err := svc.Delete(context.Background(), smqauthn.Session{}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		policyCall1.Unset()
		repoCall2.Unset()
		channelsCall.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		policyCall2.Unset()
	}
}

func TestSetParentGroup(t *testing.T) {
	svc := newService()

	parentedClient := client
	parentedClient.ParentGroup = validID

	cparentedClient := client
	cparentedClient.ParentGroup = testsutil.GenerateUUID(t)

	cases := []struct {
		desc               string
		clientID           string
		parentGroupID      string
		session            smqauthn.Session
		retrieveByIDResp   clients.Client
		retrieveByIDErr    error
		retrieveEntityResp *grpcCommonV1.RetrieveEntityRes
		retrieveEntityErr  error
		addPoliciesErr     error
		deletePoliciesErr  error
		setParentGroupErr  error
		err                error
	}{
		{
			desc:             "set parent group successfully",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: nil,
		},
		{
			desc:             "set parent group with failed to retrieve client",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: clients.Client{},
			retrieveByIDErr:  svcerr.ErrNotFound,
			err:              svcerr.ErrUpdateEntity,
		},
		{
			desc:             "set parent group with parent already set",
			clientID:         parentedClient.ID,
			parentGroupID:    validID,
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: parentedClient,
			err:              nil,
		},
		{
			desc:             "set parent group of client with existing parent group",
			clientID:         cparentedClient.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: cparentedClient,
			err:              svcerr.ErrConflict,
		},
		{
			desc:              "set parent group with failed to retrieve entity",
			clientID:          client.ID,
			parentGroupID:     testsutil.GenerateUUID(t),
			session:           smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp:  client,
			retrieveEntityErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrUpdateEntity,
		},
		{
			desc:             "set parent group with parent group from different domain",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: testsutil.GenerateUUID(t),
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: svcerr.ErrUpdateEntity,
		},
		{
			desc:             "set parent group with disabled parent group",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: validID,
					Status:   uint32(clients.DisabledStatus),
				},
			},
			err: svcerr.ErrUpdateEntity,
		},
		{
			desc:             "set parent group with failed to add policies",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			addPoliciesErr: svcerr.ErrUpdateEntity,
			err:            svcerr.ErrAddPolicies,
		},
		{
			desc:             "set parent group with failed to set parent group",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			setParentGroupErr: svcerr.ErrUpdateEntity,
			err:               svcerr.ErrUpdateEntity,
		},
		{
			desc:             "set parent group with failed to set parent group and failed rollback",
			clientID:         client.ID,
			parentGroupID:    testsutil.GenerateUUID(t),
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: client,
			retrieveEntityResp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       testsutil.GenerateUUID(t),
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			setParentGroupErr: svcerr.ErrUpdateEntity,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		pols := []policysvc.Policy{
			{
				Domain:      tc.session.DomainID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.parentGroupID,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.ClientType,
				Object:      tc.clientID,
			},
		}
		repoCall := repo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResp, tc.retrieveByIDErr)
		groupsCall := gpgRPCClient.On("RetrieveEntity", context.Background(), &grpcCommonV1.RetrieveEntityReq{Id: tc.parentGroupID}).Return(tc.retrieveEntityResp, tc.retrieveEntityErr)
		policyCall := pService.On("AddPolicies", context.Background(), pols).Return(tc.addPoliciesErr)
		policyCall1 := pService.On("DeletePolicies", context.Background(), pols).Return(tc.deletePoliciesErr)
		repoCall2 := repo.On("SetParentGroup", context.Background(), mock.Anything).Return(tc.setParentGroupErr)
		err := svc.SetParentGroup(context.Background(), tc.session, tc.parentGroupID, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		groupsCall.Unset()
		policyCall.Unset()
		repoCall2.Unset()
		policyCall1.Unset()
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc := newService()

	parentedGroup := client
	parentedGroup.ParentGroup = validID

	cases := []struct {
		desc                 string
		clientID             string
		session              smqauthn.Session
		retrieveByIDResp     clients.Client
		retrieveByIDErr      error
		deletePoliciesErr    error
		addPoliciesErr       error
		removeParentGroupErr error
		err                  error
	}{
		{
			desc:             "remove parent group successfully",
			clientID:         parentedGroup.ID,
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: parentedGroup,
			err:              nil,
		},
		{
			desc:             "remove parent group with failed to retrieve client",
			clientID:         parentedGroup.ID,
			session:          smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp: clients.Client{},
			retrieveByIDErr:  svcerr.ErrNotFound,
			err:              svcerr.ErrViewEntity,
		},
		{
			desc:              "remove parent group with failed to delete policies",
			clientID:          parentedGroup.ID,
			session:           smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp:  parentedGroup,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:                 "remove parent group with failed to remove parent group",
			clientID:             parentedGroup.ID,
			session:              smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp:     parentedGroup,
			removeParentGroupErr: svcerr.ErrUpdateEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "remove parent group with failed to remove parent group and failed to add policies",
			clientID:             parentedGroup.ID,
			session:              smqauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID + "_" + validID},
			retrieveByIDResp:     parentedGroup,
			removeParentGroupErr: svcerr.ErrUpdateEntity,
			addPoliciesErr:       svcerr.ErrUpdateEntity,
			err:                  apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		pols := []policysvc.Policy{
			{
				Domain:      tc.session.DomainID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.retrieveByIDResp.ParentGroup,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.ClientType,
				Object:      tc.clientID,
			},
		}
		repoCall := repo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResp, tc.retrieveByIDErr)
		policyCall := pService.On("DeletePolicies", context.Background(), pols).Return(tc.deletePoliciesErr)
		policyCall1 := pService.On("AddPolicies", context.Background(), pols).Return(tc.addPoliciesErr)
		repoCall2 := repo.On("RemoveParentGroup", context.Background(), mock.Anything).Return(tc.removeParentGroupErr)
		err := svc.RemoveParentGroup(context.Background(), tc.session, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
		repoCall2.Unset()
		policyCall1.Unset()
	}
}
