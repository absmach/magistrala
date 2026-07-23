// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type atomAuthorizer struct {
	req  atom.AuthzRequest
	reqs []atom.AuthzRequest
}

func (a *atomAuthorizer) CheckAuthz(_ context.Context, req atom.AuthzRequest) (atom.AuthzResponse, error) {
	a.req = req
	a.reqs = append(a.reqs, req)
	return atom.AuthzResponse{Allowed: true}, nil
}

func TestAtomAuthorizationGenerateSecureCredentialRequiresConfigReadAndTenantManage(t *testing.T) {
	svc := mocks.NewService(t)
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	want := bootstrap.SecureBootstrapCredential{ExternalID: "device-1"}
	svc.On("GenerateSecureCredential", mock.Anything, session, "config-1").Return(want, nil).Once()
	authorizer := &atomAuthorizer{}

	got, err := AtomAuthorizationMiddleware(svc, authorizer).GenerateSecureCredential(
		context.Background(), session, "config-1",
	)

	require.NoError(t, err)
	assert.Equal(t, want, got)
	require.Len(t, authorizer.reqs, 2)
	assert.Equal(t, "read", authorizer.reqs[0].Action)
	assert.Equal(t, "resource", authorizer.reqs[0].ObjectKind)
	assert.Equal(t, atom.KindBootstrapConfig, authorizer.reqs[0].Context["legacy_object_type"])
	assert.Equal(t, "config-1", authorizer.reqs[0].ObjectID)
	assert.Equal(t, "manage", authorizer.reqs[1].Action)
	assert.Equal(t, "tenant", authorizer.reqs[1].ObjectKind)
	assert.Equal(t, session.DomainID, authorizer.reqs[1].ObjectID)
}

func TestAtomAuthorizationListUsesExistingTenantWriteGrant(t *testing.T) {
	svc := mocks.NewService(t)
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	filter := bootstrap.Filter{}
	want := bootstrap.ConfigsPage{}
	svc.On("List", mock.Anything, session, filter, uint64(0), uint64(10)).Return(want, nil).Once()
	authorizer := &atomAuthorizer{}

	page, err := AtomAuthorizationMiddleware(svc, authorizer).List(context.Background(), session, filter, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, want, page)
	assert.Equal(t, "write", authorizer.req.Action)
	assert.Equal(t, "tenant", authorizer.req.ObjectKind)
	assert.Equal(t, session.DomainID, authorizer.req.ObjectID)
}

func TestAtomAuthorizationListProfilesUsesExistingTenantWriteGrant(t *testing.T) {
	svc := mocks.NewService(t)
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	want := bootstrap.ProfilesPage{}
	svc.On("ListProfiles", mock.Anything, session, uint64(0), uint64(10), "").Return(want, nil).Once()
	authorizer := &atomAuthorizer{}

	page, err := AtomAuthorizationMiddleware(svc, authorizer).ListProfiles(context.Background(), session, 0, 10, "")

	require.NoError(t, err)
	assert.Equal(t, want, page)
	assert.Equal(t, "write", authorizer.req.Action)
	assert.Equal(t, "tenant", authorizer.req.ObjectKind)
	assert.Equal(t, session.DomainID, authorizer.req.ObjectID)
}

func TestAtomAuthorizationBindResourcesUsesTenantWriteGrant(t *testing.T) {
	svc := mocks.NewService(t)
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	bindings := []bootstrap.BindingRequest{{
		Slot: "telemetry", Type: "channel", ResourceID: "channel-1",
	}}
	svc.On("BindResources", mock.Anything, session, "token", "config-1", bindings).Return(nil).Once()
	authorizer := &atomAuthorizer{}

	err := AtomAuthorizationMiddleware(svc, authorizer).BindResources(
		context.Background(), session, "token", "config-1", bindings,
	)

	require.NoError(t, err)
	assert.Equal(t, "write", authorizer.req.Action)
	assert.Equal(t, "tenant", authorizer.req.ObjectKind)
	assert.Equal(t, session.DomainID, authorizer.req.ObjectID)
}
