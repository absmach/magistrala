// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	policysvc "github.com/absmach/magistrala/pkg/policy"
	policymocks "github.com/absmach/magistrala/pkg/policy/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func newService() (things.Service, *mocks.Repository, *authmocks.AuthServiceClient, *policymocks.PolicyClient, *mocks.Cache) {
	auth := new(authmocks.AuthServiceClient)
	policyClient := new(policymocks.PolicyClient)
	thingCache := new(mocks.Cache)
	idProvider := uuid.NewMock()
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	return things.NewService(auth, policyClient, cRepo, gRepo, thingCache, idProvider), cRepo, auth, policyClient, thingCache
}

func TestCreateThings(t *testing.T) {
	svc, cRepo, auth, policy, _ := newService()

	cases := []struct {
		desc            string
		thing           mgclients.Client
		token           string
		authResponse    *magistrala.AuthorizeRes
		authorizeErr    error
		identifyErr     error
		addPolicyErr    error
		deletePolicyErr error
		saveErr         error
		err             error
	}{
		{
			desc:         "create a new thing successfully",
			thing:        client,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "create a an existing thing",
			thing:        client,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			saveErr:      repoerr.ErrConflict,
			err:          repoerr.ErrConflict,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:        validToken,
			err:          nil,
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
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:        validToken,
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc: "create a new disabled thing",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
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
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          svcerr.ErrInvalidStatus,
		},
		{
			desc: "create a new thing with invalid token",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidtoken@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:       inValidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "create a new thing by unathorized user",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithunathorizeduser@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc: "create a new thing with failed add policy response",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			addPolicyErr: svcerr.ErrInvalidPolicy,
			err:          svcerr.ErrInvalidPolicy,
		},
		{
			desc: "create a new thing with failed delete policy response",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			token:           validToken,
			authResponse:    &magistrala.AuthorizeRes{Authorized: true},
			saveErr:         repoerr.ErrConflict,
			deletePolicyErr: svcerr.ErrInvalidPolicy,
			err:             repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, tc.identifyErr)
		authcall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return([]mgclients.Client{tc.thing}, tc.saveErr)
		authCall1 := policy.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		authCall2 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolicyErr)
		expected, err := svc.CreateThings(context.Background(), tc.token, tc.thing)
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
		authcall.Unset()
		repoCall1.Unset()
		authCall1.Unset()
		authCall2.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, auth, _, _ := newService()

	cases := []struct {
		desc              string
		token             string
		clientID          string
		response          mgclients.Client
		authorizeResponse *magistrala.AuthorizeRes
		authorizeErr      error
		retrieveErr       error
		err               error
	}{
		{
			desc:              "view client successfully",
			response:          client,
			token:             validToken,
			clientID:          client.ID,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:               nil,
		},
		{
			desc:              "view client with an invalid token",
			response:          mgclients.Client{},
			token:             inValidToken,
			clientID:          "",
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "view client with valid token and invalid client id",
			response:          mgclients.Client{},
			token:             validToken,
			clientID:          wrongID,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:       svcerr.ErrNotFound,
			err:               svcerr.ErrNotFound,
		},
		{
			desc:              "view client with an invalid token and invalid client id",
			response:          mgclients.Client{},
			token:             inValidToken,
			clientID:          wrongID,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.response, tc.err)
		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo, auth, policy, _ := newService()

	adminID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	nonAdminID := testsutil.GenerateUUID(t)
	client.Permissions = []string{"read", "write"}

	cases := []struct {
		desc                    string
		userKind                string
		token                   string
		page                    mgclients.Page
		identifyResponse        *magistrala.IdentityRes
		authorizeResponse       *magistrala.AuthorizeRes
		authorizeResponse1      *magistrala.AuthorizeRes
		authorizeResponse2      *magistrala.AuthorizeRes
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                mgclients.ClientsPage
		id                      string
		size                    uint64
		identifyErr             error
		authorizeErr            error
		authorizeErr1           error
		authorizeErr2           error
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all clients successfully as non admin",
			userKind: "non-admin",
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			authorizeResponse2:  &magistrala.AuthorizeRes{Authorized: true},
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
			desc:     "list all clients as non admin with invalid token",
			userKind: "non-admin",
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			token:            inValidToken,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:     "list all clients as non admin with empty domain id",
			userKind: "non-admin",
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			token:            validToken,
			identifyResponse: &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: ""},
			err:              svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "list all clients as non admin with failed to retrieve all",
			userKind: "non-admin",
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{},
			response:            mgclients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as non admin with failed to list permissions",
			userKind: "non-admin",
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:   &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: true},
			authorizeResponse2: &magistrala.AuthorizeRes{Authorized: true},
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
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			authorizeResponse1:  &magistrala.AuthorizeRes{Authorized: true},
			response:            mgclients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
		},
		{
			desc:     " list all clients as non admin with failed super admin and failed authorization",
			userKind: "non-admin",
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			authorizeResponse1:  &magistrala.AuthorizeRes{Authorized: false},
			response:            mgclients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc:     "list all clients as non admin with failed to list objects",
			userKind: "non-admin",
			token:    validToken,
			id:       nonAdminID,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			authorizeResponse1:  &magistrala.AuthorizeRes{Authorized: true},
			response:            mgclients.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authorizeCall := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.identifyResponse.UserId,
			Permission:  policysvc.AdminPermission,
			ObjectType:  policysvc.PlatformType,
			Object:      policysvc.MagistralaObject,
		}).Return(tc.authorizeResponse, tc.authorizeErr)
		authorizeCall2 := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			Domain:      "",
			SubjectType: policysvc.UserType,
			SubjectKind: policysvc.UsersKind,
			Subject:     tc.identifyResponse.UserId,
			Permission:  "membership",
			ObjectType:  "domain",
			Object:      tc.identifyResponse.DomainId,
		}).Return(tc.authorizeResponse1, tc.authorizeErr1)
		listAllObjectsCall := policy.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := policy.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)

		page, err := svc.ListClients(context.Background(), tc.token, tc.id, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		authorizeCall.Unset()
		authorizeCall2.Unset()
		listAllObjectsCall.Unset()
		retrieveAllCall.Unset()
		listPermissionsCall.Unset()
	}

	cases2 := []struct {
		desc                    string
		userKind                string
		token                   string
		page                    mgclients.Page
		identifyResponse        *magistrala.IdentityRes
		authorizeResponse       *magistrala.AuthorizeRes
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                mgclients.ClientsPage
		id                      string
		size                    uint64
		identifyErr             error
		authorizeErr            error
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all clients as admin successfully",
			userKind: "admin",
			id:       adminID,
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
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
			desc:     "list all clients as admin with unauthorized user",
			userKind: "admin",
			id:       adminID,
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:  &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:     "list all clients as admin with failed to retrieve all",
			userKind: "admin",
			id:       adminID,
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: mgclients.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list permissions",
			userKind: "admin",
			id:       adminID,
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
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
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list things",
			userKind: "admin",
			id:       adminID,
			token:    validToken,
			page: mgclients.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authorizeCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		listAllObjectsCall := policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.identifyResponse.DomainId + "_" + adminID,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		listAllObjectsCall2 := policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.identifyResponse.Id,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := policy.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)

		page, err := svc.ListClients(context.Background(), tc.token, tc.id, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		authorizeCall.Unset()
		listAllObjectsCall.Unset()
		listAllObjectsCall2.Unset()
		retrieveAllCall.Unset()
		listPermissionsCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc, cRepo, auth, _, _ := newService()

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = mgclients.Metadata{"role": "test"}

	cases := []struct {
		desc              string
		client            mgclients.Client
		updateResponse    mgclients.Client
		authorizeResponse *magistrala.AuthorizeRes
		authorizeErr      error
		updateErr         error
		token             string
		err               error
	}{
		{
			desc:              "update client name successfully",
			client:            client1,
			updateResponse:    client1,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			err:               nil,
		},
		{
			desc:              "update client name with invalid token",
			client:            client1,
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             inValidToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc: "update client name with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "Updated Client",
			},
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "update client metadata with valid token",
			client:            client2,
			updateResponse:    client2,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			err:               nil,
		},
		{
			desc:              "update client metadata with invalid token",
			client:            client2,
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			token:             inValidToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc: "update client metadata with invalid ID",
			client: mgclients.Client{
				ID:       wrongID,
				Metadata: mgclients.Metadata{"role": "test"},
			},
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "update client with failed to update repo",
			client:            client1,
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:         repoerr.ErrMalformedEntity,
			token:             validToken,
			err:               svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	svc, cRepo, auth, _, _ := newService()

	client.Tags = []string{"updated"}

	cases := []struct {
		desc              string
		client            mgclients.Client
		updateResponse    mgclients.Client
		authorizeResponse *magistrala.AuthorizeRes
		authorizeErr      error
		updateErr         error
		token             string
		err               error
	}{
		{
			desc:              "update client tags successfully",
			client:            client,
			updateResponse:    client,
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			err:               nil,
		},
		{
			desc:              "update client tags with invalid token",
			client:            client,
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			token:             inValidToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc: "update client tags with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "Updated name",
			},
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			token:             validToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "update client tags with failed to update repo",
			client:            client,
			updateResponse:    mgclients.Client{},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:         repoerr.ErrMalformedEntity,
			token:             validToken,
			err:               svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, cRepo, auth, _, _ := newService()

	cases := []struct {
		desc                 string
		client               mgclients.Client
		newSecret            string
		updateSecretResponse mgclients.Client
		authorizeResponse    *magistrala.AuthorizeRes
		token                string
		updateErr            error
		authorizeErr         error
		err                  error
	}{
		{
			desc:      "update client secret successfully",
			client:    client,
			newSecret: "newSecret",
			updateSecretResponse: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: client.Credentials.Identity,
					Secret:   "newSecret",
				},
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			err:               nil,
		},
		{
			desc:                 "update client secret with invalid token",
			client:               client,
			newSecret:            "newSecret",
			updateSecretResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: false},
			token:                inValidToken,
			authorizeErr:         svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc: "update client secret with invalid ID",
			client: mgclients.Client{
				ID: wrongID,
			},
			newSecret:            "newSecret",
			updateSecretResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: false},
			token:                validToken,
			authorizeErr:         svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:                 "update client secret with failed to update repo",
			client:               client,
			newSecret:            "newSecret",
			updateSecretResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			updateErr:            repoerr.ErrMalformedEntity,
			token:                validToken,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateErr)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.client.ID, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateSecretResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateSecretResponse, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	svc, cRepo, auth, _, _ := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mgclients.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		token                string
		client               mgclients.Client
		changeStatusResponse mgclients.Client
		retrieveByIDResponse mgclients.Client
		authorizeResponse    *magistrala.AuthorizeRes
		changeStatusErr      error
		retrieveIDErr        error
		authorizeErr         error
		err                  error
	}{
		{
			desc:                 "enable disabled client",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			changeStatusResponse: endisabledClient1,
			retrieveByIDResponse: disabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			err:                  nil,
		},
		{
			desc:                 "enable disabled client with failed to update repo",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: disabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "enable enabled client",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			changeStatusResponse: enabledClient1,
			retrieveByIDResponse: enabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable non-existing client",
			id:                   wrongID,
			token:                validToken,
			client:               mgclients.Client{},
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "enable client with invalid token",
			id:                   enabledClient1.ID,
			token:                inValidToken,
			client:               enabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:         svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}

	cases2 := []struct {
		desc     string
		status   mgclients.Status
		size     uint64
		response mgclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus,
			size:   2,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, endisabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus,
			size:   1,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus,
			size:   3,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, disabledClient1, endisabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mgclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall3 := cRepo.On("SearchClients", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall3.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, auth, _, cache := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		token                string
		client               mgclients.Client
		changeStatusResponse mgclients.Client
		retrieveByIDResponse mgclients.Client
		authorizeResponse    *magistrala.AuthorizeRes
		changeStatusErr      error
		retrieveIDErr        error
		authorizeErr         error
		removeErr            error
		err                  error
	}{
		{
			desc:                 "disable enabled client",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			err:                  nil,
		},
		{
			desc:                 "disable client with failed to update repo",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: enabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "disable disabled client",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: disabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable non-existing client",
			id:                   wrongID,
			client:               mgclients.Client{},
			token:                validToken,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable client with invalid token",
			id:                   disabledClient1.ID,
			token:                inValidToken,
			client:               disabledClient1,
			changeStatusResponse: mgclients.Client{},
			retrieveByIDResponse: mgclients.Client{},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:         svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:                 "disable client with failed to remove from cache",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledClient1,
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			removeErr:            svcerr.ErrRemoveEntity,
			err:                  svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		repoCall3 := cache.On("Remove", mock.Anything, mock.Anything).Return(tc.removeErr)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}

	cases2 := []struct {
		desc     string
		status   mgclients.Status
		size     uint64
		response mgclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus,
			size:   1,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus,
			size:   2,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{disenabledClient1, disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus,
			size:   3,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, disabledClient1, disenabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mgclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall3 := cRepo.On("SearchClients", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall3.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, auth, policy, _ := newService()

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
		token                    string
		groupID                  string
		page                     mgclients.Page
		identifyResponse         *magistrala.IdentityRes
		authorizeResponse        *magistrala.AuthorizeRes
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
			token:                   validToken,
			groupID:                 testsutil.GenerateUUID(t),
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
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
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mgclients.AllStatus,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
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
			desc:             "list members with an invalid token",
			token:            inValidToken,
			groupID:          testsutil.GenerateUUID(t),
			identifyResponse: &magistrala.IdentityRes{},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:                     "list members with an invalid id",
			token:                    validToken,
			groupID:                  wrongID,
			identifyResponse:         &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:        &magistrala.AuthorizeRes{Authorized: true},
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
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
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
			desc:    "list members with unauthorized user",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:    "list members with failed to list objects",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:    "list members with failed to list permissions",
			token:   validToken,
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
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := policy.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.retreiveAllByIDsResponse, tc.retreiveAllByIDsErr)
		repoCall4 := policy.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClientsByGroup(context.Background(), tc.token, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestDeleteClient(t *testing.T) {
	svc, cRepo, auth, policy, cache := newService()

	client := mgclients.Client{
		ID: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc              string
		token             string
		identifyResponse  *magistrala.IdentityRes
		authorizeResponse *magistrala.AuthorizeRes
		clientID          string
		identifyErr       error
		authorizeErr      error
		removeErr         error
		deleteErr         error
		deletePolicyErr   error
		err               error
	}{
		{
			desc:              "Delete client with authorized token",
			token:             validToken,
			clientID:          client.ID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:               nil,
		},
		{
			desc:             "Delete client with unauthorized token",
			token:            inValidToken,
			clientID:         client.ID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "Delete invalid client",
			token:             validToken,
			clientID:          wrongID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "Delete client with repo error ",
			token:             validToken,
			clientID:          client.ID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			deleteErr:         repoerr.ErrRemoveEntity,
			err:               repoerr.ErrRemoveEntity,
		},
		{
			desc:              "Delete client with cache error ",
			token:             validToken,
			clientID:          client.ID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			removeErr:         svcerr.ErrRemoveEntity,
			err:               repoerr.ErrRemoveEntity,
		},
		{
			desc:              "Delete client with failed to delete policy",
			token:             validToken,
			clientID:          client.ID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyErr:   errRemovePolicies,
			err:               errRemovePolicies,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cache.On("Remove", mock.Anything, tc.clientID).Return(tc.removeErr)
		repoCall3 := policy.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyErr)
		repoCall4 := cRepo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		err := svc.DeleteClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestShare(t *testing.T) {
	svc, _, auth, policy, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc              string
		token             string
		clientID          string
		relation          string
		userID            string
		identifyResponse  *magistrala.IdentityRes
		authorizeResponse *magistrala.AuthorizeRes
		identifyErr       error
		authorizeErr      error
		addPoliciesErr    error
		err               error
	}{
		{
			desc:              "share thing successfully",
			token:             validToken,
			clientID:          clientID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:               nil,
		},
		{
			desc:             "share thing with invalid token",
			token:            inValidToken,
			clientID:         clientID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "share thing with invalid ID",
			token:             validToken,
			clientID:          invalid,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "share thing with failed to add policies",
			token:             validToken,
			clientID:          clientID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			addPoliciesErr:    svcerr.ErrInvalidPolicy,
			err:               svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := policy.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
		err := svc.Share(context.Background(), tc.token, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUnShare(t *testing.T) {
	svc, _, auth, policy, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc              string
		token             string
		clientID          string
		relation          string
		userID            string
		identifyResponse  *magistrala.IdentityRes
		authorizeResponse *magistrala.AuthorizeRes
		identifyErr       error
		authorizeErr      error
		deletePoliciesErr error
		err               error
	}{
		{
			desc:              "unshare thing successfully",
			token:             validToken,
			clientID:          clientID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:               nil,
		},
		{
			desc:             "unshare thing with invalid token",
			token:            inValidToken,
			clientID:         clientID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "unshare thing with invalid ID",
			token:             validToken,
			clientID:          invalid,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "share thing with failed to delete policies",
			token:             validToken,
			clientID:          clientID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			deletePoliciesErr: svcerr.ErrInvalidPolicy,
			err:               svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		err := svc.Unshare(context.Background(), tc.token, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestViewClientPerms(t *testing.T) {
	svc, _, auth, policy, _ := newService()

	validID := valid

	cases := []struct {
		desc              string
		token             string
		thingID           string
		identifyResponse  *magistrala.IdentityRes
		authorizeResponse *magistrala.AuthorizeRes
		listPermResponse  policysvc.Permissions
		identifyErr       error
		authorizeErr      error
		listPermErr       error
		err               error
	}{
		{
			desc:              "view client permissions successfully",
			token:             validToken,
			thingID:           validID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listPermResponse:  policysvc.Permissions{"admin"},
			err:               nil,
		},
		{
			desc:             "view client permissions with invalid token",
			token:            inValidToken,
			thingID:          validID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "view client permissions with invalid ID",
			token:             validToken,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "view permissions with failed retrieve list permissions response",
			token:             validToken,
			thingID:           validID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listPermResponse:  []string{},
			listPermErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := policy.On("ListPermissions", mock.Anything, mock.Anything, []string{}).Return(tc.listPermResponse, tc.listPermErr)
		res, err := svc.ViewClientPerms(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.ElementsMatch(t, tc.listPermResponse, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.listPermResponse, res))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestIdentify(t *testing.T) {
	svc, cRepo, _, _, cache := newService()

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

func TestAuthorize(t *testing.T) {
	svc, cRepo, auth, _, cache := newService()

	cases := []struct {
		desc                string
		request             *magistrala.AuthorizeReq
		cacheIDRes          string
		cacheIDErr          error
		retrieveBySecretRes mgclients.Client
		retrieveBySecretErr error
		cacheSaveErr        error
		authorizeRes        *magistrala.AuthorizeRes
		authErr             error
		id                  string
		err                 error
	}{
		{
			desc:                "authorize client with valid key not in cache",
			request:             &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: mgclients.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			authorizeRes:        &magistrala.AuthorizeRes{Authorized: true},
			authErr:             nil,
			id:                  valid,
			err:                 nil,
		},
		{
			desc:         "authorize client with valid key in cache",
			request:      &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:   valid,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			id:           valid,
		},
		{
			desc:                "authorize client with invalid key not in cache for non existing client",
			request:             &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: mgclients.Client{},
			retrieveBySecretErr: repoerr.ErrNotFound,
			err:                 repoerr.ErrNotFound,
		},
		{
			desc:                "authorize client with valid key not in cache with failed to save to cache",
			request:             &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: mgclients.Client{ID: valid},
			cacheSaveErr:        errors.ErrMalformedEntity,
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc:                "authorize client with valid key not in cache and failed to authorize",
			request:             &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: mgclients.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			authorizeRes:        &magistrala.AuthorizeRes{},
			authErr:             svcerr.ErrAuthorization,
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc:                "authorize client with valid key not in cache and not authorize",
			request:             &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: mgclients.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			authorizeRes:        &magistrala.AuthorizeRes{Authorized: false},
			authErr:             nil,
			err:                 svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		cacheCall := cache.On("ID", context.Background(), tc.request.GetSubject()).Return(tc.cacheIDRes, tc.cacheIDErr)
		repoCall := cRepo.On("RetrieveBySecret", context.Background(), tc.request.GetSubject()).Return(tc.retrieveBySecretRes, tc.retrieveBySecretErr)
		cacheCall1 := cache.On("Save", context.Background(), tc.request.GetSubject(), tc.retrieveBySecretRes.ID).Return(tc.cacheSaveErr)
		authCall := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authErr)
		id, err := svc.Authorize(context.Background(), tc.request)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, id))
		}
		cacheCall.Unset()
		cacheCall1.Unset()
		repoCall.Unset()
		authCall.Unset()
	}
}
