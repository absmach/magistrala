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
	validCMetadata = users.Metadata{"role": "user"}
	userID         = "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f"
	user           = users.User{
		ID:          userID,
		FirstName:   "firstname",
		LastName:    "lastname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: users.Credentials{Username: "username", Secret: secret},
		Email:       "useremail@email.com",
		Metadata:    validCMetadata,
		Status:      users.EnabledStatus,
	}
	basicUser = users.User{
		Credentials: users.Credentials{
			Username: "username",
		},
		ID:        userID,
		FirstName: "firstname",
		LastName:  "lastname",
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
	tokenUser := new(authmocks.TokenServiceClient)
	return users.NewService(tokenUser, cRepo, policies, e, phasher, idProvider), cRepo
}

func TestRegister(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	cases := []struct {
		desc                      string
		user                      users.User
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		err                       error
	}{
		{
			desc: "register new user successfully",
			user: user,
			err:  nil,
		},
		{
			desc:    "register existing user",
			user:    user,
			saveErr: repoerr.ErrConflict,
			err:     repoerr.ErrConflict,
		},
		{
			desc: "register a new enabled user with name",
			user: users.User{
				FirstName: "userWithName",
				Email:     "newuserwithname@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Status: users.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "register a new disabled user with name",
			user: users.User{
				FirstName: "userWithName",
				Email:     "newuserwithname@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
			},
			err: nil,
		},
		{
			desc: "register a new user with all fields",
			user: users.User{
				FirstName: "newuserwithallfields",
				Tags:      []string{"tag1", "tag2"},
				Email:     "newuserwithallfields@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Metadata: users.Metadata{
					"name": "newuserwithallfields",
				},
				Status: users.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "register a new user with missing email",
			user: users.User{
				FirstName: "userWithMissingEmail",
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
				FirstName: "userWithMissingSecret",
				Email:     "userwithmissingsecret@example.com",
				Credentials: users.Credentials{
					Secret: "",
				},
			},
			err: nil,
		},
		{
			desc: " register a user with a secret that is too long",
			user: users.User{
				FirstName: "clientWithLongSecret",
				Email:     "clientwithlongsecret@example.com",
				Credentials: users.Credentials{
					Secret: strings.Repeat("a", 73),
				},
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "register a new user with invalid status",
			user: users.User{
				FirstName: "userWithInvalidStatus",
				Email:     "user with invalid status",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Status: users.AllStatus,
			},
			err: svcerr.ErrInvalidStatus,
		},
		{
			desc: "register a new user with invalid role",
			user: users.User{
				FirstName: "clientWithInvalidRole",
				Email:     "clientwithinvalidrole@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Role: 2,
			},
			err: svcerr.ErrInvalidRole,
		},
		{
			desc: "register a new user with failed to add policies with err",
			user: users.User{
				FirstName: "clientWithFailedToAddPolicies",
				Email:     "clientwithfailedpolicies@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Role: users.AdminRole,
			},
			addPoliciesResponseErr: svcerr.ErrAddPolicies,
			err:                    svcerr.ErrAddPolicies,
		},
		{
			desc: "register a new user with failed to delete policies with err",
			user: users.User{
				FirstName: "clientWithFailedToDeletePolicies",
				Email:     "clientwithfailedtodelete@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
				Role: users.AdminRole,
			},
			deletePoliciesResponseErr: svcerr.ErrConflict,
			saveErr:                   repoerr.ErrConflict,
			err:                       svcerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponseErr)
		policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponseErr)
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.user, tc.saveErr)
		expected, err := svc.Register(context.Background(), authn.Session{}, tc.user, true)
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
		user                      users.User
		session                   authn.Session
		addPoliciesResponseErr    error
		deletePoliciesResponseErr error
		saveErr                   error
		checkSuperAdminErr        error
		err                       error
	}{
		{
			desc:    "register new user successfully as admin",
			user:    user,
			session: authn.Session{UserID: validID, SuperAdmin: true},
			err:     nil,
		},
		{
			desc:               "register a new user as admin with failed check on super admin",
			user:               user,
			session:            authn.Session{UserID: validID, SuperAdmin: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases2 {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesResponseErr)
		policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesResponseErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.user, tc.saveErr)
		expected, err := svc.Register(context.Background(), authn.Session{UserID: validID}, tc.user, false)
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

func TestViewUser(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	cases := []struct {
		desc                 string
		token                string
		reqUserID            string
		clientID             string
		retrieveByIDResponse users.User
		response             users.User
		identifyErr          error
		authorizeErr         error
		retrieveByIDErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "view user as normal user successfully",
			retrieveByIDResponse: user,
			response:             user,
			token:                validToken,
			reqUserID:            user.ID,
			clientID:             user.ID,
			err:                  nil,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view user as normal user with failed to retrieve user",
			retrieveByIDResponse: users.User{},
			token:                validToken,
			reqUserID:            user.ID,
			clientID:             user.ID,
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  svcerr.ErrNotFound,
			checkSuperAdminErr:   svcerr.ErrAuthorization,
		},
		{
			desc:                 "view user as admin user successfully",
			retrieveByIDResponse: user,
			response:             user,
			token:                validToken,
			reqUserID:            user.ID,
			clientID:             user.ID,
			err:                  nil,
		},
		{
			desc:                 "view user as admin user with failed check on super admin",
			token:                validToken,
			retrieveByIDResponse: basicUser,
			response:             basicUser,
			reqUserID:            user.ID,
			clientID:             "",
			checkSuperAdminErr:   svcerr.ErrAuthorization,
			err:                  nil,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		rUser, err := svc.View(context.Background(), authn.Session{UserID: tc.reqUserID}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestListUsers(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	cases := []struct {
		desc                string
		token               string
		page                users.Page
		retrieveAllResponse users.UsersPage
		response            users.UsersPage
		size                uint64
		retrieveAllErr      error
		superAdminErr       error
		err                 error
	}{
		{
			desc: "list clients as admin successfully",
			page: users.Page{
				Total: 1,
			},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			response: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "list clients as admin with failed to retrieve clients",
			page: users.Page{
				Total: 1,
			},
			retrieveAllResponse: users.UsersPage{},
			token:               validToken,
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrViewEntity,
		},
		{
			desc: "list clients as admin with failed check on super admin",
			page: users.Page{
				Total: 1,
			},
			token:         validToken,
			superAdminErr: svcerr.ErrAuthorization,
			err:           svcerr.ErrAuthorization,
		},
		{
			desc: "list clients as normal user with failed to retrieve clients",
			page: users.Page{
				Total: 1,
			},
			retrieveAllResponse: users.UsersPage{},
			token:               validToken,
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.superAdminErr)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListUsers(context.Background(), authn.Session{UserID: user.ID}, tc.page)
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
		page        users.Page
		response    users.UsersPage
		responseErr error
		err         error
	}{
		{
			desc:  "search clients with valid token",
			token: validToken,
			page:  users.Page{Offset: 0, FirstName: "clientname", Limit: 100},
			response: users.UsersPage{
				Page:  users.Page{Total: 1, Offset: 0, Limit: 100},
				Users: []users.User{user},
			},
		},
		{
			desc:  "search clients with id",
			token: validToken,
			page:  users.Page{Offset: 0, Id: "d8dd12ef-aa2a-43fe-8ef2-2e4fe514360f", Limit: 100},
			response: users.UsersPage{
				Page:  users.Page{Total: 1, Offset: 0, Limit: 100},
				Users: []users.User{user},
			},
		},
		{
			desc:  "search clients with random name",
			token: validToken,
			page:  users.Page{Offset: 0, FirstName: "randomname", Limit: 100},
			response: users.UsersPage{
				Page:  users.Page{Total: 0, Offset: 0, Limit: 100},
				Users: []users.User{},
			},
		},
		{
			desc:  "search clients with repo failed",
			token: validToken,
			page:  users.Page{Offset: 0, FirstName: "randomname", Limit: 100},
			response: users.UsersPage{
				Page: users.Page{Total: 0, Offset: 0, Limit: 0},
			},
			responseErr: repoerr.ErrViewEntity,
			err:         svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("SearchUsers", context.Background(), mock.Anything).Return(tc.response, tc.responseErr)
		page, err := svc.SearchUsers(context.Background(), tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
	}
}

func TestUpdateUser(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user1 := user
	user2 := user
	user1.FirstName = "Updated user"
	user2.Metadata = users.Metadata{"role": "test"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc               string
		user               users.User
		session            authn.Session
		updateResponse     users.User
		token              string
		updateErr          error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:           "update user name  successfully as normal user",
			user:           user1,
			session:        authn.Session{UserID: user1.ID},
			updateResponse: user1,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update metadata successfully as normal user",
			user:           user2,
			session:        authn.Session{UserID: user2.ID},
			updateResponse: user2,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update user name as normal user with repo error on update",
			user:           user1,
			session:        authn.Session{UserID: user1.ID},
			updateResponse: users.User{},
			token:          validToken,
			updateErr:      errors.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
		{
			desc:           "update user name as admin successfully",
			user:           user1,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: user1,
			token:          validToken,
			err:            nil,
		},
		{
			desc:           "update user metadata as admin successfully",
			user:           user2,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: user2,
			token:          validToken,
			err:            nil,
		},
		{
			desc:               "update user with failed check on super admin",
			user:               user1,
			session:            authn.Session{UserID: adminID},
			token:              validToken,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:           "update user name as admin with repo error on update",
			user:           user1,
			session:        authn.Session{UserID: adminID, SuperAdmin: true},
			updateResponse: users.User{},
			token:          validToken,
			updateErr:      errors.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.err)
		updatedUser, err := svc.Update(context.Background(), tc.session, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateUserTags(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user.Tags = []string{"updated"}
	adminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                   string
		user                   users.User
		session                authn.Session
		updateUserTagsResponse users.User
		updateUserTagsErr      error
		checkSuperAdminErr     error
		err                    error
	}{
		{
			desc:                   "update user tags as normal user successfully",
			user:                   user,
			session:                authn.Session{UserID: user.ID},
			updateUserTagsResponse: user,
			err:                    nil,
		},
		{
			desc:                   "update user tags as normal user with repo error on update",
			user:                   user,
			session:                authn.Session{UserID: user.ID},
			updateUserTagsResponse: users.User{},
			updateUserTagsErr:      errors.ErrMalformedEntity,
			err:                    svcerr.ErrUpdateEntity,
		},
		{
			desc:    "update user tags as admin successfully",
			user:    user,
			session: authn.Session{UserID: adminID, SuperAdmin: true},
			err:     nil,
		},
		{
			desc:               "update user tags as admin with failed check on super admin",
			user:               user,
			session:            authn.Session{UserID: adminID},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                   "update user tags as admin with repo error on update",
			user:                   user,
			session:                authn.Session{UserID: adminID, SuperAdmin: true},
			updateUserTagsResponse: users.User{},
			updateUserTagsErr:      errors.ErrMalformedEntity,
			err:                    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateUserTagsResponse, tc.updateUserTagsErr)
		updatedUser, err := svc.UpdateTags(context.Background(), tc.session, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateUserTagsResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateUserTagsResponse, updatedUser))

		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateUserRole(t *testing.T) {
	svc, _, cRepo, policies, _ := newService()

	user2 := user
	user.Role = users.AdminRole
	user2.Role = users.UserRole

	cases := []struct {
		desc               string
		user               users.User
		session            authn.Session
		updateRoleResponse users.User
		deletePolicyErr    error
		addPolicyErr       error
		updateRoleErr      error
		checkSuperAdminErr error
		err                error
	}{
		{
			desc:               "update user role successfully",
			user:               user,
			session:            authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleResponse: user,
			err:                nil,
		},
		{
			desc:               "update user role with failed check on super admin",
			user:               user,
			session:            authn.Session{UserID: validID, SuperAdmin: false},
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:         "update user role with failed to add policies",
			user:         user,
			session:      authn.Session{UserID: validID, SuperAdmin: true},
			addPolicyErr: errors.ErrMalformedEntity,
			err:          svcerr.ErrAddPolicies,
		},
		{
			desc:               "update user role to user role successfully  ",
			user:               user2,
			session:            authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleResponse: user2,
			err:                nil,
		},
		{
			desc:            "update user role to user role with failed to delete policies",
			user:            user2,
			session:         authn.Session{UserID: validID, SuperAdmin: true},
			deletePolicyErr: svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:            "update user role to user role with failed to delete policies with error",
			user:            user2,
			session:         authn.Session{UserID: validID, SuperAdmin: true},
			deletePolicyErr: svcerr.ErrMalformedEntity,
			err:             svcerr.ErrDeletePolicies,
		},
		{
			desc:          "Update user with failed repo update and roll back",
			user:          user,
			session:       authn.Session{UserID: validID, SuperAdmin: true},
			updateRoleErr: svcerr.ErrAuthentication,
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:            "Update user with failed repo update and failedroll back",
			user:            user,
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
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateRoleResponse, tc.updateRoleErr)

		updatedUser, err := svc.UpdateRole(context.Background(), tc.session, tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateRoleResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateRoleResponse, updatedUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		policyCall.Unset()
		policyCall1.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateUserSecret(t *testing.T) {
	svc, authUser, cRepo, _, _ := newService()

	newSecret := "newstrongSecret"
	rUser := user
	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)
	responseUser := user
	responseUser.Credentials.Secret = newSecret

	cases := []struct {
		desc                    string
		oldSecret               string
		newSecret               string
		session                 authn.Session
		retrieveByIDResponse    users.User
		retrieveByEmailResponse users.User
		updateSecretResponse    users.User
		issueResponse           *magistrala.Token
		response                users.User
		retrieveByIDErr         error
		retrieveByEmailErr      error
		updateSecretErr         error
		issueErr                error
		err                     error
	}{
		{
			desc:                    "update user secret with valid token",
			oldSecret:               user.Credentials.Secret,
			newSecret:               newSecret,
			session:                 authn.Session{UserID: user.ID},
			retrieveByEmailResponse: rUser,
			retrieveByIDResponse:    user,
			updateSecretResponse:    responseUser,
			issueResponse:           &magistrala.Token{AccessToken: validToken},
			response:                responseUser,
			err:                     nil,
		},
		{
			desc:                 "update user secret with failed to retrieve user by ID",
			oldSecret:            user.Credentials.Secret,
			newSecret:            newSecret,
			session:              authn.Session{UserID: user.ID},
			retrieveByIDResponse: users.User{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                    "update user secret with failed to retrieve user by email",
			oldSecret:               user.Credentials.Secret,
			newSecret:               newSecret,
			session:                 authn.Session{UserID: user.ID},
			retrieveByIDResponse:    user,
			retrieveByEmailResponse: users.User{},
			retrieveByEmailErr:      repoerr.ErrNotFound,
			err:                     repoerr.ErrNotFound,
		},
		{
			desc:                    "update user secret with invalod old secret",
			oldSecret:               "invalid",
			newSecret:               newSecret,
			session:                 authn.Session{UserID: user.ID},
			retrieveByIDResponse:    user,
			retrieveByEmailResponse: rUser,
			err:                     svcerr.ErrLogin,
		},
		{
			desc:                    "update user secret with too long new secret",
			oldSecret:               user.Credentials.Secret,
			newSecret:               strings.Repeat("a", 73),
			session:                 authn.Session{UserID: user.ID},
			retrieveByIDResponse:    user,
			retrieveByEmailResponse: rUser,
			err:                     repoerr.ErrMalformedEntity,
		},
		{
			desc:                    "update user secret with failed to update secret",
			oldSecret:               user.Credentials.Secret,
			newSecret:               newSecret,
			session:                 authn.Session{UserID: user.ID},
			retrieveByIDResponse:    user,
			retrieveByEmailResponse: rUser,
			updateSecretResponse:    users.User{},
			updateSecretErr:         repoerr.ErrMalformedEntity,
			err:                     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), user.ID).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall1 := cRepo.On("RetrieveByEmail", context.Background(), user.Email).Return(tc.retrieveByEmailResponse, tc.retrieveByEmailErr)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateSecretErr)
		authCall := authUser.On("Issue", context.Background(), mock.Anything).Return(tc.issueResponse, tc.issueErr)
		updatedUser, err := svc.UpdateSecret(context.Background(), tc.session, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedUser))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByEmail", context.Background(), tc.response.Email)
			assert.True(t, ok, fmt.Sprintf("RetrieveByEmail was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		authCall.Unset()
	}
}

func TestUpdateUserEmail(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user2 := user
	user2.Email = "updated@example.com"

	cases := []struct {
		desc                string
		email               string
		token               string
		reqUserID           string
		id                  string
		updateEmailResponse users.User
		updateEmailErr      error
		checkSuperAdminErr  error
		err                 error
	}{
		{
			desc:                "update user as normal user successfully",
			email:               "updated@example.com",
			token:               validToken,
			reqUserID:           user.ID,
			id:                  user.ID,
			updateEmailResponse: user2,
			err:                 nil,
		},
		{
			desc:                "update user email as normal user with repo error on update",
			email:               "updated@example.com",
			token:               validToken,
			reqUserID:           user.ID,
			id:                  user.ID,
			updateEmailResponse: users.User{},
			updateEmailErr:      errors.ErrMalformedEntity,
			err:                 svcerr.ErrUpdateEntity,
		},
		{
			desc:  "update user email as admin successfully",
			email: "updated@example.com",
			token: validToken,
			id:    user.ID,
			err:   nil,
		},
		{
			desc:                "update user email as admin with repo error on update",
			email:               "updated@exmaple.com",
			token:               validToken,
			reqUserID:           user.ID,
			id:                  user.ID,
			updateEmailResponse: users.User{},
			updateEmailErr:      errors.ErrMalformedEntity,
			err:                 svcerr.ErrUpdateEntity,
		},
		{
			desc:                "update user as admin user with failed check on super admin",
			email:               "updated@exmaple.com",
			token:               validToken,
			reqUserID:           user.ID,
			id:                  "",
			updateEmailResponse: users.User{},
			updateEmailErr:      errors.ErrMalformedEntity,
			checkSuperAdminErr:  svcerr.ErrAuthorization,
			err:                 svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateEmailResponse, tc.updateEmailErr)
		updatedUser, err := svc.UpdateEmail(context.Background(), authn.Session{DomainUserID: tc.reqUserID, UserID: validID, DomainID: validID}, tc.id, tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateEmailResponse, updatedUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateEmailResponse, updatedUser))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableUser(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "user1@example.com", Secret: "password"}, Status: users.EnabledStatus}
	disabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "user3@example.com", Secret: "password"}, Status: users.DisabledStatus}
	endisabledUser1 := disabledUser1
	endisabledUser1.Status = users.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		user                 users.User
		retrieveByIDResponse users.User
		changeStatusResponse users.User
		response             users.User
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "enable disabled user",
			id:                   disabledUser1.ID,
			user:                 disabledUser1,
			retrieveByIDResponse: disabledUser1,
			changeStatusResponse: endisabledUser1,
			response:             endisabledUser1,
			err:                  nil,
		},
		{
			desc:               "enable disabled user with normal user token",
			id:                 disabledUser1.ID,
			user:               disabledUser1,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "enable disabled user with failed to retrieve user by ID",
			id:                   disabledUser1.ID,
			user:                 disabledUser1,
			retrieveByIDResponse: users.User{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "enable already enabled user",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			retrieveByIDResponse: enabledUser1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable disabled user with failed to change status",
			id:                   disabledUser1.ID,
			user:                 disabledUser1,
			retrieveByIDResponse: disabledUser1,
			changeStatusResponse: users.User{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.Enable(context.Background(), authn.Session{}, tc.id)
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

func TestDisableUser(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "client1@example.com", Secret: "password"}, Status: users.EnabledStatus}
	disabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "client3@example.com", Secret: "password"}, Status: users.DisabledStatus}
	disenabledUser1 := enabledUser1
	disenabledUser1.Status = users.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		user                 users.User
		retrieveByIDResponse users.User
		changeStatusResponse users.User
		response             users.User
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "disable enabled user",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			retrieveByIDResponse: enabledUser1,
			changeStatusResponse: disenabledUser1,
			response:             disenabledUser1,
			err:                  nil,
		},
		{
			desc:               "disable enabled user with normal user token",
			id:                 enabledUser1.ID,
			user:               enabledUser1,
			checkSuperAdminErr: svcerr.ErrAuthorization,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc:                 "disable enabled user with failed to retrieve user by ID",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			retrieveByIDResponse: users.User{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable already disabled user",
			id:                   disabledUser1.ID,
			user:                 disabledUser1,
			retrieveByIDResponse: disabledUser1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable enabled user with failed to change status",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			changeStatusResponse: users.User{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)

		_, err := svc.Disable(context.Background(), authn.Session{}, tc.id)
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

func TestDeleteUser(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	enabledUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "user1@example.com", Secret: "password"}, Status: users.EnabledStatus}
	deletedUser1 := users.User{ID: testsutil.GenerateUUID(t), Credentials: users.Credentials{Username: "user3@example.com", Secret: "password"}, Status: users.DeletedStatus}
	disenabledUser1 := enabledUser1
	disenabledUser1.Status = users.DeletedStatus

	cases := []struct {
		desc                 string
		id                   string
		session              authn.Session
		user                 users.User
		retrieveByIDResponse users.User
		changeStatusResponse users.User
		response             users.User
		retrieveByIDErr      error
		changeStatusErr      error
		checkSuperAdminErr   error
		err                  error
	}{
		{
			desc:                 "delete enabled user",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: enabledUser1,
			changeStatusResponse: disenabledUser1,
			response:             disenabledUser1,
			err:                  nil,
		},
		{
			desc:                 "delete enabled user with failed to retrieve user by ID",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: users.User{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "delete already deleted user",
			id:                   deletedUser1.ID,
			user:                 deletedUser1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: deletedUser1,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "delete enabled user with failed to change status",
			id:                   enabledUser1.ID,
			user:                 enabledUser1,
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: enabledUser1,
			changeStatusResponse: users.User{},
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall2 := cRepo.On("CheckSuperAdmin", context.Background(), mock.Anything).Return(tc.checkSuperAdminErr)
		repoCall3 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveByIDResponse, tc.retrieveByIDErr)
		repoCall4 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		err := svc.Delete(context.Background(), tc.session, tc.id)
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
		page                    users.Page
		listAllSubjectsReq      policysvc.Policy
		listAllSubjectsResponse policysvc.PolicyPage
		retrieveAllResponse     users.UsersPage
		listPermissionsResponse policysvc.Permissions
		response                users.MembersPage
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
			page:                    users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			response: users.MembersPage{
				Page: users.Page{
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			response: users.MembersPage{
				Page: users.Page{
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{basicUser},
			},
			listPermissionsResponse: []string{"read"},
			response: users.MembersPage{
				Page: users.Page{
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read", ListPerms: true},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			listPermissionsResponse: []string{},
			response:                users.MembersPage{},
			listPermissionErr:       svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:       "list members with of the things kind with failed to list all subjects",
			groupID:    validID,
			objectKind: policysvc.ThingsKind,
			objectID:   validID,
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read"},
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.ThingType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse:     users.UsersPage{},
			response:                users.MembersPage{},
			retrieveAllErr:          repoerr.ErrNotFound,
			err:                     repoerr.ErrNotFound,
		},
		{
			desc:                    "list members with no policies successfully of the domain kind",
			groupID:                 validID,
			objectKind:              policysvc.DomainsKind,
			objectID:                validID,
			page:                    users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.DomainType,
			},
			response: users.MembersPage{
				Page: users.Page{
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.DomainType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{basicUser},
			},
			response: users.MembersPage{
				Page: users.Page{
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
			page:                    users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsResponse: policysvc.PolicyPage{},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.GroupType,
			},
			response: users.MembersPage{
				Page: users.Page{
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
			page:       users.Page{Offset: 0, Limit: 100, Permission: "read"},
			listAllSubjectsReq: policysvc.Policy{
				SubjectType: policysvc.UserType,
				Permission:  "read",
				Object:      validID,
				ObjectType:  policysvc.GroupType,
			},
			listAllSubjectsResponse: policysvc.PolicyPage{Policies: []string{validPolicy}},
			retrieveAllResponse: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Users: []users.User{user},
			},
			response: users.MembersPage{
				Page: users.Page{
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
		desc                    string
		domainID                string
		user                    users.User
		retrieveByEmailResponse users.User
		issueResponse           *magistrala.Token
		retrieveByEmailErr      error
		issueErr                error
		err                     error
	}{
		{
			desc:                    "issue token for an existing user",
			user:                    user,
			retrieveByEmailResponse: rUser,
			issueResponse:           &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                     nil,
		},
		{
			desc:                    "issue token for non-empty domain id",
			domainID:                validID,
			user:                    user,
			retrieveByEmailResponse: rUser,
			issueResponse:           &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                     nil,
		},
		{
			desc:                    "issue token for a non-existing user",
			user:                    user,
			retrieveByEmailResponse: users.User{},
			retrieveByEmailErr:      repoerr.ErrNotFound,
			err:                     repoerr.ErrNotFound,
		},
		{
			desc:                    "issue token for a user with wrong secret",
			user:                    user,
			retrieveByEmailResponse: rUser3,
			err:                     svcerr.ErrLogin,
		},
		{
			desc:                    "issue token with empty domain id",
			user:                    user,
			retrieveByEmailResponse: rUser,
			issueResponse:           &magistrala.Token{},
			issueErr:                svcerr.ErrAuthentication,
			err:                     svcerr.ErrAuthentication,
		},
		{
			desc:                    "issue token with grpc error",
			user:                    user,
			retrieveByEmailResponse: rUser,
			issueResponse:           &magistrala.Token{},
			issueErr:                svcerr.ErrAuthentication,
			err:                     svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByEmail", context.Background(), tc.user.Email).Return(tc.retrieveByEmailResponse, tc.retrieveByEmailErr)
		authCall := auth.On("Issue", context.Background(), &magistrala.IssueReq{UserId: tc.user.ID, DomainId: &tc.domainID, Type: uint32(mgauth.AccessKey)}).Return(tc.issueResponse, tc.issueErr)
		token, err := svc.IssueToken(context.Background(), tc.user.Email, tc.user.Credentials.Secret, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByEmail", context.Background(), tc.user.Email)
			assert.True(t, ok, fmt.Sprintf("RetrieveByEmail was not called on %s", tc.desc))
			ok = authCall.Parent.AssertCalled(t, "Issue", context.Background(), &magistrala.IssueReq{UserId: tc.user.ID, DomainId: &tc.domainID, Type: uint32(mgauth.AccessKey)})
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
		repoResp    users.User
		repoErr     error
		err         error
	}{
		{
			desc:        "refresh token with refresh token for an existing user",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID:    validID,
			refreshResp: &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:    rUser,
			err:         nil,
		},
		{
			desc:        "refresh token with refresh token for empty domain id",
			session:     authn.Session{UserID: validID},
			refreshResp: &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			repoResp:    rUser,
			err:         nil,
		},
		{
			desc:        "refresh token with access token for an existing user",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID:    validID,
			refreshResp: &magistrala.Token{},
			refresErr:   svcerr.ErrAuthentication,
			repoResp:    rUser,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "refresh token with refresh token for a non-existing user",
			session:  authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID: validID,
			repoErr:  repoerr.ErrNotFound,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "refresh token with refresh token for a disable user",
			session:  authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			domainID: validID,
			repoResp: users.User{Status: users.DisabledStatus},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:        "refresh token with empty domain id",
			session:     authn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			refreshResp: &magistrala.Token{},
			refresErr:   svcerr.ErrAuthentication,
			repoResp:    rUser,
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
		desc                    string
		email                   string
		host                    string
		retrieveByEmailResponse users.User
		issueResponse           *magistrala.Token
		retrieveByEmailErr      error
		issueErr                error
		err                     error
	}{
		{
			desc:                    "generate reset token for existing user",
			email:                   "existingemail@example.com",
			host:                    "examplehost",
			retrieveByEmailResponse: user,
			issueResponse:           &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"},
			err:                     nil,
		},
		{
			desc:  "generate reset token for user with non-existing user",
			email: "example@example.com",
			host:  "examplehost",
			retrieveByEmailResponse: users.User{
				ID:    testsutil.GenerateUUID(t),
				Email: "",
			},
			retrieveByEmailErr: repoerr.ErrNotFound,
			err:                repoerr.ErrNotFound,
		},
		{
			desc:                    "generate reset token with failed to issue token",
			email:                   "existingemail@example.com",
			host:                    "examplehost",
			retrieveByEmailResponse: user,
			issueResponse:           &magistrala.Token{},
			issueErr:                svcerr.ErrAuthorization,
			err:                     svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByEmail", context.Background(), tc.email).Return(tc.retrieveByEmailResponse, tc.retrieveByEmailErr)
		authCall := auth.On("Issue", context.Background(), mock.Anything).Return(tc.issueResponse, tc.issueErr)
		svcCall := e.On("SendPasswordReset", []string{tc.email}, tc.host, user.Credentials.Username, validToken).Return(tc.err)
		err := svc.GenerateResetToken(context.Background(), tc.email, tc.host)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByEmail", context.Background(), tc.email)
		repoCall.Unset()
		authCall.Unset()
		svcCall.Unset()
	}
}

func TestResetSecret(t *testing.T) {
	svc, cRepo := newServiceMinimal()

	user := users.User{
		ID:    "userID",
		Email: "test@example.com",
		Credentials: users.Credentials{
			Secret: "Strongsecret",
		},
	}

	cases := []struct {
		desc                 string
		newSecret            string
		session              authn.Session
		retrieveByIDResponse users.User
		updateSecretResponse users.User
		retrieveByIDErr      error
		updateSecretErr      error
		err                  error
	}{
		{
			desc:                 "reset secret with successfully",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: user,
			updateSecretResponse: users.User{
				ID:    "userID",
				Email: "test@example.com",
				Credentials: users.Credentials{
					Secret: "newStrongSecret",
				},
			},
			err: nil,
		},
		{
			desc:                 "reset secret with invalid ID",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: users.User{},
			retrieveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:      "reset secret with empty email",
			session:   authn.Session{UserID: validID, SuperAdmin: true},
			newSecret: "newStrongSecret",
			retrieveByIDResponse: users.User{
				ID:    "userID",
				Email: "",
			},
			err: nil,
		},
		{
			desc:                 "reset secret with failed to update secret",
			newSecret:            "newStrongSecret",
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: user,
			updateSecretResponse: users.User{},
			updateSecretErr:      svcerr.ErrUpdateEntity,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:                 "reset secret with a too long secret",
			newSecret:            strings.Repeat("strongSecret", 10),
			session:              authn.Session{UserID: validID, SuperAdmin: true},
			retrieveByIDResponse: user,
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

	user := users.User{
		ID:    "userID",
		Email: "existingIdentity",
		Credentials: users.Credentials{
			Secret: "Strongsecret",
		},
	}
	cases := []struct {
		desc                 string
		user                 users.User
		session              authn.Session
		retrieveByIDResponse users.User
		retrieveByIDErr      error
		err                  error
	}{
		{
			desc:                 "view profile successfully",
			user:                 user,
			session:              authn.Session{UserID: validID},
			retrieveByIDResponse: user,
			err:                  nil,
		},
		{
			desc:                 "view profile with invalid ID",
			user:                 user,
			session:              authn.Session{UserID: wrongID},
			retrieveByIDResponse: users.User{},
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
		desc                    string
		user                    users.User
		retrieveByEmailResponse users.User
		retrieveByEmailErr      error
		saveResponse            users.User
		saveErr                 error
		addPoliciesErr          error
		deletePoliciesErr       error
		err                     error
	}{
		{
			desc: "oauth signin callback with successfully",
			user: users.User{
				Email: "test@example.com",
			},
			retrieveByEmailResponse: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: users.UserRole,
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with successfully",
			user: users.User{
				Email: "test@example.com",
			},
			retrieveByEmailErr: repoerr.ErrNotFound,
			saveResponse: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: users.UserRole,
			},
			err: nil,
		},
		{
			desc: "oauth signup callback with unknown error",
			user: users.User{
				Email: "test@example.com",
			},
			retrieveByEmailErr: repoerr.ErrMalformedEntity,
			err:                repoerr.ErrMalformedEntity,
		},
		{
			desc: "oauth signup callback with failed to register user",
			user: users.User{
				Email: "test@example.com",
			},
			addPoliciesErr:     svcerr.ErrAuthorization,
			retrieveByEmailErr: repoerr.ErrNotFound,
			err:                svcerr.ErrAuthorization,
		},
		{
			desc: "oauth signin callback with user not in the platform",
			user: users.User{
				Email: "test@example.com",
			},
			retrieveByEmailResponse: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: users.UserRole,
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByEmail", context.Background(), tc.user.Email).Return(tc.retrieveByEmailResponse, tc.retrieveByEmailErr)
		repoCall1 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.saveResponse, tc.saveErr)
		policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesErr)
		_, err := svc.OAuthCallback(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Parent.AssertCalled(t, "RetrieveByEmail", context.Background(), tc.user.Email)
		repoCall.Unset()
		repoCall1.Unset()
		policyCall.Unset()
	}
}
