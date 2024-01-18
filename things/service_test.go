// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	authsvc "github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
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
	ID             = testsutil.GenerateUUID(&testing.T{})
	client         = mgclients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	adminEmail        = "admin@example.com"
	validToken        = "token"
	inValidToken      = invalid
	valid             = "valid"
	invalid           = "invalid"
	validID           = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID           = testsutil.GenerateUUID(&testing.T{})
	errAddPolicies    = errors.New("failed to add policies")
	errRemovePolicies = errors.New("failed to remove the policies")
)

func newService() (things.Service, *mocks.Repository, *authmocks.Service, *mocks.Cache) {
	auth := new(authmocks.Service)
	thingCache := new(mocks.Cache)
	idProvider := uuid.NewMock()
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	return things.NewService(auth, cRepo, gRepo, thingCache, idProvider), cRepo, auth, thingCache
}

func TestCreateThings(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	cases := []struct {
		desc              string
		thing             mgclients.Client
		token             string
		authResponse      *magistrala.AuthorizeRes
		addPolicyResponse *magistrala.AddPoliciesRes
		authorizeErr      error
		identifyErr       error
		addPolicyErr      error
		saveErr           error
		err               error
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
			saveErr:      errors.ErrConflict,
			err:          svcerr.ErrConflict,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
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
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			token:             validToken,
			err:               nil,
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
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			token:             validToken,
			err:               nil,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
		},
		{
			desc: "create a new disabled thing",
			thing: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               nil,
		},
		{
			desc: "create a new thing with invalid owner",
			thing: mgclients.Client{
				Owner: wrongID,
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidowner@example.com",
					Secret:   secret,
				},
			},
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			saveErr:           repoerr.ErrMalformedEntity,
			err:               repoerr.ErrCreateEntity,
		},
		{
			desc: "create a new thing with empty secret",
			thing: mgclients.Client{
				Owner: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "newclientwithemptysecret@example.com",
				},
			},
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			saveErr:           repoerr.ErrMissingSecret,
			err:               repoerr.ErrCreateEntity,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:               svcerr.ErrInvalidStatus,
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
			token:             validToken,
			authResponse:      &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPoliciesRes{Authorized: false},
			addPolicyErr:      svcerr.ErrInvalidPolicy,
			err:               svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("Save", context.Background(), mock.Anything).Return([]mgclients.Client{tc.thing}, tc.saveErr)
		repoCall3 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyResponse, tc.addPolicyErr)
		expected, err := svc.CreateThings(context.Background(), tc.token, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.thing.ID = expected[0].ID
			tc.thing.CreatedAt = expected[0].CreatedAt
			tc.thing.UpdatedAt = expected[0].UpdatedAt
			tc.thing.Credentials.Secret = expected[0].Credentials.Secret
			tc.thing.Owner = expected[0].Owner
			tc.thing.UpdatedBy = expected[0].UpdatedBy
			assert.Equal(t, tc.thing, expected[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.thing, expected[0]))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, auth, _ := newService()

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
			token:             authmocks.InvalidValue,
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
	svc, cRepo, auth, _ := newService()

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
		listObjectsResponse     *magistrala.ListObjectsRes
		listObjectsResponse1    *magistrala.ListObjectsRes
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse *magistrala.ListPermissionsRes
		response                mgclients.ClientsPage
		id                      string
		size                    uint64
		identifyErr             error
		authorizeErr            error
		authorizeErr1           error
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
			listObjectsResponse: &magistrala.ListObjectsRes{},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{
				Permissions: []string{"read", "write"},
			},
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
			err:              errors.ErrDomainAuthorization,
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
			identifyResponse:  &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
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
			listObjectsResponse: &magistrala.ListObjectsRes{},
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
			listObjectsResponse: &magistrala.ListObjectsRes{},
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
			listObjectsResponse: &magistrala.ListObjectsRes{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authorizeCall := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: authsvc.UserType,
			Subject:     tc.identifyResponse.UserId,
			Permission:  authsvc.AdminPermission,
			ObjectType:  authsvc.PlatformType,
			Object:      authsvc.MagistralaObject,
		}).Return(tc.authorizeResponse, tc.authorizeErr)
		authorizeCall2 := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			Domain:      "",
			SubjectType: authsvc.UserType,
			SubjectKind: authsvc.UsersKind,
			Subject:     tc.identifyResponse.UserId,
			Permission:  "membership",
			ObjectType:  "domain",
			Object:      tc.identifyResponse.DomainId,
		}).Return(tc.authorizeResponse1, tc.authorizeErr1)
		listAllObjectsCall := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)

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
		listObjectsResponse     *magistrala.ListObjectsRes
		listObjectsResponse1    *magistrala.ListObjectsRes
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse *magistrala.ListPermissionsRes
		response                mgclients.ClientsPage
		id                      string
		size                    uint64
		identifyErr             error
		authorizeErr            error
		listObjectsErr          error
		listObjectsErr1         error
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
			},
			identifyResponse:     &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:  &magistrala.ListObjectsRes{Policies: []string{"test", "test"}},
			listObjectsResponse1: &magistrala.ListObjectsRes{Policies: []string{"test", "test"}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{
				Permissions: []string{"read", "write"},
			},
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
			},
			identifyResponse:     &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:  &magistrala.ListObjectsRes{},
			listObjectsResponse1: &magistrala.ListObjectsRes{},
			retrieveAllResponse:  mgclients.ClientsPage{},
			retrieveAllErr:       repoerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
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
			},
			identifyResponse:     &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:  &magistrala.ListObjectsRes{},
			listObjectsResponse1: &magistrala.ListObjectsRes{},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client, client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
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
			},
			identifyResponse:     &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:  &magistrala.ListObjectsRes{},
			listObjectsResponse1: &magistrala.ListObjectsRes{},
			listObjectsErr:       svcerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
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
			},
			identifyResponse:     &magistrala.IdentityRes{Id: nonAdminID, UserId: nonAdminID, DomainId: domainID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:  &magistrala.ListObjectsRes{},
			listObjectsResponse1: &magistrala.ListObjectsRes{},
			listObjectsErr1:      svcerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authorizeCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		listAllObjectsCall := auth.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
			SubjectType: authsvc.UserType,
			Subject:     tc.identifyResponse.DomainId + "_" + adminID,
			Permission:  "",
			ObjectType:  authsvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		listAllObjectsCall2 := auth.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
			SubjectType: authsvc.UserType,
			Subject:     tc.identifyResponse.Id,
			Permission:  "",
			ObjectType:  authsvc.ThingType,
		}).Return(tc.listObjectsResponse1, tc.listObjectsErr1)
		retrieveAllCall := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		listPermissionsCall := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)

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
	svc, cRepo, auth, _ := newService()

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
	svc, cRepo, auth, _ := newService()

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
	svc, cRepo, auth, _ := newService()

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
	svc, cRepo, auth, _ := newService()

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
			changeStatusErr:      mgclients.ErrStatusAlreadyAssigned,
			err:                  mgclients.ErrStatusAlreadyAssigned,
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
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: getIDs(tc.response.Clients)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, auth, cache := newService()

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
			changeStatusErr:      mgclients.ErrStatusAlreadyAssigned,
			err:                  mgclients.ErrStatusAlreadyAssigned,
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
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: getIDs(tc.response.Clients)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	nClients := uint64(10)
	aClients := []mgclients.Client{}
	owner := testsutil.GenerateUUID(t)
	for i := uint64(0); i < nClients; i++ {
		identity := fmt.Sprintf("member_%d@example.com", i)
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: identity,
			Credentials: mgclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mgclients.Metadata{"role": "client"},
		}
		if i%3 == 0 {
			client.Owner = owner
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
		listObjectsResponse      *magistrala.ListObjectsRes
		listPermissionsResponse  *magistrala.ListPermissionsRes
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
			desc:    "list members with authorized token",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:     &magistrala.ListObjectsRes{},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
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
				Owner:  adminEmail,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:     &magistrala.ListObjectsRes{},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
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
			desc:    "list members with an invalid token",
			token:   authmocks.InvalidValue,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
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
			desc:    "list members with an invalid id",
			token:   validToken,
			groupID: wrongID,
			page: mgclients.Page{
				Owner: adminEmail,
			},
			identifyResponse:         &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:        &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:      &magistrala.ListObjectsRes{},
			listPermissionsResponse:  &magistrala.ListPermissionsRes{},
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
			desc:    "list members for an owner",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:     &magistrala.ListObjectsRes{},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
			retreiveAllByIDsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 4,
				},
				Clients: []mgclients.Client{aClients[0], aClients[3], aClients[6], aClients[9]},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 4,
				},
				Members: []mgclients.Client{aClients[0], aClients[3], aClients[6], aClients[9]},
			},
			err: nil,
		},
		{
			desc:    "list members for an owner with permissions",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner:     adminEmail,
				ListPerms: true,
			},
			identifyResponse:        &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse:     &magistrala.ListObjectsRes{},
			listPermissionsResponse: &magistrala.ListPermissionsRes{Permissions: []string{"admin"}},
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
				Owner:     adminEmail,
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
				Owner:     adminEmail,
				ListPerms: true,
			},
			identifyResponse:    &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: &magistrala.ListObjectsRes{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:    "list members with failed to list permissions",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner:     adminEmail,
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
			listObjectsResponse:     &magistrala.ListObjectsRes{},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), tc.page).Return(tc.retreiveAllByIDsResponse, tc.retreiveAllByIDsErr)
		repoCall4 := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
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
	svc, cRepo, auth, cache := newService()

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "TestClient",
		Credentials: mgclients.Credentials{
			Identity: "TestClient@example.com",
			Secret:   "password",
		},
		Tags:     []string{"tag1", "tag2"},
		Metadata: mgclients.Metadata{"role": "client"},
	}
	invalidClientID := "invalidClientID"
	_ = invalidClientID
	cases := []struct {
		desc                  string
		token                 string
		identifyResponse      *magistrala.IdentityRes
		authorizeResponse     *magistrala.AuthorizeRes
		deletePolicyResponse  *magistrala.DeletePolicyRes
		deletePolicyResponse1 *magistrala.DeletePolicyRes
		deletePolicyResponse2 *magistrala.DeletePolicyRes
		clientID              string
		identifyErr           error
		authorizeErr          error
		removeErr             error
		deleteErr             error
		deletePolicyErr       error
		deletePolicyErr1      error
		deletePolicyErr2      error
		err                   error
	}{
		{
			desc:                  "Delete client with authorized token",
			token:                 validToken,
			clientID:              client.ID,
			identifyResponse:      &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:     &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse:  &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse1: &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse2: &magistrala.DeletePolicyRes{Deleted: true},
			err:                   nil,
		},
		{
			desc:             "Delete client with unauthorized token",
			token:            authmocks.InvalidValue,
			clientID:         client.ID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      errors.ErrAuthentication,
			err:              errors.ErrAuthentication,
		},
		{
			desc:              "Delete invalid client",
			token:             validToken,
			clientID:          authmocks.InvalidValue,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               errors.ErrAuthorization,
		},
		{
			desc:                  "Delete client with repo error ",
			token:                 validToken,
			clientID:              client.ID,
			identifyResponse:      &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:     &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse:  &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse1: &magistrala.DeletePolicyRes{Deleted: true},
			deleteErr:             errors.ErrRemoveEntity,
			err:                   errors.ErrRemoveEntity,
		},
		{
			desc:              "Delete client with cache error ",
			token:             validToken,
			clientID:          client.ID,
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			removeErr:         svcerr.ErrRemoveEntity,
			err:               errors.ErrRemoveEntity,
		},
		{
			desc:                 "Delete client with failed to delete groups policy",
			token:                validToken,
			clientID:             client.ID,
			identifyResponse:     &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: false},
			deletePolicyErr:      errRemovePolicies,
			err:                  errRemovePolicies,
		},
		{
			desc:                  "Delete client with failed to delete domains policy",
			token:                 validToken,
			clientID:              client.ID,
			identifyResponse:      &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:     &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse:  &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse1: &magistrala.DeletePolicyRes{Deleted: false},
			deletePolicyErr1:      errRemovePolicies,
			err:                   errRemovePolicies,
		},
		{
			desc:                  "Delete client with failed to delete users policy",
			token:                 validToken,
			clientID:              client.ID,
			identifyResponse:      &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:     &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse:  &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse1: &magistrala.DeletePolicyRes{Deleted: true},
			deletePolicyResponse2: &magistrala.DeletePolicyRes{Deleted: false},
			deletePolicyErr2:      errRemovePolicies,
			err:                   errRemovePolicies,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cache.On("Remove", mock.Anything, tc.clientID).Return(tc.removeErr)
		repoCall3 := auth.On("DeletePolicy", context.Background(), &magistrala.DeletePolicyReq{
			SubjectType: authsvc.GroupType,
			Object:      tc.clientID,
			ObjectType:  authsvc.ThingType,
		}).Return(tc.deletePolicyResponse, tc.deletePolicyErr)
		repoCall4 := auth.On("DeletePolicy", mock.Anything, &magistrala.DeletePolicyReq{
			SubjectType: authsvc.DomainType,
			Object:      tc.clientID,
			ObjectType:  authsvc.ThingType,
		}).Return(tc.deletePolicyResponse1, tc.deletePolicyErr1)
		repoCall5 := cRepo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		repoCall6 := auth.On("DeletePolicy", mock.Anything, &magistrala.DeletePolicyReq{
			SubjectType: authsvc.UserType,
			Object:      tc.clientID,
			ObjectType:  authsvc.ThingType,
		}).Return(tc.deletePolicyResponse2, tc.deletePolicyErr2)
		err := svc.DeleteClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		repoCall5.Unset()
		repoCall6.Unset()
	}
}

