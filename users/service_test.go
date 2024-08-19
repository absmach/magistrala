// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	authsvc "github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/hasher"
	"github.com/absmach/magistrala/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider     = uuid.New()
	phasher        = hasher.New()
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	clientID       = "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f"
	client         = mgclients.Client{
		ID:          clientID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	basicClient = mgclients.Client{
		Name: "clientname",
		ID:   clientID,
	}
	validToken      = "token"
	inValidToken    = "invalid"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID         = testsutil.GenerateUUID(&testing.T{})
	errHashPassword = errors.New("generate hash from password failed")
)

func newService(selfRegister bool) (users.Service, *mocks.Repository, *authmocks.AuthClient, *mocks.Emailer) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.AuthClient)
	e := new(mocks.Emailer)
	return users.NewService(cRepo, auth, e, phasher, idProvider, selfRegister), cRepo, auth, e
}

func TestRegisterClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	cases := []struct {
		desc                      string
		client                    mgclients.Client
		identifyResponse          *magistrala.IdentityRes
		addPoliciesResponse       *magistrala.AddPoliciesRes
		deletePoliciesResponse    *magistrala.DeletePolicyRes
		token                     string
		identifyErr               error
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		err                       error
	}{
		{
			desc:                "register new client successfully",
			client:              client,
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			token:               validToken,
			err:                 nil,
		},
		{
			desc:                   "register existing client",
			client:                 client,
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			token:                  validToken,
			saveErr:                repoerr.ErrConflict,
			err:                    repoerr.ErrConflict,
		},
		{
			desc: "register a new enabled client with name",
			client: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			err:                 nil,
			token:               validToken,
		},
		{
			desc: "register a new disabled client with name",
			client: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
			},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			err:                 nil,
			token:               validToken,
		},
		{
			desc: "register a new client with all fields",
			client: mgclients.Client{
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			err:                 nil,
			token:               validToken,
		},
		{
			desc: "register a new client with missing identity",
			client: mgclients.Client{
				Name: "clientWithMissingIdentity",
				Credentials: mgclients.Credentials{
					Secret: secret,
				},
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			saveErr:                errors.ErrMalformedEntity,
			err:                    errors.ErrMalformedEntity,
			token:                  validToken,
		},
		{
			desc: "register a new client with missing secret",
			client: mgclients.Client{
				Name: "clientWithMissingSecret",
				Credentials: mgclients.Credentials{
					Identity: "clientwithmissingsecret@example.com",
					Secret:   "",
				},
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			err:                    nil,
		},
		{
			desc: " register a client with a secret that is too long",
			client: mgclients.Client{
				Name: "clientWithLongSecret",
				Credentials: mgclients.Credentials{
					Identity: "clientwithlongsecret@example.com",
					Secret:   strings.Repeat("a", 73),
				},
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			err:                    repoerr.ErrMalformedEntity,
		},
		{
			desc: "register a new client with invalid status",
			client: mgclients.Client{
				Name: "clientWithInvalidStatus",
				Credentials: mgclients.Credentials{
					Identity: "client with invalid status",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			err:                    svcerr.ErrInvalidStatus,
		},
		{
			desc: "register a new client with invalid role",
			client: mgclients.Client{
				Name: "clientWithInvalidRole",
				Credentials: mgclients.Credentials{
					Identity: "clientwithinvalidrole@example.com",
					Secret:   secret,
				},
				Role: 2,
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: true},
			err:                    svcerr.ErrInvalidRole,
		},
		{
			desc: "register a new client with failed to authorize add policies",
			client: mgclients.Client{
				Name: "clientWithFailedToAddPolicies",
				Credentials: mgclients.Credentials{
					Identity: "clientwithfailedpolicies@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: false},
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc: "register a new client with failed to add policies with err",
			client: mgclients.Client{
				Name: "clientWithFailedToAddPolicies",
				Credentials: mgclients.Credentials{
					Identity: "clientwithfailedpolicies@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			addPoliciesResponseErr: svcerr.ErrAddPolicies,
			err:                    svcerr.ErrAddPolicies,
		},
		{
			desc: "register a new client with failed to delete policies with err",
			client: mgclients.Client{
				Name: "clientWithFailedToDeletePolicies",
				Credentials: mgclients.Credentials{
					Identity: "clientwithfailedtodelete@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			addPoliciesResponse:       &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse:    &magistrala.DeletePolicyRes{Deleted: false},
			deletePoliciesResponseErr: svcerr.ErrConflict,
			saveErr:                   repoerr.ErrConflict,
			err:                       svcerr.ErrConflict,
		},
		{
			desc: "register a new client with failed to delete policies with failed to delete",
			client: mgclients.Client{
				Name: "clientWithFailedToDeletePolicies",
				Credentials: mgclients.Credentials{
					Identity: "clientwithfailedtodelete@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Added: true},
			deletePoliciesResponse: &magistrala.DeletePolicyRes{Deleted: false},
			saveErr:                repoerr.ErrConflict,
			err:                    svcerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesResponseErr)
		authCall1 := auth.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponse, tc.deletePoliciesResponseErr)
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
		authCall1.Unset()
		authCall.Unset()
	}

	svc, cRepo, auth, _ = newService(false)

	cases2 := []struct {
		desc                      string
		client                    mgclients.Client
		identifyResponse          *magistrala.IdentityRes
		authorizeResponse         *magistrala.AuthorizeRes
		addPoliciesResponse       *magistrala.AddPoliciesRes
		deletePoliciesResponse    *magistrala.DeletePolicyRes
		token                     string
		identifyErr               error
		authorizeErr              error
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		checkSuperAdminErr        error
		err                       error
	}{
		{
			desc:                "register new client successfully as admin",
			client:              client,
			identifyResponse:    &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			token:               validToken,
			err:                 nil,
		},
		{
			desc:             "register a new clinet as admin with invalid token",
			client:           client,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "register  a new client as admin with failed to authorize",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			identifyErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "register a new client as admin with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases2 {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		authCall2 := auth.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesResponseErr)
		authCall3 := auth.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponse, tc.deletePoliciesResponseErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall1.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}

		repoCall1.Unset()
		authCall3.Unset()
		authCall2.Unset()
		repoCall.Unset()
		authCall1.Unset()
		authCall.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	adminID := testsutil.GenerateUUID(t)
	cases := []struct {
		desc                 string
		token                string
		clientID             string
		identifyResponse     *magistrala.IdentityRes
		authorizeResponse    *magistrala.AuthorizeRes
		retrieveByIDResponse mgclients.Client
		response             mgclients.Client
		identifyErr          error
		authorizeErr         error
		retrieveByIDErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "view client as normal user successfully",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: client,
			response:             client,
			token:                validToken,
			clientID:             client.ID,
			err:                  nil,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view client with an invalid token",
			token:                inValidToken,
			clientID:             clientID,
			identifyResponse:     &magistrala.IdentityRes{},
			authorizeResponse:    &magistrala.AuthorizeRes{},
			retrieveByIDResponse: mgclients.Client{},
			response:             mgclients.Client{},
			identifyErr:          svcerr.ErrAuthentication,
			err:                  svcerr.ErrAuthentication,
		},
		{
			desc:                 "view client as normal user with failed to retrieve client",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: mgclients.Client{},
			token:                validToken,
			clientID:             client.ID,
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view client as admin user successfully",
			identifyResponse:     &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: client,
			response:             client,
			token:                validToken,
			clientID:             client.ID,
			err:                  nil,
		},
		{
			desc:             "view client as admin user with invalid token",
			identifyResponse: &magistrala.IdentityRes{},
			token:            inValidToken,
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:               "view client as admin user with invalid ID",
			identifyResponse:   &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			clientID:           client.ID,
			identifyErr:        svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
			checkSuperAdminErr: nil,
		},
		{
			desc:                 "view client as admin user with failed check on super admin",
			identifyResponse:     &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: false},
			token:                validToken,
			retrieveByIDResponse: basicClient,
			response:             basicClient,
			clientID:             client.ID,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
			err:                  nil,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)

		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}

		repoCall1.Unset()
		repoCall.Unset()
		authCall1.Unset()
		authCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	var clients, strippedClients, permClients []mgclients.Client
	var policies []string
	for i := 0; i < 10; i++ {
		cl := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: fmt.Sprintf("client%d", i),
		}
		strippedClients = append(strippedClients, cl)
		cl.Domain = validID
		cl.Tags = []string{"tag1", "tag2"}
		cl.Credentials = mgclients.Credentials{Identity: fmt.Sprintf("client%d", i), Secret: secret}
		cl.Metadata = validCMetadata
		policies = append(policies, authsvc.EncodeDomainUserID(cl.Domain, cl.ID))
		clients = append(clients, cl)
		cl.Permissions = []string{"view", "edit"}
		permClients = append(permClients, cl)
	}

	cases := []struct {
		desc                    string
		token                   string
		pageMeta                mgclients.Page
		identifyResponse        *magistrala.IdentityRes
		platformAuthResponse    *magistrala.AuthorizeRes
		domainAuthResponse      *magistrala.AuthorizeRes
		retrieveAllResponse     mgclients.ClientsPage
		listAllSubjectsResponse *magistrala.ListSubjectsRes
		listPermissionsResponse *magistrala.ListPermissionsRes
		response                mgclients.ClientsPage
		size                    uint64
		identifyErr             error
		authorizeErr            error
		retrieveAllErr          error
		superAdminErr           error
		listAllSubjectsErr      error
		listPermissionsErr      error
		platformAuthErr         error
		domainAuthErr           error
		err                     error
	}{
		{
			desc:                 "list clients as super admin successfully",
			pageMeta:             mgclients.Page{},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			err: nil,
		},
		{
			desc: "list clients as super admin with entity type and id",
			pageMeta: mgclients.Page{
				EntityType: authsvc.ThingType,
				EntityID:   validID,
			},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID, DomainId: validID, Id: authsvc.EncodeDomainUserID(validID, client.ID)},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: true},
			domainAuthResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: policies,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			err: nil,
		},
		{
			desc: "list clients as super admin with entuty type and id with list perms",
			pageMeta: mgclients.Page{
				EntityType: authsvc.ThingType,
				EntityID:   validID,
				ListPerms:  true,
			},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID, DomainId: validID, Id: authsvc.EncodeDomainUserID(validID, client.ID)},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: true},
			domainAuthResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: policies,
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{
				Permissions: []string{"view", "edit"},
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: permClients,
			},
			err: nil,
		},
		{
			desc:                 "list clients as super admin with failed to retrieve clients",
			pageMeta:             mgclients.Page{},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse:  mgclients.ClientsPage{},
			retrieveAllErr:       repoerr.ErrNotFound,
			err:                  svcerr.ErrViewEntity,
		},
		{
			desc: "list clients as non super admin with entity type and id",
			pageMeta: mgclients.Page{
				EntityType: authsvc.ThingType,
				EntityID:   validID,
			},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID, DomainId: validID, Id: authsvc.EncodeDomainUserID(validID, client.ID)},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: false},
			domainAuthResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: policies,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: strippedClients,
			},
			err: nil,
		},
		{
			desc: "list clients as non super admin with entity type and id with failed to list all subjects",
			pageMeta: mgclients.Page{
				EntityType: authsvc.ThingType,
				EntityID:   validID,
			},
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID, DomainId: validID, Id: authsvc.EncodeDomainUserID(validID, client.ID)},
			platformAuthResponse: &magistrala.AuthorizeRes{Authorized: false},
			domainAuthResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(clients)),
				},
				Clients: clients,
			},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{},
			listAllSubjectsErr:      svcerr.ErrAuthorization,
			err:                     svcerr.ErrViewEntity,
		},
		{
			desc:             "list clients with invalid token",
			pageMeta:         mgclients.Page{},
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		platformCall := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: authsvc.UserType,
			SubjectKind: authsvc.UsersKind,
			Subject:     client.ID,
			Permission:  authsvc.AdminPermission,
			ObjectType:  authsvc.PlatformType,
			Object:      authsvc.MagistralaObject,
		}).Return(tc.platformAuthResponse, tc.platformAuthErr)
		domainCall := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			Domain:      validID,
			SubjectType: authsvc.UserType,
			SubjectKind: authsvc.UsersKind,
			Subject:     tc.identifyResponse.Id,
			Permission:  "",
			ObjectType:  authsvc.ThingType,
			Object:      validID,
		}).Return(tc.domainAuthResponse, tc.domainAuthErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.superAdminErr)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		authCall3 := auth.On("ListAllSubjects", context.Background(), mock.Anything).Return(tc.listAllSubjectsResponse, tc.listAllSubjectsErr)
		authCall4 := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClients(context.Background(), tc.token, tc.pageMeta)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		authCall.Unset()
		platformCall.Unset()
		domainCall.Unset()
		authCall3.Unset()
		authCall4.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestSearchUsers(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)
	cases := []struct {
		desc               string
		token              string
		page               mgclients.Page
		identifyResp       *magistrala.IdentityRes
		authorizeResponse  *magistrala.AuthorizeRes
		response           mgclients.ClientsPage
		responseErr        error
		identifyErr        error
		authorizeErr       error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:  "search clients with valid token",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Name: "clientname", Limit: 100},
			response: mgclients.ClientsPage{
				Page:    mgclients.Page{Total: 1, Offset: 0, Limit: 100},
				Clients: []mgclients.Client{client},
			},
			identifyResp:      &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
		},
		{
			desc:        "search clients with invalid token",
			token:       inValidToken,
			page:        mgclients.Page{Offset: 0, Name: "clientname", Limit: 100},
			response:    mgclients.ClientsPage{},
			responseErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:  "search clients with id",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Id: "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f", Limit: 100},
			response: mgclients.ClientsPage{
				Page:    mgclients.Page{Total: 1, Offset: 0, Limit: 100},
				Clients: []mgclients.Client{client},
			},
			identifyResp:      &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
		},
		{
			desc:  "search clients with random name",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Name: "randomname", Limit: 100},
			response: mgclients.ClientsPage{
				Page:    mgclients.Page{Total: 0, Offset: 0, Limit: 100},
				Clients: []mgclients.Client{},
			},
			identifyResp:      &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
		},
		{
			desc:               "search clients as a normal user",
			token:              validToken,
			page:               mgclients.Page{Offset: 0, Identity: "clientidentity", Limit: 100},
			response:           mgclients.ClientsPage{},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			responseErr:        nil,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResp, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("SearchClients", context.Background(), mock.Anything).Return(tc.response, tc.responseErr)
		page, err := svc.SearchUsers(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = mgclients.Metadata{"role": "test"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc               string
		client             mgclients.Client
		identifyResponse   *magistrala.IdentityRes
		authorizeResponse  *magistrala.AuthorizeRes
		updateResponse     mgclients.Client
		token              string
		identifyErr        error
		authorizeErr       error
		updateErr          error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:             "update client name  successfully as normal user",
			client:           client1,
			identifyResponse: &magistrala.IdentityRes{UserId: client1.ID},
			updateResponse:   client1,
			token:            validToken,
			err:              nil,
		},
		{
			desc:             "update metadata successfully as normal user",
			client:           client2,
			identifyResponse: &magistrala.IdentityRes{UserId: client2.ID},
			updateResponse:   client2,
			token:            validToken,
			err:              nil,
		},
		{
			desc:             "update client name as normal user with invalid token",
			client:           client1,
			identifyResponse: &magistrala.IdentityRes{},
			token:            inValidToken,
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:             "update client name as normal user with repo error on update",
			client:           client1,
			identifyResponse: &magistrala.IdentityRes{UserId: client1.ID},
			updateResponse:   mgclients.Client{},
			token:            validToken,
			updateErr:        errors.ErrMalformedEntity,
			err:              svcerr.ErrUpdateEntity,
		},
		{
			desc:              "update client name as admin successfully",
			client:            client1,
			identifyResponse:  &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateResponse:    client1,
			token:             validToken,
			err:               nil,
		},
		{
			desc:              "update client metadata as admin successfully",
			client:            client2,
			identifyResponse:  &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateResponse:    client2,
			token:             validToken,
			err:               nil,
		},
		{
			desc:             "update client name as admin with invalid token",
			client:           client1,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update cient name as admin with invalid ID",
			client:            client1,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "update client with failed check on super admin",
			client:             client1,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:              "update client name as admin with repo error on update",
			client:            client1,
			identifyResponse:  &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateResponse:    mgclients.Client{},
			token:             validToken,
			updateErr:         errors.ErrMalformedEntity,
			err:               svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))

		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client.Tags = []string{"updated"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                     string
		client                   mgclients.Client
		identifyResponse         *magistrala.IdentityRes
		authorizeResponse        *magistrala.AuthorizeRes
		updateClientTagsResponse mgclients.Client
		token                    string
		identifyErr              error
		authorizeErr             error
		updateClientTagsErr      error
		checkSuperAdminErr       error
		err                      error
	}{
		{
			desc:                     "update client tags as normal user successfully",
			client:                   client,
			identifyResponse:         &magistrala.IdentityRes{UserId: client.ID},
			updateClientTagsResponse: client,
			token:                    validToken,
			err:                      nil,
		},
		{
			desc:             "update client tags as normal user with invalid token",
			client:           client,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                     "update client tags as normal user with repo error on update",
			client:                   client,
			identifyResponse:         &magistrala.IdentityRes{UserId: client.ID},
			updateClientTagsResponse: mgclients.Client{},
			token:                    validToken,
			updateClientTagsErr:      errors.ErrMalformedEntity,
			err:                      svcerr.ErrUpdateEntity,
		},
		{
			desc:              "update client tags as admin successfully",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			err:               nil,
		},
		{
			desc:             "update client tags as admin with invalid token",
			client:           client,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update client tags as admin with invalid ID",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			identifyErr:       svcerr.ErrAuthorization,
			token:             validToken,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "update client tags as admin with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			token:              validToken,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                     "update client tags as admin with repo error on update",
			client:                   client,
			identifyResponse:         &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:        &magistrala.AuthorizeRes{Authorized: true},
			updateClientTagsResponse: mgclients.Client{},
			token:                    validToken,
			updateClientTagsErr:      errors.ErrMalformedEntity,
			err:                      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateClientTagsResponse, tc.updateClientTagsErr)

		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateClientTagsResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateClientTagsResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client2 := client
	client2.Credentials.Identity = "updated@example.com"
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                         string
		identity                     string
		token                        string
		id                           string
		identifyResponse             *magistrala.IdentityRes
		authorizeResponse            *magistrala.AuthorizeRes
		updateClientIdentityResponse mgclients.Client
		identifyErr                  error
		authorizeErr                 error
		updateClientIdentityErr      error
		checkSuperAdminErr           error
		err                          error
	}{
		{
			desc:                         "update client as normal user successfully",
			identity:                     "updated@example.com",
			token:                        validToken,
			id:                           client.ID,
			identifyResponse:             &magistrala.IdentityRes{UserId: client.ID},
			updateClientIdentityResponse: client2,
			err:                          nil,
		},
		{
			desc:             "update client identity as normal user with invalid token",
			identity:         "updated@example.com",
			token:            inValidToken,
			id:               client.ID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                         "update client identity as normal user with repo error on update",
			identity:                     "updated@example.com",
			token:                        validToken,
			id:                           client.ID,
			identifyResponse:             &magistrala.IdentityRes{UserId: client.ID},
			updateClientIdentityResponse: mgclients.Client{},
			updateClientIdentityErr:      errors.ErrMalformedEntity,
			err:                          svcerr.ErrUpdateEntity,
		},
		{
			desc:              "update client identity as admin successfully",
			identity:          "updated@example.com",
			token:             validToken,
			id:                client.ID,
			identifyResponse:  &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:               nil,
		},
		{
			desc:             "update client identity as admin with invalid token",
			identity:         "updated@example.com",
			token:            inValidToken,
			id:               client.ID,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update client identity as admin with invalid ID",
			identity:          "updated@example.com",
			token:             validToken,
			id:                client.ID,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			identifyErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "update client identity as admin with failed check on super admin",
			identity:           "updated@example.com",
			token:              validToken,
			id:                 client.ID,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                         "update client identity as admin with repo error on update",
			identity:                     "updated@exmaple.com",
			token:                        validToken,
			id:                           client.ID,
			identifyResponse:             &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:            &magistrala.AuthorizeRes{Authorized: true},
			updateClientIdentityResponse: mgclients.Client{},
			updateClientIdentityErr:      errors.ErrMalformedEntity,
			err:                          svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("UpdateIdentity", context.Background(), mock.Anything).Return(tc.updateClientIdentityResponse, tc.updateClientIdentityErr)

		updatedClient, err := svc.UpdateClientIdentity(context.Background(), tc.token, tc.id, tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateClientIdentityResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateClientIdentityResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateIdentity", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client2 := client
	client.Role = mgclients.AdminRole
	client2.Role = mgclients.UserRole

	superAdminAuthReq := &magistrala.AuthorizeReq{
		SubjectType: authsvc.UserType,
		SubjectKind: authsvc.UsersKind,
		Subject:     client.ID,
		Permission:  authsvc.AdminPermission,
		ObjectType:  authsvc.PlatformType,
		Object:      authsvc.MagistralaObject,
	}

	membershipAuthReq := &magistrala.AuthorizeReq{
		SubjectType: authsvc.UserType,
		SubjectKind: authsvc.UsersKind,
		Subject:     client.ID,
		Permission:  authsvc.MembershipPermission,
		ObjectType:  authsvc.PlatformType,
		Object:      authsvc.MagistralaObject,
	}

	cases := []struct {
		desc                       string
		client                     mgclients.Client
		identifyResponse           *magistrala.IdentityRes
		superAdminAuthReq          *magistrala.AuthorizeReq
		membershipAuthReq          *magistrala.AuthorizeReq
		superAdminAuthRes          *magistrala.AuthorizeRes
		membershipAuthRes          *magistrala.AuthorizeRes
		deletePolicyFilterResponse *magistrala.DeletePolicyRes
		addPolicyResponse          *magistrala.AddPolicyRes
		updateRoleResponse         mgclients.Client
		token                      string
		identifyErr                error
		authorizeErr               error
		membershipAuthErr          error
		deletePolicyErr            error
		addPolicyErr               error
		updateRoleErr              error
		checkSuperAdminErr         error
		err                        error
	}{
		{
			desc:               "update client role successfully",
			client:             client,
			superAdminAuthReq:  superAdminAuthReq,
			identifyResponse:   &magistrala.IdentityRes{UserId: client.ID},
			membershipAuthReq:  membershipAuthReq,
			membershipAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:  &magistrala.AddPolicyRes{Added: true},
			updateRoleResponse: client,
			token:              validToken,
			err:                nil,
		},
		{
			desc:              "update client role with invalid token",
			client:            client,
			token:             inValidToken,
			superAdminAuthReq: superAdminAuthReq,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			identifyResponse:  &magistrala.IdentityRes{},
			identifyErr:       svcerr.ErrAuthentication,
			err:               svcerr.ErrAuthentication,
		},
		{
			desc:              "update client role with invalid ID",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			superAdminAuthReq: superAdminAuthReq,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			token:             validToken,
			identifyErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "update client role with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq:  superAdminAuthReq,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:              "update client role with failed authorization on add policy",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq: superAdminAuthReq,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq: membershipAuthReq,
			membershipAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPolicyRes{Added: false},
			token:             validToken,
			authorizeErr:      svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:              "update client role with failed to add policy",
			client:            client,
			superAdminAuthReq: superAdminAuthReq,
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq: membershipAuthReq,
			membershipAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPolicyRes{},
			token:             validToken,
			addPolicyErr:      errors.ErrMalformedEntity,
			err:               svcerr.ErrAddPolicies,
		},
		{
			desc:                       "update client role to user role successfully  ",
			client:                     client2,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq:          superAdminAuthReq,
			superAdminAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq:          membershipAuthReq,
			membershipAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyFilterResponse: &magistrala.DeletePolicyRes{Deleted: true},
			updateRoleResponse:         client2,
			token:                      validToken,
			err:                        nil,
		},
		{
			desc:                       "update client role to user role with failed to delete policy",
			client:                     client2,
			superAdminAuthReq:          superAdminAuthReq,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq:          membershipAuthReq,
			membershipAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyFilterResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:         mgclients.Client{},
			token:                      validToken,
			deletePolicyErr:            svcerr.ErrAuthorization,
			err:                        svcerr.ErrAuthorization,
		},
		{
			desc:                       "update client role to user role with failed to delete policy with error",
			client:                     client2,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq:          superAdminAuthReq,
			superAdminAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq:          membershipAuthReq,
			membershipAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyFilterResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:         mgclients.Client{},
			token:                      validToken,
			deletePolicyErr:            svcerr.ErrMalformedEntity,
			err:                        svcerr.ErrDeletePolicies,
		},
		{
			desc:                       "Update client with failed repo update and roll back",
			client:                     client,
			superAdminAuthReq:          superAdminAuthReq,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq:          membershipAuthReq,
			membershipAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:          &magistrala.AddPolicyRes{Added: true},
			deletePolicyFilterResponse: &magistrala.DeletePolicyRes{Deleted: true},
			updateRoleResponse:         mgclients.Client{},
			token:                      validToken,
			updateRoleErr:              svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
		{
			desc:                       "Update client with failed repo update and failedroll back",
			client:                     client,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq:          superAdminAuthReq,
			superAdminAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq:          membershipAuthReq,
			membershipAuthRes:          &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:          &magistrala.AddPolicyRes{Added: true},
			deletePolicyFilterResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:         mgclients.Client{},
			token:                      validToken,
			updateRoleErr:              svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
		{
			desc:              "update client role with failed MembershipPermission authorization",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			superAdminAuthReq: superAdminAuthReq,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			membershipAuthReq: membershipAuthReq,
			membershipAuthRes: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			membershipAuthErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), tc.superAdminAuthReq).Return(tc.superAdminAuthRes, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		authCall2 := auth.On("Authorize", context.Background(), tc.membershipAuthReq).Return(tc.membershipAuthRes, tc.membershipAuthErr)
		authCall3 := auth.On("AddPolicy", context.Background(), mock.Anything).Return(tc.addPolicyResponse, tc.addPolicyErr)
		authCall4 := auth.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyFilterResponse, tc.deletePolicyErr)
		repoCall1 := cRepo.On("UpdateRole", context.Background(), mock.Anything).Return(tc.updateRoleResponse, tc.updateRoleErr)

		updatedClient, err := svc.UpdateClientRole(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateRoleResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateRoleResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateRole", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateRole was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		authCall2.Unset()
		authCall3.Unset()
		authCall4.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	newSecret := "newstrongSecret"
	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)
	responseClient := client
	responseClient.Credentials.Secret = newSecret

	cases := []struct {
		desc                       string
		oldSecret                  string
		newSecret                  string
		token                      string
		identifyResponse           *magistrala.IdentityRes
		retrieveByIDResponse       mgclients.Client
		retrieveByIdentityResponse mgclients.Client
		updateSecretResponse       mgclients.Client
		issueResponse              *magistrala.Token
		response                   mgclients.Client
		identifyErr                error
		retrieveByIDErr            error
		retrieveByIdentityErr      error
		updateSecretErr            error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "update client secret with valid token",
			oldSecret:                  client.Credentials.Secret,
			newSecret:                  newSecret,
			token:                      validToken,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIdentityResponse: rClient,
			retrieveByIDResponse:       client,
			updateSecretResponse:       responseClient,
			issueResponse:              &magistrala.Token{AccessToken: validToken},
			response:                   responseClient,
			err:                        nil,
		},
		{
			desc:             "update client secret with invalid token",
			oldSecret:        client.Credentials.Secret,
			newSecret:        newSecret,
			token:            inValidToken,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                 "update client secret with failed to retrieve client by ID",
			oldSecret:            client.Credentials.Secret,
			newSecret:            newSecret,
			token:                validToken,
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                       "update client secret with failed to retrieve client by identity",
			oldSecret:                  client.Credentials.Secret,
			newSecret:                  newSecret,
			token:                      validToken,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: mgclients.Client{},
			retrieveByIdentityErr:      repoerr.ErrNotFound,
			err:                        repoerr.ErrNotFound,
		},
		{
			desc:                       "update client secret with invalod old secret",
			oldSecret:                  "invalid",
			newSecret:                  newSecret,
			token:                      validToken,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			err:                        svcerr.ErrLogin,
		},
		{
			desc:                       "update client secret with too long new secret",
			oldSecret:                  client.Credentials.Secret,
			newSecret:                  strings.Repeat("a", 73),
			token:                      validToken,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			err:                        repoerr.ErrMalformedEntity,
		},
		{
			desc:                       "update client secret with failed to update secret",
			oldSecret:                  client.Credentials.Secret,
			newSecret:                  newSecret,
			token:                      validToken,
			identifyResponse:           &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			updateSecretResponse:       mgclients.Client{},
			updateSecretErr:            repoerr.ErrMalformedEntity,
			err:                        svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall := cRepo.On("RetrieveByID", context.Background(), client.ID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall1 := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		authCall1 := auth.On("Issue", context.Background(), mock.Anything).Return(tc.issueResponse, tc.issueErr)

		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.response.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		authCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		authCall1.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	enabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mgclients.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		token                string
		client               mgclients.Client
		identifyResponse     *magistrala.IdentityRes
		authorizeResponse    *magistrala.AuthorizeRes
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		identifyErr          error
		authorizeErr         error
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "enable disabled client",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: disabledClient1,
			changeStatusResponse: endisabledClient1,
			response:             endisabledClient1,
			err:                  nil,
		},
		{
			desc:             "enable disabled client with invalid token",
			id:               disabledClient1.ID,
			token:            inValidToken,
			client:           disabledClient1,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "enable disabled client with failed to authorize",
			id:                disabledClient1.ID,
			token:             validToken,
			client:            disabledClient1,
			identifyResponse:  &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			identifyErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "enable disabled client with normal user token",
			id:                 disabledClient1.ID,
			token:              validToken,
			client:             disabledClient1,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "enable disabled client with failed to retrieve client by ID",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "enable already enabled client",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: enabledClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable disabled client with failed to change status",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: disabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	enabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		token                string
		client               mgclients.Client
		identifyResponse     *magistrala.IdentityRes
		authorizeResponse    *magistrala.AuthorizeRes
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		identifyErr          error
		authorizeErr         error
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "disable enabled client",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: disenabledClient1,
			response:             disenabledClient1,
			err:                  nil,
		},
		{
			desc:             "disable enabled client with invalid token",
			id:               enabledClient1.ID,
			token:            inValidToken,
			client:           enabledClient1,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "disable enabled client with failed to authorize",
			id:                enabledClient1.ID,
			token:             validToken,
			client:            enabledClient1,
			identifyResponse:  &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			identifyErr:       svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "disable enabled client with normal user token",
			id:                 enabledClient1.ID,
			token:              validToken,
			client:             enabledClient1,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "disable enabled client with failed to retrieve client by ID",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable already disabled client",
			id:                   disabledClient1.ID,
			token:                validToken,
			client:               disabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: disabledClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable enabled client with failed to change status",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDeleteClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	enabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	deletedClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DeletedStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DeletedStatus

	cases := []struct {
		desc                 string
		id                   string
		token                string
		client               mgclients.Client
		identifyResponse     *magistrala.IdentityRes
		authorizeResponse    *magistrala.AuthorizeRes
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		identifyErr          error
		authorizeErr         error
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "delete enabled client",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: disenabledClient1,
			response:             disenabledClient1,
			err:                  nil,
		},
		{
			desc:             "delete enabled client with invalid token",
			id:               enabledClient1.ID,
			token:            inValidToken,
			client:           enabledClient1,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "delete enabled client with failed to authorize",
			id:                enabledClient1.ID,
			token:             validToken,
			client:            enabledClient1,
			identifyResponse:  &magistrala.IdentityRes{UserId: deletedClient1.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:               "delete enabled client with normal user token",
			id:                 enabledClient1.ID,
			token:              validToken,
			client:             enabledClient1,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "delete enabled client with failed to retrieve client by ID",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "delete already deleted client",
			id:                   deletedClient1.ID,
			token:                validToken,
			client:               deletedClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: deletedClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: deletedClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "delete enabled client with failed to change status",
			id:                   enabledClient1.ID,
			token:                validToken,
			client:               enabledClient1,
			identifyResponse:     &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall4 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		err := svc.DeleteClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall4.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestIssueToken(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	rClient := client
	rClient2 := client
	rClient3 := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)
	rClient2.Credentials.Secret = "wrongsecret"
	rClient3.Credentials.Secret, _ = phasher.Hash("wrongsecret")

	cases := []struct {
		desc                       string
		domainID                   string
		client                     mgclients.Client
		retrieveByIdentityResponse mgclients.Client
		issueResponse              *magistrala.Token
		retrieveByIdentityErr      error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "issue token for an existing client",
			client:                     client,
			retrieveByIdentityResponse: rClient,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:                       "issue token for non-empty domain id",
			domainID:                   validID,
			client:                     client,
			retrieveByIdentityResponse: rClient,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:                       "issue token for a non-existing client",
			client:                     client,
			retrieveByIdentityResponse: mgclients.Client{},
			retrieveByIdentityErr:      repoerr.ErrNotFound,
			err:                        repoerr.ErrNotFound,
		},
		{
			desc:                       "issue token for a client with wrong secret",
			client:                     client,
			retrieveByIdentityResponse: rClient3,
			err:                        svcerr.ErrLogin,
		},
		{
			desc:                       "issue token with empty domain id",
			client:                     client,
			retrieveByIdentityResponse: rClient,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
		{
			desc:                       "issue token with grpc error",
			client:                     client,
			retrieveByIdentityResponse: rClient,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		authCall := auth.On("Issue", context.Background(), &magistrala.IssueReq{UserId: tc.client.ID, DomainId: &tc.domainID, Type: uint32(authsvc.AccessKey)}).Return(tc.issueResponse, tc.issueErr)
		token, err := svc.IssueToken(context.Background(), tc.client.Credentials.Identity, tc.client.Credentials.Secret, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = authCall.Parent.AssertCalled(t, "Issue", context.Background(), &magistrala.IssueReq{UserId: tc.client.ID, DomainId: &tc.domainID, Type: uint32(authsvc.AccessKey)})
			assert.True(t, ok, fmt.Sprintf("Issue was not called on %s", tc.desc))
		}
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	svc, crepo, auth, _ := newService(true)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	cases := []struct {
		desc         string
		token        string
		domainID     string
		identifyResp *magistrala.IdentityRes
		identifyErr  error
		refreshResp  *magistrala.Token
		refresErr    error
		repoResp     mgclients.Client
		repoErr      error
		err          error
	}{
		{
			desc:         "refresh token with refresh token for an existing client",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			refreshResp:  &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:     rClient,
			err:          nil,
		},
		{
			desc:         "refresh token with refresh token for empty domain id",
			token:        validToken,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			refreshResp:  &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:     rClient,
			err:          nil,
		},
		{
			desc:         "refresh token with access token for an existing client",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			refreshResp:  &magistrala.Token{},
			refresErr:    svcerr.ErrAuthentication,
			repoResp:     rClient,
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:         "refresh token with invalid token",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{},
			identifyErr:  svcerr.ErrAuthentication,
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:         "refresh token with refresh token for a non-existing client",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			repoErr:      repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
		{
			desc:         "refresh token with refresh token for a disable client",
			token:        validToken,
			domainID:     validID,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			repoResp:     mgclients.Client{Status: mgclients.DisabledStatus},
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:         "refresh token with empty domain id",
			token:        validToken,
			identifyResp: &magistrala.IdentityRes{UserId: client.ID},
			refreshResp:  &magistrala.Token{},
			refresErr:    svcerr.ErrAuthentication,
			repoResp:     rClient,
			err:          svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResp, tc.identifyErr)
		authCall1 := auth.On("Refresh", context.Background(), &magistrala.RefreshReq{RefreshToken: tc.token, DomainId: &tc.domainID}).Return(tc.refreshResp, tc.refresErr)
		repoCall := crepo.On("RetrieveByID", context.Background(), tc.identifyResp.GetUserId()).Return(tc.repoResp, tc.repoErr)
		token, err := svc.RefreshToken(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := authCall.Parent.AssertCalled(t, "Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token})
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = authCall.Parent.AssertCalled(t, "Refresh", context.Background(), &magistrala.RefreshReq{RefreshToken: tc.token, DomainId: &tc.domainID})
			assert.True(t, ok, fmt.Sprintf("Refresh was not called on %s", tc.desc))
			ok = repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.identifyResp.UserId)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc, cRepo, auth, e := newService(true)

	cases := []struct {
		desc                       string
		email                      string
		host                       string
		retrieveByIdentityResponse mgclients.Client
		issueResponse              *magistrala.Token
		retrieveByIdentityErr      error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "generate reset token for existing client",
			email:                      "existingemail@example.com",
			host:                       "examplehost",
			retrieveByIdentityResponse: client,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:  "generate reset token for client with non-existing client",
			email: "example@example.com",
			host:  "examplehost",
			retrieveByIdentityResponse: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "",
				},
			},
			retrieveByIdentityErr: repoerr.ErrNotFound,
			err:                   repoerr.ErrNotFound,
		},
		{
			desc:                       "generate reset token with failed to issue token",
			email:                      "existingemail@example.com",
			host:                       "examplehost",
			retrieveByIdentityResponse: client,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthorization,
			err:                        svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.email).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		authCall := auth.On("Issue", context.Background(), mock.Anything).Return(tc.issueResponse, tc.issueErr)

		svcCall := e.On("SendPasswordReset", []string{tc.email}, tc.host, client.Name, validToken).Return(tc.err)
		err := svc.GenerateResetToken(context.Background(), tc.email, tc.host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.email)
		repoCall.Unset()
		authCall.Unset()
		svcCall.Unset()
	}
}

func TestResetSecret(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client := mgclients.Client{
		ID: "clientID",
		Credentials: mgclients.Credentials{
			Identity: "test@example.com",
			Secret:   "Strongsecret",
		},
	}

	cases := []struct {
		desc                 string
		token                string
		newSecret            string
		identifyResponse     *magistrala.IdentityRes
		retrieveByIDResponse mgclients.Client
		updateSecretResponse mgclients.Client
		identifyErr          error
		retrieveByIDErr      error
		updateSecretErr      error
		err                  error
	}{
		{
			desc:                 "reset secret with successfully",
			token:                validToken,
			newSecret:            "newStrongSecret",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: client,
			updateSecretResponse: mgclients.Client{
				ID: "clientID",
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
					Secret:   "newStrongSecret",
				},
			},
			err: nil,
		},
		{
			desc:             "reset secret with invalid token",
			token:            inValidToken,
			newSecret:        "newStrongSecret",
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                 "reset secret with invalid ID",
			token:                validToken,
			newSecret:            "newStrongSecret",
			identifyResponse:     &magistrala.IdentityRes{UserId: wrongID},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:             "reset secret with empty identity",
			token:            validToken,
			newSecret:        "newStrongSecret",
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: mgclients.Client{
				ID: "clientID",
				Credentials: mgclients.Credentials{
					Identity: "",
				},
			},
			err: nil,
		},
		{
			desc:                 "reset secret with failed to update secret",
			token:                validToken,
			newSecret:            "newStrongSecret",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: client,
			updateSecretResponse: mgclients.Client{},
			updateSecretErr:      svcerr.ErrUpdateEntity,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:                 "reset secret with a too long secret",
			token:                validToken,
			newSecret:            strings.Repeat("strongSecret", 10),
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: client,
			err:                  errHashPassword,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall1 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)

		err := svc.ResetSecret(context.Background(), tc.token, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		repoCall1.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
		repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), client.ID)
		authCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
		authCall.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestViewProfile(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client := mgclients.Client{
		ID: "clientID",
		Credentials: mgclients.Credentials{
			Identity: "existingIdentity",
			Secret:   "Strongsecret",
		},
	}
	cases := []struct {
		desc                 string
		token                string
		client               mgclients.Client
		identifyResponse     *magistrala.IdentityRes
		retrieveByIDResponse mgclients.Client
		identifyErr          error
		retrieveByIDErr      error
		err                  error
	}{
		{
			desc:                 "view profile successfully",
			token:                validToken,
			client:               client,
			identifyResponse:     &magistrala.IdentityRes{UserId: validID},
			retrieveByIDResponse: client,
			err:                  nil,
		},
		{
			desc:             "view profile with invalid token",
			token:            inValidToken,
			client:           client,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                 "view profile with invalid ID",
			token:                validToken,
			client:               client,
			identifyResponse:     &magistrala.IdentityRes{UserId: wrongID},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), mock.Anything).Return(tc.identifyResponse, tc.identifyErr)
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)

		_, err := svc.ViewProfile(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		authCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
		repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), mock.Anything)
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestOAuthCallback(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	cases := []struct {
		desc                       string
		client                     mgclients.Client
		retrieveByIdentityResponse mgclients.Client
		retrieveByIdentityErr      error
		addPoliciesResponse        *magistrala.AddPoliciesRes
		addPoliciesErr             error
		saveResponse               mgclients.Client
		saveErr                    error
		deletePoliciesResponse     *magistrala.DeletePolicyRes
		deletePoliciesErr          error
		authorizeResponse          *magistrala.AuthorizeRes
		authorizeErr               error
		issueResponse              *magistrala.Token
		issueErr                   error
		err                        error
	}{
		{
			desc: "oauth signin callback with successfully",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			issueResponse: &magistrala.Token{
				AccessToken:  strings.Repeat("a", 10),
				RefreshToken: &validToken,
				AccessType:   "Bearer",
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with successfully",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityErr: repoerr.ErrNotFound,
			addPoliciesResponse: &magistrala.AddPoliciesRes{
				Added: true,
			},
			saveResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			issueResponse: &magistrala.Token{
				AccessToken:  strings.Repeat("a", 10),
				RefreshToken: &validToken,
				AccessType:   "Bearer",
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with unknown error",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityErr: repoerr.ErrMalformedEntity,
			err:                   repoerr.ErrMalformedEntity,
		},
		{
			desc: "oauth signup callback with failed to register user",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityErr: repoerr.ErrNotFound,
			addPoliciesResponse:   &magistrala.AddPoliciesRes{Added: false},
			addPoliciesErr:        svcerr.ErrAuthorization,
			err:                   svcerr.ErrAuthorization,
		},
		{
			desc: "oauth signin callback with user not in the platform",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:        svcerr.ErrAuthorization,
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: true},
			issueResponse: &magistrala.Token{
				AccessToken:  strings.Repeat("a", 10),
				RefreshToken: &validToken,
				AccessType:   "Bearer",
			},
			err: nil,
		},
		{
			desc: "oauth signin callback with user not in the platform and failed to add policy",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr:        svcerr.ErrAuthorization,
			addPoliciesResponse: &magistrala.AddPoliciesRes{Added: false},
			addPoliciesErr:      svcerr.ErrAuthorization,
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc: "oauth signin callback with failed to issue token",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			issueErr:          svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		id := tc.saveResponse.ID
		if tc.retrieveByIdentityResponse.ID != "" {
			id = tc.retrieveByIdentityResponse.ID
		}
		authReq := &magistrala.AuthorizeReq{
			SubjectType: authsvc.UserType,
			SubjectKind: authsvc.UsersKind,
			Subject:     id,
			Permission:  authsvc.MembershipPermission,
			ObjectType:  authsvc.PlatformType,
			Object:      authsvc.MagistralaObject,
		}
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.saveResponse, tc.saveErr)
		authCall := auth.On("Issue", mock.Anything, mock.Anything).Return(tc.issueResponse, tc.issueErr)
		authCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesErr)
		authCall2 := auth.On("Authorize", mock.Anything, authReq).Return(tc.authorizeResponse, tc.authorizeErr)
		token, err := svc.OAuthCallback(context.Background(), tc.client)
		if err == nil {
			assert.Equal(t, tc.issueResponse.AccessToken, token.AccessToken)
			assert.Equal(t, tc.issueResponse.RefreshToken, token.RefreshToken)
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
		repoCall.Unset()
		repoCall1.Unset()
		authCall.Unset()
		authCall1.Unset()
		authCall2.Unset()
	}
}
