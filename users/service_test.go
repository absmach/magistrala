// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
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
	"github.com/stretchr/testify/require"
)

var (
	idProvider     = uuid.New()
	phasher        = hasher.New()
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	client         = mgclients.Client{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	passRegex         = regexp.MustCompile("^.{8,}$")
	validToken        = "token"
	inValidToken      = "invalid"
	validID           = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID           = testsutil.GenerateUUID(&testing.T{})
	errHashPassword   = errors.New("generate hash from password failed")
	errAddPolicies    = errors.New("failed to add policies")
	errDeletePolicies = errors.New("failed to delete policies")
)

func newService(selfRegister bool) (users.Service, *mocks.Repository, *authmocks.Service, users.Emailer) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	return users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, selfRegister), cRepo, auth, e
}

func TestRegisterClient(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	cases := []struct {
		desc                      string
		client                    mgclients.Client
		identifyResponse          *magistrala.IdentityRes
		addPoliciesResponse       *magistrala.AddPoliciesRes
		deletePoliciesResponse    *magistrala.DeletePoliciesRes
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
			token:               validToken,
			err:                 nil,
		},
		{
			desc:                   "register existing client",
			client:                 client,
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
			token:                  validToken,
			saveErr:                repoerr.ErrConflict,
			err:                    errors.ErrConflict,
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
			err:                    repoerr.ErrMissingSecret,
		},
		{
			desc: "register a new client with a weak secret",
			client: mgclients.Client{
				Name: "clientWithWeakSecret",
				Credentials: mgclients.Credentials{
					Identity: "clientwithweaksecret@example.com",
					Secret:   "weak",
				},
			},
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: true},
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: false},
			err:                 errors.ErrAuthorization,
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			addPoliciesResponseErr: errAddPolicies,
			err:                    errAddPolicies,
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
			addPoliciesResponse:       &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse:    &magistrala.DeletePoliciesRes{Deleted: false},
			deletePoliciesResponseErr: errDeletePolicies,
			saveErr:                   repoerr.ErrConflict,
			err:                       errDeletePolicies,
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
			addPoliciesResponse:    &magistrala.AddPoliciesRes{Authorized: true},
			deletePoliciesResponse: &magistrala.DeletePoliciesRes{Deleted: false},
			saveErr:                repoerr.ErrConflict,
			err:                    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesResponseErr)
		repoCall1 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesResponse, tc.deletePoliciesResponseErr)
		repoCall2 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.Owner = expected.Owner
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall2.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}

	svc, cRepo, auth, _ = newService(false)

	cases2 := []struct {
		desc                      string
		client                    mgclients.Client
		identifyResponse          *magistrala.IdentityRes
		authorizeResponse         *magistrala.AuthorizeRes
		addPoliciesResponse       *magistrala.AddPoliciesRes
		deletePoliciesResponse    *magistrala.DeletePoliciesRes
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
			addPoliciesResponse: &magistrala.AddPoliciesRes{Authorized: true},
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
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "register a new client as admin with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
		},
	}
	for _, tc := range cases2 {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesResponse, tc.addPoliciesResponseErr)
		repoCall4 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesResponse, tc.deletePoliciesResponseErr)
		repoCall5 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.Owner = expected.Owner
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall5.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}

		repoCall5.Unset()
		repoCall4.Unset()
		repoCall3.Unset()
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
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
		},
		{
			desc:             "view client with an invalid token",
			identifyResponse: &magistrala.IdentityRes{},
			response:         mgclients.Client{},
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:                 "view client as normal user with failed to retrieve client",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: mgclients.Client{},
			token:                validToken,
			clientID:             client.ID,
			retrieveByIDErr:      errors.ErrNotFound,
			err:                  svcerr.ErrNotFound,
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
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "view client as admin user with invalid ID",
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			clientID:          client.ID,
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "view client as admin user with failed check on super admin",
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			clientID:           client.ID,
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)

		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}

		repoCall3.Unset()
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	cases := []struct {
		desc                string
		token               string
		page                mgclients.Page
		identifyResponse    *magistrala.IdentityRes
		authorizeResponse   *magistrala.AuthorizeRes
		retrieveAllResponse mgclients.ClientsPage
		response            mgclients.ClientsPage
		size                uint64
		identifyErr         error
		authorizeErr        error
		retrieveAllErr      error
		superAdminErr       error
		err                 error
	}{
		{
			desc: "list clients as admin successfully",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "list clients as admin with invalid token",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      svcerr.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc: "list clients as admin with invalid ID",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			authorizeErr:      svcerr.ErrAuthorization,
			err:               nil,
		},
		{
			desc: "list clients as admin with failed to retrieve clients",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:    &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveAllResponse: mgclients.ClientsPage{},
			token:               validToken,
			retrieveAllErr:      errors.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc: "list clients as admin with failed check on super admin",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			superAdminErr:     errors.ErrAuthorization,
			err:               nil,
		},
		{
			desc: "list clients as normal user successfully",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "list clients as normal user with failed to retrieve clients",
			page: mgclients.Page{
				Total: 1,
			},
			identifyResponse:    &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:   &magistrala.AuthorizeRes{Authorized: false},
			retrieveAllResponse: mgclients.ClientsPage{},
			token:               validToken,
			retrieveAllErr:      errors.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.superAdminErr)
		repoCall3 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListClients(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
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
			identifyErr:      errors.ErrAuthentication,
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
			identifyErr:      errors.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update cient name as admin with invalid ID",
			client:            client1,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "update client with failed check on super admin",
			client:             client1,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			token:              validToken,
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))

		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
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
			identifyErr:      errors.ErrAuthentication,
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
			identifyErr:      errors.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update client tags as admin with invalid ID",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "update client tags as admin with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: errors.ErrAuthorization,
			token:              validToken,
			err:                errors.ErrAuthorization,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateClientTagsResponse, tc.updateClientTagsErr)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateClientTagsResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateClientTagsResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
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
			identifyErr:      errors.ErrAuthentication,
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
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update client identity as admin with invalid ID",
			identity:          "updated@example.com",
			token:             validToken,
			id:                client.ID,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "update client identity as admin with failed check on super admin",
			identity:           "updated@example.com",
			token:              validToken,
			id:                 client.ID,
			identifyResponse:   &magistrala.IdentityRes{UserId: adminID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("UpdateIdentity", context.Background(), mock.Anything).Return(tc.updateClientIdentityResponse, tc.updateClientIdentityErr)
		updatedClient, err := svc.UpdateClientIdentity(context.Background(), tc.token, tc.id, tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateClientIdentityResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateClientIdentityResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "UpdateIdentity", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	client2 := client
	client.Role = mgclients.AdminRole
	client2.Role = mgclients.UserRole

	cases := []struct {
		desc                 string
		client               mgclients.Client
		identifyResponse     *magistrala.IdentityRes
		authorizeResponse    *magistrala.AuthorizeRes
		deletePolicyResponse *magistrala.DeletePolicyRes
		addPolicyResponse    *magistrala.AddPolicyRes
		updateRoleResponse   mgclients.Client
		token                string
		identifyErr          error
		authorizeErr         error
		deletePolicyErr      error
		addPolicyErr         error
		updateRoleErr        error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:               "update client role successfully",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:  &magistrala.AddPolicyRes{Authorized: true},
			updateRoleResponse: client,
			token:              validToken,
			err:                nil,
		},
		{
			desc:             "update client role with invalid token",
			client:           client,
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      errors.ErrAuthentication,
			token:            inValidToken,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "update client role with invalid ID",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: wrongID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			token:             validToken,
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "update client role with failed check on super admin",
			client:             client,
			identifyResponse:   &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: errors.ErrAuthorization,
			token:              validToken,
			err:                errors.ErrAuthorization,
		},
		{
			desc:              "update client role with failed authorization on add policy",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPolicyRes{Authorized: false},
			token:             validToken,
			err:               errors.ErrAuthorization,
		},
		{
			desc:              "update client role with failed to add policy",
			client:            client,
			identifyResponse:  &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse: &magistrala.AddPolicyRes{},
			addPolicyErr:      errors.ErrMalformedEntity,
			token:             validToken,
			err:               errAddPolicies,
		},
		{
			desc:                 "update client role to user role successfully  ",
			client:               client2,
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: true},
			updateRoleResponse:   client2,
			token:                validToken,
			err:                  nil,
		},
		{
			desc:                 "update client role to user role with failed to delete policy",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:   mgclients.Client{},
			token:                validToken,
			err:                  errDeletePolicies,
		},
		{
			desc:                 "update client role to user role with failed to delete policy with error",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:   mgclients.Client{},
			token:                validToken,
			deletePolicyErr:      svcerr.ErrMalformedEntity,
			err:                  errDeletePolicies,
		},
		{
			desc:                 "Update client with failed repo update and roll back",
			client:               client,
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:    &magistrala.AddPolicyRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: true},
			updateRoleResponse:   mgclients.Client{},
			token:                validToken,
			updateRoleErr:        svcerr.ErrAuthentication,
			err:                  svcerr.ErrAuthentication,
		},
		{
			desc:                 "Update client with failed repo update and failedroll back",
			client:               client,
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			authorizeResponse:    &magistrala.AuthorizeRes{Authorized: true},
			addPolicyResponse:    &magistrala.AddPolicyRes{Authorized: true},
			deletePolicyResponse: &magistrala.DeletePolicyRes{Deleted: false},
			updateRoleResponse:   mgclients.Client{},
			token:                validToken,
			updateRoleErr:        svcerr.ErrAuthentication,
			err:                  svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(tc.addPolicyResponse, tc.addPolicyErr)
		repoCall4 := auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(tc.deletePolicyResponse, tc.deletePolicyErr)
		repoCall5 := cRepo.On("UpdateRole", context.Background(), mock.Anything).Return(tc.updateRoleResponse, tc.updateRoleErr)
		updatedClient, err := svc.UpdateClientRole(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateRoleResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateRoleResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall5.Parent.AssertCalled(t, "UpdateRole", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateRole was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		repoCall5.Unset()
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
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:             "update client secret with weak secret",
			oldSecret:        client.Credentials.Secret,
			newSecret:        "weak",
			token:            validToken,
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			err:              users.ErrPasswordFormat,
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
			err:                        errors.ErrLogin,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), client.ID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall3 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		repoCall4 := auth.On("Issue", mock.Anything, mock.Anything).Return(tc.issueResponse, tc.issueErr)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.response.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
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
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "enable disabled client with failed to authorize",
			id:                disabledClient1.ID,
			token:             validToken,
			client:            disabledClient1,
			identifyResponse:  &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "enable disabled client with normal user token",
			id:                 disabledClient1.ID,
			token:              validToken,
			client:             disabledClient1,
			identifyResponse:   &magistrala.IdentityRes{UserId: validID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
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
			err:                  mgclients.ErrStatusAlreadyAssigned,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall4 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: client.ID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
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
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
		{
			desc:              "disable enabled client with failed to authorize",
			id:                enabledClient1.ID,
			token:             validToken,
			client:            enabledClient1,
			identifyResponse:  &magistrala.IdentityRes{UserId: disabledClient1.ID},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               errors.ErrAuthorization,
		},
		{
			desc:               "disable enabled client with normal user token",
			id:                 enabledClient1.ID,
			token:              validToken,
			client:             enabledClient1,
			identifyResponse:   &magistrala.IdentityRes{UserId: enabledClient1.ID},
			authorizeResponse:  &magistrala.AuthorizeRes{Authorized: false},
			checkSuperAdminErr: errors.ErrAuthorization,
			err:                errors.ErrAuthorization,
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
			err:                  mgclients.ErrStatusAlreadyAssigned,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := cRepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall4 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: client.ID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

	validPolicy := fmt.Sprintf("%s_%s", validID, client.ID)
	permissionsClient := client
	permissionsClient.Permissions = []string{"read"}

	cases := []struct {
		desc                    string
		token                   string
		groupID                 string
		objectKind              string
		objectID                string
		page                    mgclients.Page
		identifyResponse        *magistrala.IdentityRes
		authorizeReq            *magistrala.AuthorizeReq
		listAllSubjectsReq      *magistrala.ListSubjectsReq
		authorizeResponse       *magistrala.AuthorizeRes
		listAllSubjectsResponse *magistrala.ListSubjectsRes
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse *magistrala.ListPermissionsRes
		response                mgclients.MembersPage
		authorizeErr            error
		listAllSubjectsErr      error
		retrieveAllErr          error
		identifyErr             error
		listPermissionErr       error
		err                     error
	}{
		{
			desc:                    "list members with no policies successfully of the things kind",
			token:                   validToken,
			groupID:                 validID,
			objectKind:              authsvc.ThingsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse:        &magistrala.IdentityRes{UserId: client.ID},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  100,
				},
			},
			err: nil,
		},
		{
			desc:             "list members with policies successsfully of the things kind",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []mgclients.Client{client},
			},
			err: nil,
		},
		{
			desc:             "list members with policies successsfully of the things kind with permissions",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{Permissions: []string{"read"}},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []mgclients.Client{permissionsClient},
			},
			err: nil,
		},
		{
			desc:             "list members with policies of the things kind with permissionswith failed list permissions",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client},
			},
			listPermissionsResponse: &magistrala.ListPermissionsRes{},
			response:                mgclients.MembersPage{},
			listPermissionErr:       svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:             "list members with of the things kind with failed to authorize",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               errors.ErrAuthorization,
		},
		{
			desc:             "list members with of the things kind with failed to list all subjects",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse:       &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsErr:      errors.ErrNotFound,
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{},
			err:                     errors.ErrNotFound,
		},
		{
			desc:             "list members with of the things kind with failed to retrieve all",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.ThingsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.ThingType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.ThingType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{},
			response:            mgclients.MembersPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 repoerr.ErrNotFound,
		},
		{
			desc:                    "list members with no policies successfully of the domain kind",
			token:                   validToken,
			groupID:                 validID,
			objectKind:              authsvc.DomainsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse:        &magistrala.IdentityRes{UserId: client.ID},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.DomainType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.DomainType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  100,
				},
			},
			err: nil,
		},
		{
			desc:             "list members with policies successsfully of the domains kind",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.DomainsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.DomainType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.DomainType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []mgclients.Client{client},
			},
			err: nil,
		},
		{
			desc:             "list members with of the domains kind with failed to authorize",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.DomainsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.DomainType,
				Object:      validID,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:               errors.ErrAuthorization,
		},
		{
			desc:                    "list members with no policies successfully of the groups kind",
			token:                   validToken,
			groupID:                 validID,
			objectKind:              authsvc.GroupsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse:        &magistrala.IdentityRes{UserId: client.ID},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.GroupType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.GroupType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  100,
				},
			},
			err: nil,
		},
		{
			desc:             "list members with policies successsfully of the domains kind",
			token:            validToken,
			groupID:          validID,
			objectKind:       authsvc.GroupsKind,
			objectID:         validID,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{UserId: client.ID},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: authsvc.UserType,
				SubjectKind: authsvc.TokenKind,
				Subject:     validToken,
				Permission:  "read",
				ObjectType:  authsvc.GroupType,
				Object:      validID,
			},
			listAllSubjectsReq: &magistrala.ListSubjectsReq{
				SubjectType: authsvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  authsvc.GroupType,
			},
			authorizeResponse: &magistrala.AuthorizeRes{Authorized: true},
			listAllSubjectsResponse: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{client},
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []mgclients.Client{client},
			},
			err: nil,
		},
		{
			desc:             "list members with invalid token",
			token:            inValidToken,
			page:             mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			identifyResponse: &magistrala.IdentityRes{},
			identifyErr:      errors.ErrAuthentication,
			err:              svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, tc.authorizeReq).Return(tc.authorizeResponse, tc.authorizeErr)
		repoCall2 := auth.On("ListAllSubjects", mock.Anything, tc.listAllSubjectsReq).Return(tc.listAllSubjectsResponse, tc.listAllSubjectsErr)
		repoCall3 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		repoCall4 := auth.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionErr)
		page, err := svc.ListMembers(context.Background(), tc.token, tc.objectKind, tc.objectID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))

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
		DomainID                   string
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
			desc:                       "issue token for a non-existing client",
			client:                     client,
			retrieveByIdentityResponse: mgclients.Client{},
			retrieveByIdentityErr:      errors.ErrNotFound,
			err:                        repoerr.ErrNotFound,
		},
		{
			desc:                       "issue token for a client with wrong secret",
			client:                     client,
			retrieveByIdentityResponse: rClient3,
			err:                        errors.ErrLogin,
		},
		{
			desc:                       "issue token with non-empty domain id",
			DomainID:                   "domain",
			client:                     client,
			retrieveByIdentityResponse: rClient,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall1 := auth.On("Issue", mock.Anything, mock.Anything).Return(tc.issueResponse, tc.issueErr)
		token, err := svc.IssueToken(context.Background(), tc.client.Credentials.Identity, tc.client.Credentials.Secret, tc.DomainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	svc, _, auth, _ := newService(true)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	cases := []struct {
		desc     string
		token    string
		domainID string
		client   mgclients.Client
		err      error
	}{
		{
			desc:   "refresh token with refresh token for an existing client",
			token:  validToken,
			client: client,
			err:    nil,
		},
		{
			desc:   "refresh token with refresh token for a non-existing client",
			token:  validToken,
			client: mgclients.Client{},
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for an existing client",
			token:  validToken,
			client: client,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for a non-existing client",
			token:  validToken,
			client: mgclients.Client{},
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with invalid token for an existing client",
			token:  inValidToken,
			client: client,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:     "refresh token with non-empty domain id",
			token:    validToken,
			domainID: validID,
			client:   client,
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Refresh", mock.Anything, mock.Anything).Return(&magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"}, tc.err)
		token, err := svc.RefreshToken(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
		}
		repoCall.Unset()
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc, cRepo, auth, _ := newService(true)

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
			retrieveByIdentityErr: errors.ErrNotFound,
			err:                   errors.ErrNotFound,
		},
		{
			desc:                       "generate reset token with failed to issue token",
			email:                      "existingemail@example.com",
			host:                       "examplehost",
			retrieveByIdentityResponse: client,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthorization,
			err:                        users.ErrRecoveryToken,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.email).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall1 := auth.On("Issue", mock.Anything, mock.Anything).Return(tc.issueResponse, tc.issueErr)
		err := svc.GenerateResetToken(context.Background(), tc.email, tc.host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.email)
		repoCall.Unset()
		repoCall1.Unset()
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
			err: errors.ErrNotFound,
		},
		{
			desc:                 "reset secret with invalid secret format",
			token:                validToken,
			newSecret:            "weak",
			identifyResponse:     &magistrala.IdentityRes{UserId: client.ID},
			retrieveByIDResponse: client,
			err:                  users.ErrPasswordFormat,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		err := svc.ResetSecret(context.Background(), tc.token, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		repoCall2.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
		repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), client.ID)
		repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
		repoCall.Unset()
		repoCall2.Unset()
		repoCall1.Unset()
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
		repoCall := auth.On("Identify", mock.Anything, mock.Anything).Return(tc.identifyResponse, tc.identifyErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		_, err := svc.ViewProfile(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
		repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), mock.Anything)
		repoCall.Unset()
		repoCall1.Unset()
	}
}