func TestShare(t *testing.T) {
	svc, _, auth, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc                string
		token               string
		clientID            string
		relation            string
		userID              string
		identifyResponse    *magistrala.IdentityRes
		authorizeResponse   *magistrala.AuthorizeRes
		addPoliciesResponse *magistrala.AddPoliciesRes
		identifyErr         error
		authorizeErr        error
		addPoliciesErr      error
		err                 error
	}{
		{
			desc:                "share thing successfully",
			token:               validToken,
			clientID:            clientID,
			identifyResponse:    &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
			err:                 nil,
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
			desc:                "share thing with failed to add policies",
			token:               validToken,
			clientID:            clientID,
			identifyResponse:    &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			addPoliciesResponse: &magistrala.AddPoliciesRes{},
			addPoliciesErr:      svcerr.ErrInvalidPolicy,
			err:                 errAddPolicies,
		},
		{
			desc:                "share thing with failed authorization from add policies",
			token:               validToken,
			clientID:            clientID,
			identifyResponse:    &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: false},
			err:                 nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesErr)
		err := svc.Share(context.Background(), tc.token, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUnShare(t *testing.T) {
	svc, _, auth, _ := newService()

	clientID := "clientID"

	cases := []struct {
		desc                   string
		token                  string
		clientID               string
		relation               string
		userID                 string
		identifyResponse       *magistrala.IdentityRes
		authorizeResponse      *magistrala.AuthorizeRes
		deletePoliciesResponse *magistrala.DeletePoliciesRes
		identifyErr            error
		authorizeErr           error
		deletePoliciesErr      error
		err                    error
	}{
		{
			desc:                   "unshare thing successfully",
			token:                  validToken,
			clientID:               clientID,
			identifyResponse:       &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:      &magistrala.AuthorizeRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
			err:                    nil,
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
			desc:                   "share thing with failed to delete policies",
			token:                  validToken,
			clientID:               clientID,
			identifyResponse:       &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:      &magistrala.AuthorizeRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{},
			deletePoliciesErr:      svcerr.ErrInvalidPolicy,
			err:                    errRemovePolicies,
		},
		{
			desc:                   "share thing with failed delete from delete policies",
			token:                  validToken,
			clientID:               clientID,
			identifyResponse:       &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse:      &magistrala.AuthorizeRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: false},
			err:                    nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesResponse, tc.deletePoliciesErr)
		err := svc.Unshare(context.Background(), tc.token, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestViewClientPerms(t *testing.T) {
	svc, _, auth, _ := newService()

	validID := valid

	cases := []struct {
		desc              string
		token             string
		thingID           string
		permissions       []string
		identifyResponse  *magistrala.IdentityRes
		authorizeResponse *magistrala.AuthorizeRes
		listPermResponse  *magistrala.ListPermissionsRes
		identifyErr       error
		authorizeErr      error
		listPermErr       error
		err               error
	}{
		{
			desc:              "view client permissions successfully",
			token:             validToken,
			thingID:           validID,
			permissions:       []string{"admin"},
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listPermResponse:  &magistrala.ListPermissionsRes{Permissions: []string{"admin"}},
			err:               nil,
		},
		{
			desc:             "view client permissions with invalid token",
			token:            inValidToken,
			thingID:          validID,
			permissions:      []string{"admin"},
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "view client permissions with invalid ID",
			token:             validToken,
			thingID:           inValidToken,
			permissions:       []string{},
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "view permissions with failed retrieve list permissions response",
			token:             validToken,
			thingID:           validID,
			permissions:       []string{},
			identifyResponse:  &magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listPermResponse:  &magistrala.ListPermissionsRes{},
			listPermErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermResponse, tc.listPermErr)
		_, err := svc.ViewClientPerms(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
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
			cacheIDErr:      errors.ErrNotFound,
			repoIDResponse:  client,
			err:             nil,
		},
		{
			desc:                "identify client with invalid key",
			key:                 invalid,
			cacheIDResponse:     "",
			cacheIDErr:          errors.ErrNotFound,
			repoIDResponse:      mgclients.Client{},
			retrieveBySecretErr: errors.ErrNotFound,
			err:                 errors.ErrNotFound,
		},
		{
			desc:            "identify client with failed to save to cache",
			key:             valid,
			cacheIDResponse: "",
			cacheIDErr:      errors.ErrNotFound,
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
	svc, cRepo, auth, cache := newService()

	cases := []struct {
		desc        string
		token       string
		key         string
		clientID    string
		request     *magistrala.AuthorizeReq
		response    *magistrala.AuthorizeRes
		err         error
		identifyErr error
		authErr     error
	}{
		{
			desc:     "authorize client with valid key",
			key:      valid,
			clientID: valid,
			request:  &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "admin"},
			response: &magistrala.AuthorizeRes{Authorized: true},
			err:      nil,
		},
		{
			desc:        "authorize client with invalid key",
			key:         invalid,
			request:     &magistrala.AuthorizeReq{Subject: invalid, Object: inValidToken, Permission: "admin"},
			response:    &magistrala.AuthorizeRes{Authorized: false},
			identifyErr: errors.ErrNotFound,
			err:         errors.ErrNotFound,
		},
		{
			desc:     "authorize with invalid token",
			key:      valid,
			request:  &magistrala.AuthorizeReq{Subject: valid, Object: inValidToken, Permission: "admin"},
			response: &magistrala.AuthorizeRes{Authorized: false},
			authErr:  svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "authorize with valid failed authorize response",
			key:      valid,
			request:  &magistrala.AuthorizeReq{Subject: valid, Object: valid, Permission: "view"},
			response: &magistrala.AuthorizeRes{Authorized: false},
			authErr:  nil,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cache.On("ID", mock.Anything, mock.Anything).Return(mock.Anything, tc.identifyErr)
		repoCall1 := cRepo.On("RetrieveBySecret", mock.Anything, mock.Anything).Return(mgclients.Client{ID: tc.clientID}, tc.identifyErr)
		repoCall2 := cache.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		repoCall3 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.response, tc.authErr)
		_, err := svc.Authorize(context.Background(), tc.request)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func getIDs(clients []mgclients.Client) []string {
	ids := []string{}
	for _, client := range clients {
		ids = append(ids, client.ID)
	}
	return ids
}
