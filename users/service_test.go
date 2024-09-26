// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
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
	validCMetadata = users.Metadata{"role": "client"}
	userID         = "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f"
	user           = users.User{
		ID:          userID,
		Name:        "usersname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: users.Credentials{Identity: "useridentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	basicUser = users.User{
		Name: "usertname",
		ID:   userID,
	}
	validToken      = "token"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID         = testsutil.GenerateUUID(&testing.T{})
	errHashPassword = errors.New("generate hash from password failed")
)

func newService() (users.Service, *authmocks.TokenServiceClient, *mocks.Repository, *policymocks.Service, *mocks.Emailer) {
	cRepo := new(mocks.Repository)
	policies := new(policymocks.Service)
	e := new(mocks.Emailer)
	tokenClient := new(authmocks.TokenServiceClient)
	return users.NewService(tokenClient, cRepo, policies, e, phasher, idProvider), tokenClient, cRepo, policies, e
}

func newServiceMinimal() (users.Service, *mocks.Repository) {
	cRepo := new(mocks.Repository)
	policies := new(policymocks.Service)
	e := new(mocks.Emailer)
	tokenClient := new(authmocks.TokenServiceClient)
	return users.NewService(tokenClient, cRepo, policies, e, phasher, idProvider), cRepo
}

func TestRegisterClient(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	cases := []struct {
		desc                      string
		client                    mgclients.Client
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		err                       error
	}{
		{
			desc:   "register new client successfully",
			client: client,
			err:    nil,
		},
		{
			desc:    "register existing client",
			client:  client,
			saveErr: repoerr.ErrConflict,
			err:     repoerr.ErrConflict,
		},
		{
			desc: "register a new enabled user with name",
			user: users.User{
				Name: "userWithName",
				Credentials: users.Credentials{
					Identity: "newuserwithname@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "register a new disabled user with name",
			user: users.User{
				Name: "userWithName",
				Credentials: users.Credentials{
					Identity: "newuserwithname@example.com",
					Secret:   secret,
				},
			},
			err: nil,
		},
		{
			desc: "register a new user with all fields",
			user: users.User{
				Name: "newusertwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: users.Credentials{
					Identity: "newuserwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: users.Metadata{
					"name": "newuserwithallfields",
				},
				Status: mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "register a new user with missing identity",
			user: users.User{
				Name: "userWithMissingIdentity",
				Credentials: users.Credentials{
					Secret: secret,
				},
			},
			saveErr: errors.ErrMalformedEntity,
			err:     errors.ErrMalformedEntity,
		},
		{
			desc: "register a new user with missing secret",
			user: users.User{
				Name: "userWithMissingSecret",
				Credentials: users.Credentials{
					Identity: "userwithmissingsecret@example.com",
					Secret:   "",
				},
			},
			err: nil,
		},
		{
			desc: " register a user with a secret that is too long",
			user: users.User{
				Name: "userWithLongSecret",
				Credentials: users.Credentials{
					Identity: "userwithlongsecret@example.com",
					Secret:   strings.Repeat("a", 73),
				},
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "register a new user with invalid status",
			user: users.User{
				Name: "userWithInvalidStatus",
				Credentials: users.Credentials{
					Identity: "user with invalid status",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			err: svcerr.ErrInvalidStatus,
		},
		{
			desc: "register a new user with invalid role",
			user: users.User{
				Name: "userWithInvalidRole",
				Credentials: users.Credentials{
					Identity: "userwithinvalidrole@example.com",
					Secret:   secret,
				},
				Role: 2,
			},
			err: svcerr.ErrInvalidRole,
		},
		{
			desc: "register a new user with failed to add policies with err",
			user: users.User{
				Name: "userWithFailedToAddPolicies",
				Credentials: users.Credentials{
					Identity: "userwithfailedpolicies@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			addPoliciesResponseErr: svcerr.ErrAddPolicies,
			err:                    svcerr.ErrAddPolicies,
		},
		{
			desc: "register a new userient with failed to delete policies with err",
			user: users.User{
				Name: "userWithFailedToDeletePolicies",
				Credentials: users.Credentials{
					Identity: "userwithfailedtodelete@example.com",
					Secret:   secret,
				},
				Role: mgclients.AdminRole,
			},
			deletePoliciesResponseErr: svcerr.ErrConflict,
			saveErr:                   repoerr.ErrConflict,
			err:                       svcerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponseErr)
		policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponseErr)
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), authn.Session{}, tc.client, true)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.user.ID = expected.ID
			tc.user.CreatedAt = expected.CreatedAt
			tc.user.UpdatedAt = expected.UpdatedAt
			tc.user.Credentials.Secret = expected.Credentials.Secret
			tc.user.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.user, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.user, expected))
			ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
		policyCall.Unset()
		policyCall1.Unset()
	}

	svc, _, cRepo, policies, _ = newService()

	cases2 := []struct {
		desc                      string
		client                    mgclients.Client
		session                   authn.Session
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		checkSuperAdminErr        error
		err                       error
	}{
		{
			desc:    "register new client successfully as admin",
			client:  client,
			session: authn.Session{UserID: validID, SuperAdmin: true},
			err:     nil,
		},
		{
			desc:               "register a new client as admin with failed check on super admin",
			client:             client,
			session:            authn.Session{UserID: validID, SuperAdmin: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases2 {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponseErr)
		policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponseErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.saveErr)
		expected, err := svc.RegisterClient(context.Background(), authn.Session{UserID: validID}, tc.client, false)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.user.ID = expected.ID
			tc.user.CreatedAt = expected.CreatedAt
			tc.user.UpdatedAt = expected.UpdatedAt
			tc.user.Credentials.Secret = expected.Credentials.Secret
			tc.user.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.user, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.user, expected))
			ok := repoCall1.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		policyCall.Unset()
		policyCall1.Unset()
		repoCall.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	cases := []struct {
		desc                 string
		token                string
		reqClientID          string
		clientID             string
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
			retrieveByIDResponse: client,
			response:             client,
			token:                validToken,
			reqClientID:          client.ID,
			clientID:             client.ID,
			err:                  nil,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view client as normal user with failed to retrieve client",
			retrieveByIDResponse: mgclients.Client{},
			token:                validToken,
			reqClientID:          client.ID,
			clientID:             client.ID,
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view client as admin user successfully",
			retrieveByIDResponse: client,
			response:             client,
			token:                validToken,
			reqClientID:          client.ID,
			clientID:             client.ID,
			err:                  nil,
		},
		{
			desc:                 "view client as admin user with failed check on super admin",
			token:                validToken,
			retrieveByIDResponse: basicClient,
			response:             basicClient,
			reqClientID:          client.ID,
			clientID:             "",
			checkSuperAdminErr:   svcerr.ErrAuthorization,
			err:                  nil,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		rClient, err := svc.ViewClient(context.Background(), authn.Session{UserID: tc.reqClientID}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.userID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	cases := []struct {
		desc                string
		token               string
		page                mgclients.Page
		retrieveAllResponse mgclients.ClientsPage
		response            mgclients.ClientsPage
		size                uint64
		retrieveAllErr      error
		superAdminErr       error
		err                 error
	}{
		{
			desc: "list users as admin successfully",
			page: mgclients.Page{
				Total: 1,
			},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			response: users.UsersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "list users as admin with failed to retrieve users",
			page: mgclients.Page{
				Total: 1,
			},
			retrieveAllResponse: mgclients.ClientsPage{},
			token:               validToken,
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrViewEntity,
		},
		{
			desc: "list users as admin with failed check on super admin",
			page: mgclients.Page{
				Total: 1,
			},
			token:         validToken,
			superAdminErr: svcerr.ErrAuthorization,
			err:           svcerr.ErrAuthorization,
		},
		{
			desc: "list users as normal user with failed to retrieve users",
			page: mgclients.Page{
				Total: 1,
			},
			retrieveAllResponse: mgclients.ClientsPage{},
			token:               validToken,
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.superAdminErr)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListClients(context.Background(), authn.Session{UserID: client.ID}, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestSearchUsers(t *testing.T) {
	svc, cRepo := newServiceMinimal()
	cases := []struct {
		desc        string
		token       string
		page        mgclients.Page
		response    mgclients.ClientsPage
		responseErr error
		err         error
	}{
		{
			desc:  "search users with valid token",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Name: "username", Limit: 100},
			response: users.UsersPage{
				Page:  mgclients.Page{Total: 1, Offset: 0, Limit: 100},
				Users: []users.User{user},
			},
		},
		{
			desc:  "search users with id",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Id: "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f", Limit: 100}, // another mgclients.Page
			response: users.UsersPage{
				Page:  mgclients.Page{Total: 1, Offset: 0, Limit: 100}, // another mgclients.Page
				Users: []users.User{user},
			},
		},
		{
			desc:  "search users with random name",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Name: "randomname", Limit: 100},
			response: users.UsersPage{
				Page:  mgclients.Page{Total: 0, Offset: 0, Limit: 100},
				Users: []users.User{},
			},
		},
		{
			desc:  "search clients with repo failed",
			token: validToken,
			page:  mgclients.Page{Offset: 0, Name: "randomname", Limit: 100},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{Total: 0, Offset: 0, Limit: 0},
			},
			responseErr: repoerr.ErrViewEntity,
			err:         svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("SearchClients", context.Background(), mock.Anything).Return(tc.response, tc.responseErr)
		page, err := svc.SearchUsers(context.Background(), tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user1 := user
	user2 := user
	user1.Name = "Updated user"
	user2.Metadata = users.Metadata{"role": "test"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc               string
		client             mgclients.Client
		session            authn.Session
		updateResponse     mgclients.Client
		token              string
		updateErr          error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:           "update client name  successfully as normal user",
			client:         client1,
			session:        authn.Session{UserID: client1.ID},
			updateResponse: client1,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update metadata successfully as normal user",
			client:         client2,
			session:        authn.Session{UserID: client2.ID},
			updateResponse: client2,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update client name as normal user with repo error on update",
			client:         client1,
			session:        authn.Session{UserID: client1.ID},
			updateResponse: mgclients.Client{},
			token:          validToken,
			updateErr:      errors.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
		{
			desc:           "update client name as admin successfully",
			client:         client1,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: client1,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update client metadata as admin successfully",
			client:         client2,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: client2,
			token:          validToken,
			err:            nil,
		},
		{
			desc:               "update client with failed check on super admin",
			client:             client1,
			session:            authn.Session{UserID: adminID},
			token:              validToken,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:           "update client name as admin with repo error on update",
			client:         client1,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: mgclients.Client{},
			token:          validToken,
			updateErr:      errors.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user.Tags = []string{"updated"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                     string
		client                   mgclients.Client
		session                  authn.Session
		updateClientTagsResponse mgclients.Client
		updateClientTagsErr      error
		checkSuperAdminErr       error
		err                      error
	}{
		{
			desc:                     "update client tags as normal user successfully",
			client:                   client,
			session:                  authn.Session{UserID: client.ID},
			updateClientTagsResponse: client,
			err:                      nil,
		},
		{
			desc:                     "update client tags as normal user with repo error on update",
			client:                   client,
			session:                  authn.Session{UserID: client.ID},
			updateClientTagsResponse: mgclients.Client{},
			updateClientTagsErr:      errors.ErrMalformedEntity,
			err:                      svcerr.ErrUpdateEntity,
		},
		{
			desc:    "update client tags as admin successfully",
			client:  client,
			session: authn.Session{UserID: adminID, SuperAdmin: true},
			err:     nil,
		},
		{
			desc:               "update client tags as admin with failed check on super admin",
			client:             client,
			session:            authn.Session{UserID: adminID},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                     "update client tags as admin with repo error on update",
			client:                   client,
			session:                  authn.Session{UserID: adminID, SuperAdmin: true},
			updateClientTagsResponse: mgclients.Client{},
			updateClientTagsErr:      errors.ErrMalformedEntity,
			err:                      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateClientTagsResponse, tc.updateClientTagsErr)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateUserTagsResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateUserTagsResponse, updatedUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	client2 := client
	client.Role = mgclients.AdminRole
	client2.Role = mgclients.UserRole

	cases := []struct {
		desc               string
		client             mgclients.Client
		session            authn.Session
		updateRoleResponse mgclients.Client
		deletePolicyErr    error
		addPolicyErr       error
		updateRoleErr      error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:               "update client role successfully",
			client:             client,
			session:            authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleResponse: client,
			err:                nil,
		},
		{
			desc:               "update client role with failed check on super admin",
			client:             client,
			session:            authn.Session{UserID: validID, SuperAdmin: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:         "update client role with failed to add policies",
			client:       client,
			session:      authn.Session{UserID: validID, SuperAdmin: true},
			addPolicyErr: errors.ErrMalformedEntity,
			err:          svcerr.ErrAddPolicies,
		},
		{
			desc:               "update client role to user role successfully  ",
			client:             client2,
			session:            authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleResponse: client2,
			err:                nil,
		},
		{
			desc:            "update client role to user role with failed to delete policies",
			client:          client2,
			session:         authn.Session{UserID: validID, SuperAdmin: true},
			deletePolicyErr: svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:            "update client role to user role with failed to delete policies with error",
			client:          client2,
			session:         authn.Session{UserID: validID, SuperAdmin: true},
			deletePolicyErr: svcerr.ErrMalformedEntity,
			err:             svcerr.ErrDeletePolicies,
		},
		{
			desc:          "Update client with failed repo update and roll back",
			client:        client,
			session:       authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleErr: svcerr.ErrAuthentication,
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:            "Update client with failed repo update and failedroll back",
			client:          client,
			session:         authn.Session{UserID: validID, SuperAdmin: true},
			deletePolicyErr: svcerr.ErrAuthorization,
			updateRoleErr:   svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		policyCall := policies.On("AddPolicy", context.Background(), mock.Anything).Return(tc.addPolicyErr)
		policyCall1 := policies.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyErr)
		repoCall1 := cRepo.On("UpdateRole", context.Background(), mock.Anything).Return(tc.updateRoleResponse, tc.updateRoleErr)

		updatedClient, err := svc.UpdateClientRole(context.Background(), tc.session, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateRoleResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateRoleResponse, updatedUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateRole", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateRole was not called on %s", tc.desc))
		}
		repoCall.Unset()
		policyCall.Unset()
		policyCall1.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, authClient, cRepo, _, _ := newService()

	newSecret := "newstrongSecret"
	rUser := user
	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)
	responseUser := user
	responseUser.Credentials.Secret = newSecret

	cases := []struct {
		desc                       string
		oldSecret                  string
		newSecret                  string
		session                    authn.Session
		retrieveByIDResponse       mgclients.Client
		retrieveByIdentityResponse mgclients.Client
		updateSecretResponse       mgclients.Client
		issueResponse              *magistrala.Token
		response                   mgclients.Client
		retrieveByIDErr            error
		retrieveByIdentityErr      error
		updateSecretErr            error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "update user secret with valid token",
			oldSecret:                  user.Credentials.Secret,
			newSecret:                  newSecret,
			session:                    authn.Session{UserID: client.ID},
			retrieveByIdentityResponse: rClient,
			retrieveByIDResponse:       client,
			updateSecretResponse:       responseClient,
			issueResponse:              &magistrala.Token{AccessToken: validToken},
			response:                   responseUser,
			err:                        nil,
		},
		{
			desc:                 "update user secret with failed to retrieve user by ID",
			oldSecret:            user.Credentials.Secret,
			newSecret:            newSecret,
			session:              authn.Session{UserID: client.ID},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                       "update user secret with failed to retrieve user by identity",
			oldSecret:                  user.Credentials.Secret,
			newSecret:                  newSecret,
			session:                    authn.Session{UserID: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: mgclients.Client{},
			retrieveByIdentityErr:      repoerr.ErrNotFound,
			err:                        repoerr.ErrNotFound,
		},
		{
			desc:                       "update user secret with invalid old secret",
			oldSecret:                  "invalid",
			newSecret:                  newSecret,
			session:                    authn.Session{UserID: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			err:                        svcerr.ErrLogin,
		},
		{
			desc:                       "update user secret with too long new secret",
			oldSecret:                  user.Credentials.Secret,
			newSecret:                  strings.Repeat("a", 73),
			session:                    authn.Session{UserID: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			err:                        repoerr.ErrMalformedEntity,
		},
		{
			desc:                       "update user secret with failed to update secret",
			oldSecret:                  user.Credentials.Secret,
			newSecret:                  newSecret,
			session:                    authn.Session{UserID: client.ID},
			retrieveByIDResponse:       client,
			retrieveByIdentityResponse: rClient,
			updateSecretResponse:       mgclients.Client{},
			updateSecretErr:            repoerr.ErrMalformedEntity,
			err:                        svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), client.ID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall1 := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		authCall := authClient.On("Issue", context.Background(), mock.Anything).Return(tc.issueResponse, tc.issueErr)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.session, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedUser))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.response.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		authCall.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	client2 := client
	client2.Credentials.Identity = "updated@example.com"

	cases := []struct {
		desc                         string
		identity                     string
		token                        string
		reqClientID                  string
		id                           string
		updateClientIdentityResponse mgclients.Client
		updateClientIdentityErr      error
		checkSuperAdminErr           error
		err                          error
	}{
		{
			desc:                         "update client as normal user successfully",
			identity:                     "updated@example.com",
			token:                        validToken,
			reqClientID:                  client.ID,
			id:                           client.ID,
			updateClientIdentityResponse: client2,
			err:                          nil,
		},
		{
			desc:                         "update client identity as normal user with repo error on update",
			identity:                     "updated@example.com",
			token:                        validToken,
			reqClientID:                  client.ID,
			id:                           client.ID,
			updateClientIdentityResponse: mgclients.Client{},
			updateClientIdentityErr:      errors.ErrMalformedEntity,
			err:                          svcerr.ErrUpdateEntity,
		},
		{
			desc:     "update client identity as admin successfully",
			identity: "updated@example.com",
			token:    validToken,
			id:       client.ID,
			err:      nil,
		},
		{
			desc:                         "update client identity as admin with repo error on update",
			identity:                     "updated@exmaple.com",
			token:                        validToken,
			reqClientID:                  client.ID,
			id:                           client.ID,
			updateClientIdentityResponse: mgclients.Client{},
			updateClientIdentityErr:      errors.ErrMalformedEntity,
			err:                          svcerr.ErrUpdateEntity,
		},
		{
			desc:                         "update client as admin user with failed check on super admin",
			identity:                     "updated@exmaple.com",
			token:                        validToken,
			reqClientID:                  client.ID,
			id:                           "",
			updateClientIdentityResponse: mgclients.Client{},
			updateClientIdentityErr:      errors.ErrMalformedEntity,
			checkSuperAdminErr:           svcerr.ErrAuthorization,
			err:                          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("UpdateIdentity", context.Background(), mock.Anything).Return(tc.updateClientIdentityResponse, tc.updateClientIdentityErr)
		updatedClient, err := svc.UpdateClientIdentity(context.Background(), authn.Session{DomainUserID: tc.reqClientID, UserID: validID, DomainID: validID}, tc.id, tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateClientIdentityResponse, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateClientIdentityResponse, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "UpdateIdentity", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "uaer1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "uaer3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	endisabledUser1 := disabledUser1
	endisabledUser1.Status = mgclients.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		client               mgclients.Client
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "enable disabled client",
			id:                   disabledClient1.ID,
			client:               disabledClient1,
			retrieveByIDResponse: disabledClient1,
			changeStatusResponse: endisabledClient1,
			response:             endisabledClient1,
			err:                  nil,
		},
		{
			desc:               "enable disabled client with normal user token",
			id:                 disabledClient1.ID,
			client:             disabledClient1,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "enable disabled client with failed to retrieve client by ID",
			id:                   disabledClient1.ID,
			client:               disabledClient1,
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "enable already enabled client",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			retrieveByIDResponse: enabledClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable disabled client with failed to change status",
			id:                   disabledClient1.ID,
			client:               disabledClient1,
			retrieveByIDResponse: disabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.EnableClient(context.Background(), authn.Session{}, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "user1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "user3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	disenabledUser1 := enabledUser1
	disenabledUser1.Status = mgclients.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		client               mgclients.Client
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "disable enabled client",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: disenabledClient1,
			response:             disenabledClient1,
			err:                  nil,
		},
		{
			desc:               "disable enabled client with normal user token",
			id:                 enabledClient1.ID,
			client:             enabledClient1,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "disable enabled client with failed to retrieve client by ID",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable already disabled client",
			id:                   disabledClient1.ID,
			client:               disabledClient1,
			retrieveByIDResponse: disabledClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable enabled client with failed to change status",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.DisableClient(context.Background(), authn.Session{}, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDeleteClient(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "user1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	deletedUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Identity: "user3@example.com", Secret: "password"}, Status: mgclients.DeletedStatus}
	disenabledUser1 := enabledUser1
	disenabledUser1.Status = mgclients.DeletedStatus

	cases := []struct {
		desc                 string
		id                   string
		session              authn.Session
		client               mgclients.Client
		retrieveByIDResponse mgclients.Client
		changeStatusResponse mgclients.Client
		response             mgclients.Client
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "delete enabled client",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: disenabledClient1,
			response:             disenabledClient1,
			err:                  nil,
		},
		{
			desc:                 "delete enabled client with failed to retrieve client by ID",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "delete already deleted client",
			id:                   deletedClient1.ID,
			client:               deletedClient1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: deletedClient1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "delete enabled client with failed to change status",
			id:                   enabledClient1.ID,
			client:               enabledClient1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: enabledClient1,
			changeStatusResponse: mgclients.Client{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall2 := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall4 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		err := svc.DeleteClient(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall4.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	validPolicy := fmt.Sprintf("%s_%s", validID, user.ID)
	permissionsUser := basicUser
	permissionsUser.Permissions = []string{"read"}

	cases := []struct {
		desc                    string
		groupID                 string
		objectKind              string
		objectID                string
		page                    mgclients.Page
		listAllSubjectsReq      policysvc.Policy
		listAllSubjectsResponse policysvc.PolicyPage
		retrieveAllResponse     mgclients.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                mgclients.MembersPage
		listAllSubjectsErr      error
		retrieveAllErr          error
		identifyErr             error
		listPermissionErr       error
		err                     error
	}{
		{
			desc:                    "list members with no policies successfully of the things kind",
			groupID:                 validID,
			objectKind:              policysvc.ThingsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
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
			desc:       "list members with policies successsfully of the things kind",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			response: users.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []users.User{basicUser},
			},
			err: nil,
		},
		{
			desc:       "list members with policies successsfully of the things kind with permissions",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{basicUser},
			},
			listPermissionsResponse: []string{"read"},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []users.User{permissionsUser},
			},
			err: nil,
		},
		{
			desc:       "list members with policies of the things kind with permissionswith failed list permissions",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			listPermissionsResponse: []string{},
			response:                mgclients.MembersPage{},
			listPermissionErr:       svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:       "list members with of the things kind with failed to list all subjects",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsErr:      repoerr.ErrNotFound,
			listAllSubjectsResponse: policysvc.PolicyPage{},
			err:                     repoerr.ErrNotFound,
		},
		{
			desc:       "list members with of the things kind with failed to retrieve all",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse:     mgclients.ClientsPage{},
			response:                mgclients.MembersPage{},
			retrieveAllErr:          repoerr.ErrNotFound,
			err:                     repoerr.ErrNotFound,
		},
		{
			desc:                    "list members with no policies successfully of the domain kind",
			groupID:                 validID,
			objectKind:              policysvc.DomainsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.DomainType,
			},
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
			desc:       "list members with policies successsfully of the domains kind",
			groupID:    validID,
			objectKind: policysvc.DomainsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.DomainType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{basicUser},
			},
			response: users.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []users.User{basicUser},
			},
			err: nil,
		},
		{
			desc:                    "list members with no policies successfully of the groups kind",
			groupID:                 validID,
			objectKind:              policysvc.GroupsKind,
			objectID:                validID,
			page:                    mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.GroupType,
			},
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
			desc: "list members with policies successsfully of the groups kind",

			groupID:    validID,
			objectKind: policysvc.GroupsKind,
			objectID:   validID,
			page:       mgclients.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.GroupType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			response: users.MembersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Members: []users.User{basicUser},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("ListAllSubjects", context.Background(), tc.listAllSubjectsReq).Return(tc.listAllSubjectsResponse, tc.listAllSubjectsErr)
		repoCall := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		policyCall1 := policies.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionErr)
		page, err := svc.ListMembers(context.Background(), authn.Session{}, tc.objectKind, tc.objectID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		policyCall.Unset()
		repoCall.Unset()
		policyCall1.Unset()
	}
}

func TestIssueToken(t *testing.T) {
	svc, auth, cRepo, _, _ := newService()

	rUser := user
	rUser2 := user
	rUser3 := user
	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)
	rUser2.Credentials.Secret = "wrongsecret"
	rUser3.Credentials.Secret, _ = phasher.Hash("wrongsecret")

	cases := []struct {
		desc                       string
		domainID                   string
		user                       users.User
		retrieveByIdentityResponse users.User
		issueResponse              *magistrala.Token
		retrieveByIdentityErr      error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "issue token for an existing user",
			user:                       user,
			retrieveByIdentityResponse: rUser,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:                       "issue token for non-empty domain id",
			domainID:                   validID,
			user:                       user,
			retrieveByIdentityResponse: rUser,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:                       "issue token for a non-existing user",
			user:                       user,
			retrieveByIdentityResponse: users.User{},
			retrieveByIdentityErr:      repoerr.ErrNotFound,
			err:                        repoerr.ErrNotFound,
		},
		{
			desc:                       "issue token for a user with wrong secret",
			user:                       user,
			retrieveByIdentityResponse: rUser3,
			err:                        svcerr.ErrLogin,
		},
		{
			desc:                       "issue token with empty domain id",
			user:                       user,
			retrieveByIdentityResponse: rUser,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
		{
			desc:                       "issue token with grpc error",
			user:                       user,
			retrieveByIdentityResponse: rUser,
			issueResponse:              &magistrala.Token{},
			issueErr:                   svcerr.ErrAuthentication,
			err:                        svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		authCall := auth.On("Issue", context.Background(), &magistrala.IssueReq{UserId: tc.client.ID, DomainId: &tc.domainID, Type: uint32(mgauth.AccessKey)}).Return(tc.issueResponse, tc.issueErr)
		token, err := svc.IssueToken(context.Background(), tc.client.Credentials.Identity, tc.client.Credentials.Secret, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.user.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = authCall.Parent.AssertCalled(t, "Issue", context.Background(), &magistrala.IssueReq{UserId: tc.client.ID, DomainId: &tc.domainID, Type: uint32(mgauth.AccessKey)})
			assert.True(t, ok, fmt.Sprintf("Issue was not called on %s", tc.desc))
		}
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	svc, authsvc, crepo, _, _ := newService()

	rUser := user
	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

	cases := []struct {
		desc        string
		session     authn.Session
		domainID    string
		refreshResp *magistrala.Token
		refresErr   error
		repoResp    mgclients.Client
		repoErr     error
		err         error
	}{
		{
			desc:        "refresh token with refresh token for an existing client",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID:    validID,
			refreshResp: &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:    rClient,
			err:         nil,
		},
		{
			desc:        "refresh token with refresh token for empty domain id",
			session:     authn.Session{UserID: validID},
			refreshResp: &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:    rClient,
			err:         nil,
		},
		{
			desc:        "refresh token with access token for an existing client",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID:    validID,
			refreshResp: &magistrala.Token{},
			refresErr:   svcerr.ErrAuthentication,
			repoResp:    rClient,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "refresh token with refresh token for a non-existing client",
			session:  authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID: validID,
			repoErr:  repoerr.ErrNotFound,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "refresh token with refresh token for a disable client",
			session:  authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID: validID,
			repoResp: mgclients.Client{Status: mgclients.DisabledStatus},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:        "refresh token with empty domain id",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			refreshResp: &magistrala.Token{},
			refresErr:   svcerr.ErrAuthentication,
			repoResp:    rClient,
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := authsvc.On("Refresh", context.Background(), &magistrala.RefreshReq{RefreshToken: validToken, DomainId: &tc.domainID}).Return(tc.refreshResp, tc.refresErr)
		repoCall := crepo.On("RetrieveByID", context.Background(), tc.session.UserID).Return(tc.repoResp, tc.repoErr)
		token, err := svc.RefreshToken(context.Background(), tc.session, validToken, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := authCall.Parent.AssertCalled(t, "Refresh", context.Background(), &magistrala.RefreshReq{RefreshToken: validToken, DomainId: &tc.domainID})
			assert.True(t, ok, fmt.Sprintf("Refresh was not called on %s", tc.desc))
			ok = repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.session.UserID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestGenerateResetToken(t *testing.T) {
	svc, auth, cRepo, _, e := newService()

	cases := []struct {
		desc                       string
		email                      string
		host                       string
		retrieveByIdentityResponse users.User
		issueResponse              *magistrala.Token
		retrieveByIdentityErr      error
		issueErr                   error
		err                        error
	}{
		{
			desc:                       "generate reset token for existing user",
			email:                      "existingemail@example.com",
			host:                       "examplehost",
			retrieveByIdentityResponse: user,
			issueResponse:              &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                        nil,
		},
		{
			desc:  "generate reset token for user with non-existing user",
			email: "example@example.com",
			host:  "examplehost",
			retrieveByIdentityResponse: users.User{
				ID: testsutil.GenerateUUID(t),
				Credentials: users.Credentials{
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
			retrieveByIdentityResponse: user,
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
	svc, cRepo := newServiceMinimal()

	user := users.User{
		ID: "clientID",
		Credentials: users.Credentials{
			Identity: "test@example.com",
			Secret:   "Strongsecret",
		},
	}

	cases := []struct {
		desc                 string
		newSecret            string
		session              authn.Session
		retrieveByIDResponse mgclients.Client
		updateSecretResponse mgclients.Client
		retrieveByIDErr      error
		updateSecretErr      error
		err                  error
	}{
		{
			desc:                 "reset secret with successfully",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: client,
			updateSecretResponse: mgclients.Client{
				ID: "clientID",
				Credentials: users.Credentials{
					Identity: "test@example.com",
					Secret:   "newStrongSecret",
				},
			},
			err: nil,
		},
		{
			desc:                 "reset secret with invalid ID",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:      "reset secret with empty identity",
			session:   authn.Session{UserID: validID, SuperAdmin: true},
			newSecret: "newStrongSecret",
			retrieveByIDResponse: mgclients.Client{
				ID: "clientID",
				Credentials: users.Credentials{
					Identity: "",
				},
			},
			err: nil,
		},
		{
			desc:                 "reset secret with failed to update secret",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: client,
			updateSecretResponse: mgclients.Client{},
			updateSecretErr:      svcerr.ErrUpdateEntity,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:                 "reset secret with a too long secret",
			newSecret:            strings.Repeat("strongSecret", 10),
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: client,
			err:                  errHashPassword,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall1 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		err := svc.ResetSecret(context.Background(), tc.session, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			repoCall1.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), validID)
		}
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestViewProfile(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	client := users.User{
		ID: "clientID",
		Credentials: users.Credentials{
			Identity: "existingIdentity",
			Secret:   "Strongsecret",
		},
	}
	cases := []struct {
		desc                 string
		client               mgclients.Client
		session              authn.Session
		retrieveByIDResponse mgclients.Client
		retrieveByIDErr      error
		err                  error
	}{
		{
			desc:                 "view profile successfully",
			client:               client,
			session:              authn.Session{UserID: validID},
			retrieveByIDResponse: client,
			err:                  nil,
		},
		{
			desc:                 "view profile with invalid ID",
			client:               client,
			session:              authn.Session{UserID: wrongID},
			retrieveByIDResponse: mgclients.Client{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		_, err := svc.ViewProfile(context.Background(), tc.session)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), mock.Anything)
		repoCall.Unset()
	}
}

func TestOAuthCallback(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	cases := []struct {
		desc                       string
		user                       users.User
		retrieveByIdentityResponse users.User
		retrieveByIdentityErr      error
		saveResponse               mgclients.Client
		saveErr                    error
		addPoliciesErr             error
		deletePoliciesErr          error
		err                        error
	}{
		{
			desc: "oauth signin callback with successfully",
			user: users.User{
				Credentials: users.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with successfully",
			user: users.User{
				Credentials: users.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityErr: repoerr.ErrNotFound,
			saveResponse: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with unknown error",
			user: users.User{
				Credentials: users.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityErr: repoerr.ErrMalformedEntity,
			err:                   repoerr.ErrMalformedEntity,
		},
		{
			desc: "oauth signup callback with failed to register user",
			user: users.User{
				Credentials: users.Credentials{
					Identity: "test@example.com",
				},
			},
			addPoliciesErr:        svcerr.ErrAuthorization,
			retrieveByIdentityErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrAuthorization,
		},
		{
			desc: "oauth signin callback with user not in the platform",
			user: users.User{
				Credentials: users.Credentials{
					Identity: "test@example.com",
				},
			},
			retrieveByIdentityResponse: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.UserRole,
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.retrieveByIdentityResponse, tc.retrieveByIdentityErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.saveResponse, tc.saveErr)
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesErr)
		_, err := svc.OAuthCallback(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.user.Credentials.Identity)
		repoCall.Unset()
		repoCall1.Unset()
		policyCall.Unset()
	}
}
