// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/clients/events"
	"github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	storeClient  *redis.Client
	storeURL     string
	validSession = authn.Session{
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		UserID:   testsutil.GenerateUUID(&testing.T{}),
	}
	validClient      = generateTestClient(&testing.T{})
	validClientsPage = clients.ClientsPage{
		Page: clients.Page{
			Limit:  10,
			Offset: 0,
			Total:  1,
		},
		Clients: []clients.Client{validClient},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, clients.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestCreateClients(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validID := testsutil.GenerateUUID(t)
	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, validID)

	cases := []struct {
		desc        string
		session     authn.Session
		clients     []clients.Client
		svcRes      []clients.Client
		svcRoleRes  []roles.RoleProvision
		svcErr      error
		resp        []clients.Client
		respRoleRes []roles.RoleProvision
		err         error
	}{
		{
			desc:        "publish successfully",
			session:     validSession,
			clients:     []clients.Client{validClient},
			svcRes:      []clients.Client{validClient},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      nil,
			resp:        []clients.Client{validClient},
			respRoleRes: []roles.RoleProvision{},
			err:         nil,
		},
		{
			desc:        "failed to publish with service error",
			session:     validSession,
			clients:     []clients.Client{validClient},
			svcRes:      []clients.Client{},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      svcerr.ErrCreateEntity,
			resp:        []clients.Client{},
			respRoleRes: []roles.RoleProvision{},
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("CreateClients", validCtx, tc.session, tc.clients).Return(tc.svcRes, tc.svcRoleRes, tc.svcErr)
			resp, respRoleRes, err := nsvc.CreateClients(validCtx, tc.session, tc.clients...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			assert.Equal(t, tc.respRoleRes, respRoleRes, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.respRoleRes, respRoleRes))
			svcCall.Unset()
		})
	}
}

func TestView(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		clientID  string
		withRoles bool
		svcRes    clients.Client
		svcErr    error
		resp      clients.Client
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			clientID:  validClient.ID,
			withRoles: false,
			svcRes:    validClient,
			svcErr:    nil,
			resp:      validClient,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			clientID:  validClient.ID,
			withRoles: false,
			svcRes:    clients.Client{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      clients.Client{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("View", validCtx, tc.session, tc.clientID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.View(validCtx, tc.session, tc.clientID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdate(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedClient := validClient
	updatedClient.Name = "updatedName"

	cases := []struct {
		desc    string
		session authn.Session
		client  clients.Client
		svcRes  clients.Client
		svcErr  error
		resp    clients.Client
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			client:  updatedClient,
			svcRes:  updatedClient,
			svcErr:  nil,
			resp:    updatedClient,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			client:  updatedClient,
			svcRes:  clients.Client{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    clients.Client{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Update", validCtx, tc.session, tc.client).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Update(validCtx, tc.session, tc.client)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateTags(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedClient := validClient
	updatedClient.Tags = []string{"newTag1", "newTag2"}

	cases := []struct {
		desc    string
		session authn.Session
		client  clients.Client
		svcRes  clients.Client
		svcErr  error
		resp    clients.Client
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			client:  updatedClient,
			svcRes:  updatedClient,
			svcErr:  nil,
			resp:    updatedClient,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			client:  updatedClient,
			svcRes:  clients.Client{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    clients.Client{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateTags", validCtx, tc.session, tc.client).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateTags(validCtx, tc.session, tc.client)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedClient := validClient
	updatedClient.Credentials.Secret = "newSecret"

	cases := []struct {
		desc      string
		session   authn.Session
		clientID  string
		newSecret string
		svcRes    clients.Client
		svcErr    error
		resp      clients.Client
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			clientID:  validClient.ID,
			newSecret: "newSecret",
			svcRes:    updatedClient,
			svcErr:    nil,
			resp:      updatedClient,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			clientID:  validClient.ID,
			newSecret: "newSecret",
			svcRes:    clients.Client{},
			svcErr:    svcerr.ErrUpdateEntity,
			resp:      clients.Client{},
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateSecret", validCtx, tc.session, tc.clientID, tc.newSecret).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateSecret(validCtx, tc.session, tc.clientID, tc.newSecret)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnable(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		clientID string
		svcRes   clients.Client
		svcErr   error
		resp     clients.Client
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			clientID: validClient.ID,
			svcRes:   validClient,
			svcErr:   nil,
			resp:     validClient,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			clientID: validClient.ID,
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     clients.Client{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Enable", validCtx, tc.session, tc.clientID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Enable(validCtx, tc.session, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisable(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		clientID string
		svcRes   clients.Client
		svcErr   error
		resp     clients.Client
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			clientID: validClient.ID,
			svcRes:   validClient,
			svcErr:   nil,
			resp:     validClient,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			clientID: validClient.ID,
			svcRes:   clients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     clients.Client{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Disable", validCtx, tc.session, tc.clientID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.Disable(validCtx, tc.session, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListClients(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta clients.Page
		svcRes   clients.ClientsPage
		svcErr   error
		resp     clients.ClientsPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: clients.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validClientsPage,
			svcErr: nil,
			resp:   validClientsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: clients.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: clients.ClientsPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   clients.ClientsPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListClients", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListClients(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListUserClients(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		userID   string
		pageMeta clients.Page
		svcRes   clients.ClientsPage
		svcErr   error
		resp     clients.ClientsPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: clients.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validClientsPage,
			svcErr: nil,
			resp:   validClientsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: clients.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: clients.ClientsPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   clients.ClientsPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListUserClients", validCtx, tc.session, tc.userID, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListUserClients(validCtx, tc.session, tc.userID, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDelete(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		clientID string
		svcErr   error
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			clientID: validClient.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			clientID: validClient.ID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Delete", validCtx, tc.session, tc.clientID).Return(tc.svcErr)
			err := nsvc.Delete(validCtx, tc.session, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestSetParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc          string
		session       authn.Session
		parentGroupID string
		clientID      string
		svcErr        error
		err           error
	}{
		{
			desc:          "publish successfully",
			session:       validSession,
			parentGroupID: testsutil.GenerateUUID(t),
			clientID:      validClient.ID,
			svcErr:        nil,
			err:           nil,
		},
		{
			desc:          "failed to publish with service error",
			session:       validSession,
			parentGroupID: testsutil.GenerateUUID(t),
			clientID:      validClient.ID,
			svcErr:        svcerr.ErrUpdateEntity,
			err:           svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SetParentGroup", validCtx, tc.session, tc.parentGroupID, tc.clientID).Return(tc.svcErr)
			err := nsvc.SetParentGroup(validCtx, tc.session, tc.parentGroupID, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		clientID string
		svcErr   error
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			clientID: validClient.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			clientID: validClient.ID,
			svcErr:   svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveParentGroup", validCtx, tc.session, tc.clientID).Return(tc.svcErr)
			err := nsvc.RemoveParentGroup(validCtx, tc.session, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func generateTestClient(t *testing.T) clients.Client {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return clients.Client{
		ID:     testsutil.GenerateUUID(t),
		Name:   "clientname",
		Domain: testsutil.GenerateUUID(t),
		Tags:   []string{"tag1", "tag2"},
		Credentials: clients.Credentials{
			Identity: "clientidentity",
			Secret:   "clientsecret",
		},
		Metadata:  clients.Metadata{"key1": "value1"},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    clients.EnabledStatus,
	}
}
