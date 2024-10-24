// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"context"
	"fmt"
	"testing"

	chmocks "github.com/absmach/magistrala/channels/mocks"
	"github.com/absmach/magistrala/clients"
	climocks "github.com/absmach/magistrala/clients/mocks"
	gpmocks "github.com/absmach/magistrala/groups/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
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
	validToken        = "token"
	valid             = "valid"
	invalid           = "invalid"
	validID           = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID           = testsutil.GenerateUUID(&testing.T{})
	errRemovePolicies = errors.New("failed to delete policies")
)

var (
	pService   *policymocks.Service
	pEvaluator *policymocks.Evaluator
	cache      *climocks.Cache
	repo       *climocks.Repository
)

func newService() clients.Service {
	pService = new(policymocks.Service)
	cache = new(climocks.Cache)
	idProvider := uuid.NewMock()
	sidProvider := uuid.NewMock()
	repo = new(climocks.Repository)
	chgRPCClient := new(chmocks.ChannelsServiceClient)
	gpgRPCClient := new(gpmocks.GroupsServiceClient)
	tsv, _ := clients.NewService(repo, pService, cache, chgRPCClient, gpgRPCClient, idProvider, sidProvider)
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
		policyCall := pService.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		policyCall1 := pService.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolicyErr)
		expected, err := svc.CreateClients(context.Background(), mgauthn.Session{}, tc.client)
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
		rClient, err := svc.View(context.Background(), mgauthn.Session{}, tc.clientID)
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
	client.Permissions = []string{"read", "write"}

	cases := []struct {
		desc                    string
		userKind                string
		session                 mgauthn.Session
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
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
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
			listPermissionsResponse: []string{"read", "write"},
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
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: clients.ClientsPage{},
			response:            clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed to list permissions",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
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
			listPermissionsResponse: []string{},
			response:                clients.ClientsPage{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed super admin",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
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
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			response:            clients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		listAllObjectsCall := pService.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := repo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := pService.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClients(context.Background(), tc.session, tc.id, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		listAllObjectsCall.Unset()
		retrieveAllCall.Unset()
		listPermissionsCall.Unset()
	}

	cases2 := []struct {
		desc                    string
		userKind                string
		session                 mgauthn.Session
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
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
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
			listPermissionsResponse: []string{"read", "write"},
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
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list permissions",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: clients.ClientsPage{
				Page: clients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []clients.Client{client, client},
			},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list clients",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: clients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			retrieveAllResponse: clients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		listAllObjectsCall := pService.On("ListAllObjects", context.Background(), policysvc.Policy{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.DomainID + "_" + adminID,
			Permission:  "",
			ObjectType:  policysvc.ClientType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		listAllObjectsCall2 := pService.On("ListAllObjects", context.Background(), policysvc.Policy{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.UserID,
			Permission:  "",
			ObjectType:  policysvc.ClientType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := repo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := pService.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClients(context.Background(), tc.session, tc.id, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		listAllObjectsCall.Unset()
		listAllObjectsCall2.Unset()
		retrieveAllCall.Unset()
		listPermissionsCall.Unset()
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
		session        mgauthn.Session
		updateResponse clients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client name successfully",
			client:         client1,
			session:        mgauthn.Session{UserID: validID},
			updateResponse: client1,
			err:            nil,
		},
		{
			desc:           "update client metadata with valid token",
			client:         client2,
			updateResponse: client2,
			session:        mgauthn.Session{UserID: validID},
			err:            nil,
		},
		{
			desc:           "update client with failed to update repo",
			client:         client1,
			updateResponse: clients.Client{},
			session:        mgauthn.Session{UserID: validID},
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
		session        mgauthn.Session
		updateResponse clients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client tags successfully",
			client:         client,
			session:        mgauthn.Session{UserID: validID},
			updateResponse: client,
			err:            nil,
		},
		{
			desc:           "update client tags with failed to update repo",
			client:         client,
			updateResponse: clients.Client{},
			session:        mgauthn.Session{UserID: validID},
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
		session              mgauthn.Session
		updateErr            error
		err                  error
	}{
		{
			desc:      "update client secret successfully",
			client:    client,
			newSecret: "newSecret",
			session:   mgauthn.Session{UserID: validID},
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
			session:              mgauthn.Session{UserID: validID},
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
		session              mgauthn.Session
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
			session:              mgauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: endisabledClient1,
			retrieveByIDResponse: disabledClient1,
			err:                  nil,
		},
		{
			desc:                 "enable disabled client with failed to update repo",
			id:                   disabledClient1.ID,
			session:              mgauthn.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: disabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "enable enabled client",
			id:                   enabledClient1.ID,
			session:              mgauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: enabledClient1,
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable non-existing client",
			id:                   wrongID,
			session:              mgauthn.Session{UserID: validID},
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
		session              mgauthn.Session
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
			session:              mgauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			err:                  nil,
		},
		{
			desc:                 "disable client with failed to update repo",
			id:                   enabledClient1.ID,
			session:              mgauthn.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "disable disabled client",
			id:                   disabledClient1.ID,
			session:              mgauthn.Session{UserID: validID},
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
			session:              mgauthn.Session{UserID: validID},
			changeStatusResponse: clients.Client{},
			retrieveByIDResponse: clients.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable client with failed to remove from cache",
			id:                   enabledClient1.ID,
			session:              mgauthn.Session{UserID: validID},
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
		desc            string
		clientID        string
		removeErr       error
		deleteErr       error
		deletePolicyErr error
		err             error
	}{
		{
			desc:     "Delete client successfully",
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:      "Delete non-existing client",
			clientID:  wrongID,
			deleteErr: repoerr.ErrNotFound,
			err:       svcerr.ErrRemoveEntity,
		},
		{
			desc:      "Delete client with repo error ",
			clientID:  client.ID,
			deleteErr: repoerr.ErrRemoveEntity,
			err:       repoerr.ErrRemoveEntity,
		},
		{
			desc:      "Delete client with cache error ",
			clientID:  client.ID,
			removeErr: svcerr.ErrRemoveEntity,
			err:       repoerr.ErrRemoveEntity,
		},
		{
			desc:            "Delete client with failed to delete policies",
			clientID:        client.ID,
			deletePolicyErr: errRemovePolicies,
			err:             errRemovePolicies,
		},
	}

	for _, tc := range cases {
		repoCall := cache.On("Remove", mock.Anything, tc.clientID).Return(tc.removeErr)
		policyCall := pService.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyErr)
		repoCall1 := repo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		err := svc.Delete(context.Background(), mgauthn.Session{}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
		repoCall1.Unset()
	}
}
