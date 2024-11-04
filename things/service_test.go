// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
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
	validTMetadata = things.Metadata{"role": "thing"}
	ID             = "6e5e10b3-d4df-4758-b426-4929d55ad740"
	thing          = things.Client{
		ID:          ID,
		Name:        "thingname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: things.Credentials{Identity: "thingidentity", Secret: secret},
		Metadata:    validTMetadata,
		Status:      things.EnabledStatus,
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
	cache      *mocks.Cache
	cRepo      *mocks.Repository
)

func newService() things.Service {
	pService = new(policymocks.Service)
	pEvaluator = new(policymocks.Evaluator)
	cache = new(mocks.Cache)
	idProvider := uuid.NewMock()
	cRepo = new(mocks.Repository)

	return things.NewService(pEvaluator, pService, cRepo, cache, idProvider)
}

func TestCreateClients(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		thing           things.Client
		token           string
		addPolicyErr    error
		deletePolicyErr error
		saveErr         error
		err             error
	}{
		{
			desc:  "create a new thing successfully",
			thing: thing,
			token: validToken,
			err:   nil,
		},
		{
			desc:    "create an existing thing",
			thing:   thing,
			token:   validToken,
			saveErr: repoerr.ErrConflict,
			err:     repoerr.ErrConflict,
		},
		{
			desc: "create a new thing without secret",
			thing: things.Client{
				Name: "thingWithoutSecret",
				Credentials: things.Credentials{
					Identity: "newthingwithoutsecret@example.com",
				},
				Status: things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing without identity",
			thing: things.Client{
				Name: "thingWithoutIdentity",
				Credentials: things.Credentials{
					Identity: "newthingwithoutsecret@example.com",
				},
				Status: things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with name",
			thing: things.Client{
				Name: "thingWithName",
				Credentials: things.Credentials{
					Identity: "newthingwithname@example.com",
					Secret:   secret,
				},
				Status: things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},

		{
			desc: "create a new disabled thing with name",
			thing: things.Client{
				Name: "thingWithName",
				Credentials: things.Credentials{
					Identity: "newthingwithname@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with tags",
			thing: things.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: things.Credentials{
					Identity: "newthingwithtags@example.com",
					Secret:   secret,
				},
				Status: things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing with tags",
			thing: things.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: things.Credentials{
					Identity: "newthingwithtags@example.com",
					Secret:   secret,
				},
				Status: things.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new enabled thing with metadata",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validTMetadata,
				Status:   things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing with metadata",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validTMetadata,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new disabled thing",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with valid disabled status",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithvalidstatus@example.com",
					Secret:   secret,
				},
				Status: things.DisabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with all fields",
			thing: things.Client{
				Name: "newthingwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: things.Credentials{
					Identity: "newthingwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: things.Metadata{
					"name": "newthingwithallfields",
				},
				Status: things.EnabledStatus,
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "create a new thing with invalid status",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: things.AllStatus,
			},
			token: validToken,
			err:   svcerr.ErrInvalidStatus,
		},
		{
			desc: "create a new thing with failed add policies response",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: things.EnabledStatus,
			},
			token:        validToken,
			addPolicyErr: svcerr.ErrInvalidPolicy,
			err:          svcerr.ErrInvalidPolicy,
		},
		{
			desc: "create a new thing with failed delete policies response",
			thing: things.Client{
				Credentials: things.Credentials{
					Identity: "newthingwithfailedpolicy@example.com",
					Secret:   secret,
				},
				Status: things.EnabledStatus,
			},
			token:           validToken,
			saveErr:         repoerr.ErrConflict,
			deletePolicyErr: svcerr.ErrInvalidPolicy,
			err:             repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return([]things.Client{tc.thing}, tc.saveErr)
		policyCall := pService.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		policyCall1 := pService.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolicyErr)
		expected, err := svc.CreateClients(context.Background(), mgauthn.Session{}, tc.thing)
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
	svc := newService()

	cases := []struct {
		desc        string
		clientID    string
		response    things.Client
		retrieveErr error
		err         error
	}{
		{
			desc:     "view thing successfully",
			response: thing,
			clientID: thing.ID,
			err:      nil,
		},
		{
			desc:     "view thing with an invalid token",
			response: things.Client{},
			clientID: "",
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:        "view thing with valid token and invalid thing id",
			response:    things.Client{},
			clientID:    wrongID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "view thing with an invalid token and invalid thing id",
			response: things.Client{},
			clientID: wrongID,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.response, tc.err)
		rThing, err := svc.View(context.Background(), mgauthn.Session{}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rThing))
		repoCall1.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc := newService()

	adminID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	nonAdminID := testsutil.GenerateUUID(t)
	thing.Permissions = []string{"read", "write"}

	cases := []struct {
		desc                    string
		userKind                string
		session                 mgauthn.Session
		page                    things.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     things.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                things.ClientsPage
		id                      string
		size                    uint64
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all things successfully as non admin",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{thing.ID, thing.ID}},
			retrieveAllResponse: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			listPermissionsResponse: []string{"read", "write"},
			response: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			err: nil,
		},
		{
			desc:     "list all things as non admin with failed to retrieve all",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{thing.ID, thing.ID}},
			retrieveAllResponse: things.ClientsPage{},
			response:            things.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all things as non admin with failed to list permissions",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{thing.ID, thing.ID}},
			retrieveAllResponse: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			listPermissionsResponse: []string{},
			response:                things.ClientsPage{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all things as non admin with failed super admin",
			userKind: "non-admin",
			session:  mgauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			response:            things.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
		},
		{
			desc:     "list all things as non admin with failed to list objects",
			userKind: "non-admin",
			id:       nonAdminID,
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
			},
			response:            things.ClientsPage{},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		listAllObjectsCall := pService.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
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
		page                    things.Page
		listObjectsResponse     policysvc.PolicyPage
		retrieveAllResponse     things.ClientsPage
		listPermissionsResponse policysvc.Permissions
		response                things.ClientsPage
		id                      string
		size                    uint64
		listObjectsErr          error
		retrieveAllErr          error
		listPermissionsErr      error
		err                     error
	}{
		{
			desc:     "list all things as admin successfully",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{thing.ID, thing.ID}},
			retrieveAllResponse: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			listPermissionsResponse: []string{"read", "write"},
			response: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			err: nil,
		},
		{
			desc:     "list all things as admin with failed to retrieve all",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: things.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all things as admin with failed to list permissions",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			retrieveAllResponse: things.ClientsPage{
				Page: things.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []things.Client{thing, thing},
			},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
		{
			desc:     "list all things as admin with failed to list things",
			userKind: "admin",
			id:       adminID,
			session:  mgauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: things.Page{
				Offset:    0,
				Limit:     100,
				ListPerms: true,
				Domain:    domainID,
			},
			retrieveAllResponse: things.ClientsPage{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		listAllObjectsCall := pService.On("ListAllObjects", context.Background(), policysvc.Policy{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.DomainID + "_" + adminID,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		listAllObjectsCall2 := pService.On("ListAllObjects", context.Background(), policysvc.Policy{
			SubjectType: policysvc.UserType,
			Subject:     tc.session.UserID,
			Permission:  "",
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		retrieveAllCall := cRepo.On("SearchClients", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
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

	thing1 := thing
	thing2 := thing
	thing1.Name = "Updated thing"
	thing2.Metadata = things.Metadata{"role": "test"}

	cases := []struct {
		desc           string
		thing          things.Client
		session        mgauthn.Session
		updateResponse things.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update thing name successfully",
			thing:          thing1,
			session:        mgauthn.Session{UserID: validID},
			updateResponse: thing1,
			err:            nil,
		},
		{
			desc:           "update thing metadata with valid token",
			thing:          thing2,
			updateResponse: thing2,
			session:        mgauthn.Session{UserID: validID},
			err:            nil,
		},
		{
			desc:           "update thing with failed to update repo",
			thing:          thing1,
			updateResponse: things.Client{},
			session:        mgauthn.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedThing, err := svc.Update(context.Background(), tc.session, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedThing))
		repoCall1.Unset()
	}
}

func TestUpdateTags(t *testing.T) {
	svc := newService()

	thing.Tags = []string{"updated"}

	cases := []struct {
		desc           string
		thing          things.Client
		session        mgauthn.Session
		updateResponse things.Client
		updateErr      error
		err            error
	}{
		{
			desc:           "update thing tags successfully",
			thing:          thing,
			session:        mgauthn.Session{UserID: validID},
			updateResponse: thing,
			err:            nil,
		},
		{
			desc:           "update thing tags with failed to update repo",
			thing:          thing,
			updateResponse: things.Client{},
			session:        mgauthn.Session{UserID: validID},
			updateErr:      repoerr.ErrMalformedEntity,
			err:            svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.updateResponse, tc.updateErr)
		updatedThing, err := svc.UpdateTags(context.Background(), tc.session, tc.thing)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateResponse, updatedThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateResponse, updatedThing))
		repoCall1.Unset()
	}
}

func TestUpdateSecret(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc                 string
		thing                things.Client
		newSecret            string
		updateSecretResponse things.Client
		session              mgauthn.Session
		updateErr            error
		err                  error
	}{
		{
			desc:      "update thing secret successfully",
			thing:     thing,
			newSecret: "newSecret",
			session:   mgauthn.Session{UserID: validID},
			updateSecretResponse: things.Client{
				ID: thing.ID,
				Credentials: things.Credentials{
					Identity: thing.Credentials.Identity,
					Secret:   "newSecret",
				},
			},
			err: nil,
		},
		{
			desc:                 "update thing secret with failed to update repo",
			thing:                thing,
			newSecret:            "newSecret",
			session:              mgauthn.Session{UserID: validID},
			updateSecretResponse: things.Client{},
			updateErr:            repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.updateSecretResponse, tc.updateErr)
		updatedThing, err := svc.UpdateSecret(context.Background(), tc.session, tc.thing.ID, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.updateSecretResponse, updatedThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.updateSecretResponse, updatedThing))
		repoCall.Unset()
	}
}

func TestEnable(t *testing.T) {
	svc := newService()

	enabledThing1 := things.Client{ID: ID, Credentials: things.Credentials{Identity: "thing1@example.com", Secret: "password"}, Status: things.EnabledStatus}
	disabledThing1 := things.Client{ID: ID, Credentials: things.Credentials{Identity: "thing3@example.com", Secret: "password"}, Status: things.DisabledStatus}
	endisabledThing1 := disabledThing1
	endisabledThing1.Status = things.EnabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              mgauthn.Session
		thing                things.Client
		changeStatusResponse things.Client
		retrieveByIDResponse things.Client
		changeStatusErr      error
		retrieveIDErr        error
		err                  error
	}{
		{
			desc:                 "enable disabled thing",
			id:                   disabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                disabledThing1,
			changeStatusResponse: endisabledThing1,
			retrieveByIDResponse: disabledThing1,
			err:                  nil,
		},
		{
			desc:                 "enable disabled thing with failed to update repo",
			id:                   disabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                disabledThing1,
			changeStatusResponse: things.Client{},
			retrieveByIDResponse: disabledThing1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "enable enabled thing",
			id:                   enabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                enabledThing1,
			changeStatusResponse: enabledThing1,
			retrieveByIDResponse: enabledThing1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "enable non-existing thing",
			id:                   wrongID,
			session:              mgauthn.Session{UserID: validID},
			thing:                things.Client{},
			changeStatusResponse: things.Client{},
			retrieveByIDResponse: things.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		_, err := svc.Enable(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisable(t *testing.T) {
	svc := newService()

	enabledThing1 := things.Client{ID: ID, Credentials: things.Credentials{Identity: "thing1@example.com", Secret: "password"}, Status: things.EnabledStatus}
	disabledThing1 := things.Client{ID: ID, Credentials: things.Credentials{Identity: "thing3@example.com", Secret: "password"}, Status: things.DisabledStatus}
	disenabledClient1 := enabledThing1
	disenabledClient1.Status = things.DisabledStatus

	cases := []struct {
		desc                 string
		id                   string
		session              mgauthn.Session
		thing                things.Client
		changeStatusResponse things.Client
		retrieveByIDResponse things.Client
		changeStatusErr      error
		retrieveIDErr        error
		removeErr            error
		err                  error
	}{
		{
			desc:                 "disable enabled thing",
			id:                   enabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                enabledThing1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledThing1,
			err:                  nil,
		},
		{
			desc:                 "disable thing with failed to update repo",
			id:                   enabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                enabledThing1,
			changeStatusResponse: things.Client{},
			retrieveByIDResponse: enabledThing1,
			changeStatusErr:      repoerr.ErrMalformedEntity,
			err:                  svcerr.ErrUpdateEntity,
		},
		{
			desc:                 "disable disabled thing",
			id:                   disabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                disabledThing1,
			changeStatusResponse: things.Client{},
			retrieveByIDResponse: disabledThing1,
			changeStatusErr:      errors.ErrStatusAlreadyAssigned,
			err:                  errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:                 "disable non-existing thing",
			id:                   wrongID,
			thing:                things.Client{},
			session:              mgauthn.Session{UserID: validID},
			changeStatusResponse: things.Client{},
			retrieveByIDResponse: things.Client{},
			retrieveIDErr:        repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "disable thing with failed to remove from cache",
			id:                   enabledThing1.ID,
			session:              mgauthn.Session{UserID: validID},
			thing:                disabledThing1,
			changeStatusResponse: disenabledClient1,
			retrieveByIDResponse: enabledThing1,
			removeErr:            svcerr.ErrRemoveEntity,
			err:                  svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.retrieveByIDResponse, tc.retrieveIDErr)
		repoCall1 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeStatusResponse, tc.changeStatusErr)
		repoCall2 := cache.On("Remove", mock.Anything, mock.Anything).Return(tc.removeErr)
		_, err := svc.Disable(context.Background(), tc.session, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc := newService()

	nThings := uint64(10)
	aThings := []things.Client{}
	domainID := testsutil.GenerateUUID(t)
	for i := uint64(0); i < nThings; i++ {
		identity := fmt.Sprintf("member_%d@example.com", i)
		thing := things.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: domainID,
			Name:   identity,
			Credentials: things.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: things.Metadata{"role": "thing"},
		}
		aThings = append(aThings, thing)
	}
	aThings[0].Permissions = []string{"admin"}

	cases := []struct {
		desc                     string
		groupID                  string
		page                     things.Page
		session                  mgauthn.Session
		listObjectsResponse      policysvc.PolicyPage
		listPermissionsResponse  policysvc.Permissions
		retreiveAllByIDsResponse things.ClientsPage
		response                 things.MembersPage
		identifyErr              error
		authorizeErr             error
		listObjectsErr           error
		listPermissionsErr       error
		retreiveAllByIDsErr      error
		err                      error
	}{
		{
			desc:                    "list members with authorized token",
			session:                 mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID:                 testsutil.GenerateUUID(t),
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			retreiveAllByIDsResponse: things.ClientsPage{
				Page: things.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []things.Client{},
			},
			response: things.MembersPage{
				Page: things.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Members: []things.Client{},
			},
			err: nil,
		},
		{
			desc:    "list members with offset and limit",
			session: mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: things.Page{
				Offset: 6,
				Limit:  nThings,
				Status: things.AllStatus,
			},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			retreiveAllByIDsResponse: things.ClientsPage{
				Page: things.Page{
					Total: nThings - 6 - 1,
				},
				Clients: aThings[6 : nThings-1],
			},
			response: things.MembersPage{
				Page: things.Page{
					Total: nThings - 6 - 1,
				},
				Members: aThings[6 : nThings-1],
			},
			err: nil,
		},
		{
			desc:                     "list members with an invalid id",
			session:                  mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID:                  wrongID,
			listObjectsResponse:      policysvc.PolicyPage{},
			listPermissionsResponse:  []string{},
			retreiveAllByIDsResponse: things.ClientsPage{},
			response: things.MembersPage{
				Page: things.Page{
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
			session: mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: things.Page{
				ListPerms: true,
			},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{"admin"},
			retreiveAllByIDsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{aThings[0]},
			},
			response: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{aThings[0]},
			},
			err: nil,
		},
		{
			desc:    "list members with failed to list objects",
			session: mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: things.Page{
				ListPerms: true,
			},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:    "list members with failed to list permissions",
			session: mgauthn.Session{UserID: validID, DomainID: domainID},
			groupID: testsutil.GenerateUUID(t),
			page: things.Page{
				ListPerms: true,
			},
			retreiveAllByIDsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{aThings[0]},
			},
			response:                things.MembersPage{},
			listObjectsResponse:     policysvc.PolicyPage{},
			listPermissionsResponse: []string{},
			listPermissionsErr:      svcerr.ErrNotFound,
			err:                     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		policyCall := pService.On("ListAllObjects", mock.Anything, mock.Anything).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.retreiveAllByIDsResponse, tc.retreiveAllByIDsErr)
		repoCall1 := pService.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermissionsResponse, tc.listPermissionsErr)
		page, err := svc.ListClientsByGroup(context.Background(), tc.session, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		policyCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDelete(t *testing.T) {
	svc := newService()

	client := things.Client{
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
		repoCall1 := cRepo.On("Delete", context.Background(), tc.clientID).Return(tc.deleteErr)
		err := svc.Delete(context.Background(), mgauthn.Session{}, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
		repoCall1.Unset()
	}
}

func TestShare(t *testing.T) {
	svc := newService()

	clientID := "clientID"

	cases := []struct {
		desc           string
		session        mgauthn.Session
		clientID       string
		relation       string
		userID         string
		addPoliciesErr error
		err            error
	}{
		{
			desc:     "share client successfully",
			session:  mgauthn.Session{UserID: validID, DomainID: validID},
			clientID: clientID,
			err:      nil,
		},
		{
			desc:           "share client with failed to add policies",
			session:        mgauthn.Session{UserID: validID, DomainID: validID},
			clientID:       clientID,
			addPoliciesErr: svcerr.ErrInvalidPolicy,
			err:            svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		policyCall := pService.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
		err := svc.Share(context.Background(), tc.session, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}
}

func TestUnShare(t *testing.T) {
	svc := newService()

	clientID := "clientID"

	cases := []struct {
		desc              string
		session           mgauthn.Session
		clientID          string
		relation          string
		userID            string
		deletePoliciesErr error
		err               error
	}{
		{
			desc:     "unshare client successfully",
			session:  mgauthn.Session{UserID: validID, DomainID: validID},
			clientID: clientID,
			err:      nil,
		},
		{
			desc:              "share client with failed to delete policies",
			session:           mgauthn.Session{UserID: validID, DomainID: validID},
			clientID:          clientID,
			deletePoliciesErr: svcerr.ErrInvalidPolicy,
			err:               svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		policyCall := pService.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		err := svc.Unshare(context.Background(), tc.session, tc.clientID, tc.relation, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}
}

func TestViewClientPerms(t *testing.T) {
	svc := newService()

	validID := valid

	cases := []struct {
		desc             string
		session          mgauthn.Session
		clientID         string
		listPermResponse policysvc.Permissions
		listPermErr      error
		err              error
	}{
		{
			desc:             "view client permissions successfully",
			session:          mgauthn.Session{UserID: validID, DomainID: validID},
			clientID:         validID,
			listPermResponse: policysvc.Permissions{"admin"},
			err:              nil,
		},
		{
			desc:             "view permissions with failed retrieve list permissions response",
			session:          mgauthn.Session{UserID: validID, DomainID: validID},
			clientID:         validID,
			listPermResponse: []string{},
			listPermErr:      svcerr.ErrAuthorization,
			err:              svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		policyCall := pService.On("ListPermissions", mock.Anything, mock.Anything, []string{}).Return(tc.listPermResponse, tc.listPermErr)
		res, err := svc.ViewPerms(context.Background(), tc.session, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.ElementsMatch(t, tc.listPermResponse, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.listPermResponse, res))
		}
		policyCall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	valid := valid

	cases := []struct {
		desc                string
		key                 string
		cacheIDResponse     string
		cacheIDErr          error
		repoIDResponse      things.Client
		retrieveBySecretErr error
		saveErr             error
		err                 error
	}{
		{
			desc:            "identify client with valid key from cache",
			key:             valid,
			cacheIDResponse: thing.ID,
			err:             nil,
		},
		{
			desc:            "identify client with valid key from repo",
			key:             valid,
			cacheIDResponse: "",
			cacheIDErr:      repoerr.ErrNotFound,
			repoIDResponse:  thing,
			err:             nil,
		},
		{
			desc:                "identify client with invalid key",
			key:                 invalid,
			cacheIDResponse:     "",
			cacheIDErr:          repoerr.ErrNotFound,
			repoIDResponse:      things.Client{},
			retrieveBySecretErr: repoerr.ErrNotFound,
			err:                 repoerr.ErrNotFound,
		},
		{
			desc:            "identify client with failed to save to cache",
			key:             valid,
			cacheIDResponse: "",
			cacheIDErr:      repoerr.ErrNotFound,
			repoIDResponse:  thing,
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
	svc := newService()

	cases := []struct {
		desc                string
		request             things.AuthzReq
		cacheIDRes          string
		cacheIDErr          error
		retrieveBySecretRes things.Client
		retrieveBySecretErr error
		cacheSaveErr        error
		checkPolicyErr      error
		id                  string
		err                 error
	}{
		{
			desc:                "authorize client with valid key not in cache",
			request:             things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: things.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			checkPolicyErr:      nil,
			id:                  valid,
			err:                 nil,
		},
		{
			desc:           "authorize thing with valid key in cache",
			request:        things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:     valid,
			checkPolicyErr: nil,
			id:             valid,
		},
		{
			desc:                "authorize thing with invalid key not in cache for non existing thing",
			request:             things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: things.Client{},
			retrieveBySecretErr: repoerr.ErrNotFound,
			err:                 repoerr.ErrNotFound,
		},
		{
			desc:                "authorize thing with valid key not in cache with failed to save to cache",
			request:             things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: things.Client{ID: valid},
			cacheSaveErr:        errors.ErrMalformedEntity,
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc:                "authorize thing with valid key not in cache and failed to authorize",
			request:             things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: things.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			checkPolicyErr:      svcerr.ErrAuthorization,
			err:                 svcerr.ErrAuthorization,
		},
		{
			desc:                "authorize thing with valid key not in cache and not authorize",
			request:             things.AuthzReq{ClientKey: valid, ChannelID: valid, Permission: policies.PublishPermission},
			cacheIDRes:          "",
			cacheIDErr:          repoerr.ErrNotFound,
			retrieveBySecretRes: things.Client{ID: valid},
			retrieveBySecretErr: nil,
			cacheSaveErr:        nil,
			checkPolicyErr:      svcerr.ErrAuthorization,
			err:                 svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		cacheCall := cache.On("ID", context.Background(), tc.request.ClientKey).Return(tc.cacheIDRes, tc.cacheIDErr)
		repoCall := cRepo.On("RetrieveBySecret", context.Background(), tc.request.ClientKey).Return(tc.retrieveBySecretRes, tc.retrieveBySecretErr)
		cacheCall1 := cache.On("Save", context.Background(), tc.request.ClientKey, tc.retrieveBySecretRes.ID).Return(tc.cacheSaveErr)
		policyCall := pEvaluator.On("CheckPolicy", context.Background(), policies.Policy{
			SubjectType: policies.GroupType,
			Subject:     tc.request.ChannelID,
			ObjectType:  policies.ThingType,
			Object:      valid,
			Permission:  tc.request.Permission,
		}).Return(tc.checkPolicyErr)
		id, err := svc.Authorize(context.Background(), tc.request)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, id))
		}
		cacheCall.Unset()
		cacheCall1.Unset()
		repoCall.Unset()
		policyCall.Unset()
	}
}
