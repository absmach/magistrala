// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	pauth "github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	ID             = "6e5e10b3-d4df-4758-b426-4929d55ad740"
	client         = mgclients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	validToken        = "token"
	inValidToken      = invalid
	valid             = "valid"
	invalid           = "invalid"
	validID           = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID           = testsutil.GenerateUUID(&testing.T{})
	errRemovePolicies = errors.New("failed to delete policies")
)

func newService() (things.Service, *mocks.Repository, *policymocks.PolicyClient, *mocks.Cache) {
	policyClient := new(policymocks.PolicyClient)
	thingCache := new(mocks.Cache)
	idProvider := uuid.NewMock()
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	return things.NewService(policyClient, cRepo, gRepo, thingCache, idProvider), cRepo, policyClient, thingCache
}

func TestCreateThings(t *testing.T) {
	svc, cRepo, policies, _ := newService()

	cases := []struct {
		desc            string
		thing           mgclients.Client
		token           string
		addPolicyErr    error
		deletePolicyErr error
		saveErr         error
		err             error
	}{
		{
			desc:  "create a new thing successfully",
			thing: client,
			token: validToken,
			err:   nil,
		},
		{
			desc:    "create a an existing thing",
			thing:   client,
			token:   validToken,
			saveErr: repoerr.ErrConflict,
			err:     repoerr.ErrConflict,
		},
		{
			desc: "create a new thing without secret",
			thing: mgclients.Client{
				Name: "clientWithoutSecret",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithoutsecret@example.com",
				},
				Status: mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing without identity",
			thing: mgclients.Client{
				Name: "clientWithoutIdentity",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithoutsecret@example.com",
				},
				Status: mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with name",
			thing: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},

		{
			desc: "create a new disabled thing with name",
			thing: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with tags",
			thing: mgclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing with tags",
			thing: mgclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mgclients.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with metadata",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validCMetadata,
				Status:   mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing with metadata",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validCMetadata,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with valid disabled status",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with all fields",
			thing: mgclients.Client{
				Name: "newclientwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: mgclients.Metadata{
					"name": "newclientwithallfields",
				},
				Status: mgclients.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with invalid status",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			token: validToken,
			err:   svcerr.ErrInvalidStatus,
		},
		{
			desc: "create a new thing with failed add policies response",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:        validToken,
			addPolicyErr: svcerr.ErrInvalidPolicy,
			err:          svcerr.ErrInvalidPolicy,
		},
		{
			desc: "create a new thing with failed delete policies response",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:           validToken,
			saveErr:         repoerr.ErrConflict,
			deletePolicyErr: svcerr.ErrInvalidPolicy,
			err:             repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return([]mgclients.Client{tc.thing}, tc.saveErr)
		policyCall := policies.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		policyCall1 := policies.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolicyErr)
		expected, err := svc.CreateThings(context.Background(), pauth.Session{}, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.thing.ID = expected[0].ID
			tc.thing.CreatedAt = expected[0].CreatedAt
			tc.thing.UpdatedAt = expected[0].UpdatedAt
			tc.thing.Credentials.Secret = expected[0].Credentials.Secret
			tc.thing.Domain = expected[0].Domain
			tc.thing.UpdatedBy = expected[0].UpdatedBy
			assert.Equal(t, tc.thing, expected[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.thing, expected[0]))
		}
		repoCall.Unset()
		policyCall.Unset()
		policyCall1.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, _, _ := newService()

	cases := []struct {
		desc        string
		clientID    string
		response    mgclients.Client
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
			response: mgclients.Client{},
			clientID: "",
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:        "view client with valid token and invalid client id",
			response:    mgclients.Client{},
			clientID:    wrongID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: mgclients.Client{},
			clientID: wrongID,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.response, tc.err)
		rClient, err := svc.ViewClient(context.Background(), tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		repoCall1.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo, policies, _ := newService()

	adminID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	nonAdminID := testsutil.GenerateUUID(t)
	client.Permissions = []string{"read", "write"}

	cases := []struct {
		desc                    string
		userKind                string
		session                 pauth.Session
		page                    mgclients.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                mgclients.ClientsPage
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
			session:  pauth.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: []string{"read", "write"},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			err: nil,
		},
		{
			desc:     "list all clients as non admin with failed to retrieve all",
			userKind: "non-admin",
			session:  pauth.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: mgclients.ClientsPage{},
			response:            mgclients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed to list permissions",
			userKind: "non-admin",
			session:  pauth.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: []string{},
			response:                mgclients.ClientsPage{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed super admin",
			userKind: "non-admin",
			session:  pauth.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			response:            mgclients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
		},
		{
			desc:     "list all clients as non admin with failed to list objects",
			userKind: "non-admin",
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			response:            mgclients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		listAllObjectsCall := policies.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := policies.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
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
		session                 pauth.Session
		page                    mgclients.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                mgclients.ClientsPage
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
			session:  pauth.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{client.ID, client.ID}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: []string{"read", "write"},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			err: nil,
		},
		{
			desc:     "list all clients as admin with failed to retrieve all",
			userKind: "admin",
			id:       adminID,
			session:  pauth.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: mgclients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list permissions",
			userKind: "admin",
			id:       adminID,
			session:  pauth.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list clients",
			userKind: "admin",
			id:       adminID,
			session:  pauth.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			retrieveAllResponse: mgclients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		listAllObjectsCall := policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.DomainID + "_" + adminID,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		listAllObjectsCall2 := policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.UserID,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := policies.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
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
	svc, cRepo, _, _ := newService()

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = mgclients.Metadata{"role": "test"}

	cases := []struct {
		desc           string
		client         mgclients.Client
		session        pauth.Session
		updateResponse mgclients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client name successfully",
			client:         client1,
			session:        pauth.Session{UserID: validID},
			updateResponse: client1,
			err:            nil,
		},
		{
			desc:           "update client metadata with valid token",
			client:         client2,
			updateResponse: client2,
			session:        pauth.Session{UserID: validID},
			err:            nil,
		},
		{
			desc:           "update client with failed to update repo",
			client:         client1,
			updateResponse: mgclients.Client{},
			session:        pauth.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	svc, cRepo, _, _ := newService()

	client.Tags = []string{"updated"}

	cases := []struct {
		desc           string
		client         mgclients.Client
		session        pauth.Session
		updateResponse mgclients.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update client tags successfully",
			client:         client,
			session:        pauth.Session{UserID: validID},
			updateResponse: client,
			err:            nil,
		},
		{
			desc:           "update client tags with failed to update repo",
			client:         client,
			updateResponse: mgclients.Client{},
			session:        pauth.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, cRepo, _, _ := newService()

	cases := []struct {
		desc                 string
		client               mgclients.Client
		newSecret            string
		updateSecretResponse mgclients.Client
		session              pauth.Session
		updateErr            error
		err                  error
	}{
		{
			desc:      "update client secret successfully",
			client:    client,
			newSecret: "newSecret",
			session:   pauth.Session{UserID: validID},
			updateSecretResponse: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
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
			session:              pauth.Session{UserID: validID},
			updateSecretResponse: mgclients.Client{},
			updateErr:            repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.session, tc.client.ID, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateSecretResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateSecretResponse, updatedClient))
		repoCall.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	svc, cRepo, _, _ := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mgclients.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              pauth.Session
		client               mgclients.Client
		changeStatusResponse mgclients.Client
		retrieveByIDResponse mgclients.Client
		changeStatusErr      error
		retrieveIDErr        error
		err                  error
	}{
		{
			desc:                 "enable disabled client",
			id:                   disabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: endisabledClient1,
			retrieveByIDResponse: disabledClient1,
			err:                  nil,
		},
		{
			desc:                 "enable disabled client with failed to update repo",
			id:                   disabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: disabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "enable enabled client",
			id:                   enabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: enabledClient1,
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable non-existing client",
			id:                   wrongID,
			session:              pauth.Session{UserID: validID},
			client:               mgclients.Client{},
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.EnableClient(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, _, cache := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              pauth.Session
		client               mgclients.Client
		changeStatusResponse mgclients.Client
		retrieveByIDResponse mgclients.Client
		changeStatusErr      error
		retrieveIDErr        error
		removeErr            error
		err                  error
	}{
		{
			desc:                 "disable enabled client",
			id:                   enabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			err:                  nil,
		},
		{
			desc:                 "disable client with failed to update repo",
			id:                   enabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               enabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: enabledClient1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "disable disabled client",
			id:                   disabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: disabledClient1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable non-existing client",
			id:                   wrongID,
			client:               mgclients.Client{},
			session:              pauth.Session{UserID: validID},
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable client with failed to remove from cache",
			id:                   enabledClient1.ID,
			session:              pauth.Session{UserID: validID},
			client:               disabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			removeErr:            svcerr.ErrRemoveEntity,
			err:                  svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		repoCall2 := cache.On("Remove", mock.Anything, mock.Anything).Return(tc.removeErr)
		_, err := svc.DisableClient(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, policies, _ := newService()

	nClients := uint64(10)
	aClients := []mgclients.Client{}
	domainID := testsutil.GenerateUUID(t)
	for i := uint64(0); i < nClients; i++ {
		identity := fmt.Sprintf("member_%d@example.com", i)
		client := mgclients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: domainID,
			Name:   identity,
			Credentials: mgclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mgclients.Metadata{"role": "client"},
		}
		aClients = append(aClients, client)
	}
	aClients[0].Permissions = []string{"admin"}

	cases := []struct {
		desc                     string
		groupID                  string
		page                     mgclients.Page
		session                  pauth.Session
		listObjectsResponse      policysvc.PolicyPage
		listPermissionsResponse  policysvc.Permissions
		retreiveAllByIDsResponse mgclients.ClientsPage
		response                 mgclients.MembersPage
		identifyErr              error
		authorizeErr             error
		listObjectsErr           error
		listPermissionsErr       error
		retreiveAllByIDsErr      error
		err                      error
	}{
		{
			desc:                    "list members with authorized token",
			session:                 pauth.Session{UserID: validID, DomainID: domainID},
			groupID:                 testsutil.GenerateUUID(t),
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			retreiveAllByIDsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Members: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc:    "list members with offset and limit",
			session: pauth.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mgclients.AllStatus,
			},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			retreiveAllByIDsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: nClients - 6 - 1,
				},
				Clients: aClients[6 : nClients-1],
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: nClients - 6 - 1,
				},
				Members: aClients[6 : nClients-1],
			},
			err: nil,
		},
		{
			desc:                     "list members with an invalid id",
			session:                  pauth.Session{UserID: validID, DomainID: domainID},
			groupID:                  wrongID,
			listObjectsResponse:      policysvc.PolicyPage{},
			listPermissionsResponse:  []string{},
			retreiveAllByIDsResponse: mgclients.ClientsPage{},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			retreiveAllByIDsErr: svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:    "list members with permissions",
			session: pauth.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{"admin"},
			retreiveAllByIDsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{aClients[0]},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{aClients[0]},
			},
			err: nil,
		},
		{
			desc:    "list members with failed to list objects",
			session: pauth.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:    "list members with failed to list permissions",
			session: pauth.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			retreiveAllByIDsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{aClients[0]},
			},
			response:                mgclients.MembersPage{},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.retreiveAllByIDsResponse, tc.retreiveAllByIDsErr)
		repoCall1 := policies.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClientsByGroup(context.Background(), tc.session, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		policyCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDeleteClient(t *testing.T) {
	svc, cRepo, policies, cache := newService()

	client := mgclients.Client{
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
		policyCall := policies.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyErr)
		repoCall1 := cRepo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		err := svc.DeleteClient(context.Background(), tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
		repoCall1.Unset()
	}
}

func TestShare(t *testing.T) {
	svc, _, policies, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc           string
		session        pauth.Session
		clientID       string
		relation       string
		userID         string
		addPoliciesErr error
		err            error
	}{
		{
			desc:     "share thing successfully",
			session:  pauth.Session{UserID: validID, DomainID: validID},
			clientID: clientID,
			err:      nil,
		},
		{
			desc:           "share thing with failed to add policies",
			session:        pauth.Session{UserID: validID, DomainID: validID},
			clientID:       clientID,
			addPoliciesErr: svcerr.ErrInvalidPolicy,
			err:            svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
		err := svc.Share(context.Background(), tc.session, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}
}

func TestUnShare(t *testing.T) {
	svc, _, policies, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc              string
		session           pauth.Session
		clientID          string
		relation          string
		userID            string
		deletePoliciesErr error
		err               error
	}{
		{
			desc:     "unshare thing successfully",
			session:  pauth.Session{UserID: validID, DomainID: validID},
			clientID: clientID,
			err:      nil,
		},
		{
			desc:              "share thing with failed to delete policies",
			session:           pauth.Session{UserID: validID, DomainID: validID},
			clientID:          clientID,
			deletePoliciesErr: svcerr.ErrInvalidPolicy,
			err:               svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		err := svc.Unshare(context.Background(), tc.session, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}
}

func TestViewClientPerms(t *testing.T) {
	svc, _, policies, _ := newService()

	validID := valid

	cases := []struct {
		desc             string
		session          pauth.Session
		thingID          string
		listPermResponse policysvc.Permissions
		listPermErr      error
		err              error
	}{
		{
			desc:             "view client permissions successfully",
			session:          pauth.Session{UserID: validID, DomainID: validID},
			thingID:          validID,
			listPermResponse: policysvc.Permissions{"admin"},
			err:              nil,
		},
		{
			desc:             "view permissions with failed retrieve list permissions response",
			session:          pauth.Session{UserID: validID, DomainID: validID},
			thingID:          validID,
			listPermResponse: []string{},
			listPermErr:      svcerr.ErrAuthorization,
			err:              svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("ListPermissions", mock.Anything, mock.Anything, []string{}).Return(tc.listPermResponse, tc.listPermErr)
		res, err := svc.ViewClientPerms(context.Background(), tc.session, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.ElementsMatch(t, tc.listPermResponse, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.listPermResponse, res))
		}
		policyCall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	svc, cRepo, _, cache := newService()

	valid := valid

	cases := []struct {
		desc                string
		key                 string
		cacheIDResponse     string
		cacheIDErr          error
		repoIDResponse      mgclients.Client
		retrieveBySecretErr error
		saveErr             error
		err                 error
	}{
		{
			desc:            "identify client with valid key from cache",
			key:             valid,
			cacheIDResponse: client.ID,
			err:             nil,
		},
		{
			desc:            "identify client with valid key from repo",
			key:             valid,
			cacheIDResponse: "",
			cacheIDErr:      repoerr.ErrNotFound,
			repoIDResponse:  client,
			err:             nil,
		},
		{
			desc:                "identify client with invalid key",
			key:                 invalid,
			cacheIDResponse:     "",
			cacheIDErr:          repoerr.ErrNotFound,
			repoIDResponse:      mgclients.Client{},
			retrieveBySecretErr: repoerr.ErrNotFound,
			err:                 repoerr.ErrNotFound,
		},
		{
			desc:            "identify client with failed to save to cache",
			key:             valid,
			cacheIDResponse: "",
			cacheIDErr:      repoerr.ErrNotFound,
			repoIDResponse:  client,
			saveErr:         errors.ErrMalformedEntity,
			err:             svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cache.On("ID", mock.Anything, tc.key).Return(tc.cacheIDResponse, tc.cacheIDErr)
		repoCall1 := cRepo.On("RetrieveBySecret", mock.Anything, mock.Anything).Return(tc.repoIDResponse, tc.retrieveBySecretErr)
		repoCall2 := cache.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(tc.saveErr)
		_, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}
