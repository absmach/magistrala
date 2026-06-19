// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	pkgerrors "github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/mocks"
	"github.com/absmach/magistrala/re/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingAtomAuthorizer struct {
	allowed bool
	reqs    []atom.AuthzRequest
}

func (a *recordingAtomAuthorizer) CheckAuthz(_ context.Context, req atom.AuthzRequest) (atom.AuthzResponse, error) {
	a.reqs = append(a.reqs, req)
	return atom.AuthzResponse{Allowed: a.allowed}, nil
}

func TestListRulesAuthorizesRegularUser(t *testing.T) {
	svc := mocks.NewService(t)
	pm := re.PageMeta{Limit: 10}
	session := authn.Session{UserID: "user-1", DomainID: "domain-1", DomainUserID: "domain-1_user-1"}
	authz := &recordingAtomAuthorizer{allowed: true}
	wrapped, err := AtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListRules", mock.Anything, session, pm).Return(re.Page{Limit: 10}, nil).Once()
	page, err := wrapped.ListRules(context.Background(), session, pm)

	require.NoError(t, err)
	assert.Equal(t, uint64(10), page.Limit)
	require.Len(t, authz.reqs, 1)
	assert.Equal(t, atom.AuthzRequest{
		SubjectID:  "user-1",
		Action:     "list",
		ResourceID: auth.AnyIDs,
		ObjectKind: "resource",
		ObjectID:   auth.AnyIDs,
		Context: map[string]any{
			"domain_id":          "domain-1",
			"legacy_object_type": operations.EntityType,
		},
	}, authz.reqs[0])
}

func TestListRulesDeniedRegularUserDoesNotDelegate(t *testing.T) {
	svc := mocks.NewService(t)
	authz := &recordingAtomAuthorizer{allowed: false}
	wrapped, err := AtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	_, err = wrapped.ListRules(context.Background(), authn.Session{UserID: "user-1", DomainID: "domain-1"}, re.PageMeta{})

	assert.True(t, pkgerrors.Contains(err, pkgerrors.ErrAuthorization))
	require.Len(t, authz.reqs, 1)
}

func TestListRulesSuperAdminSkipsListAuthorization(t *testing.T) {
	svc := mocks.NewService(t)
	pm := re.PageMeta{Limit: 10}
	session := authn.Session{UserID: "admin-1", DomainID: "domain-1", Role: authn.SuperAdminRole}
	authz := &recordingAtomAuthorizer{allowed: true}
	wrapped, err := AtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListRules", mock.Anything, mock.MatchedBy(func(s authn.Session) bool {
		return s.SuperAdmin
	}), pm).Return(re.Page{Limit: 10}, nil).Once()
	_, err = wrapped.ListRules(context.Background(), session, pm)

	require.NoError(t, err)
	require.Len(t, authz.reqs, 1)
	assert.Equal(t, "manage", authz.reqs[0].Action)
}

func testEntitiesOps(t *testing.T) permissions.EntitiesOperations[permissions.Operation] {
	t.Helper()
	details := operations.OperationDetails()
	perms := make(map[string]permissions.Permission, len(details))
	for _, detail := range details {
		if detail.PermissionRequired {
			perms[detail.Name] = permissions.Permission(detail.Name)
		}
	}
	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{operations.EntityType: perms},
		permissions.EntitiesOperationDetails[permissions.Operation]{operations.EntityType: details},
	)
	require.NoError(t, err)
	return entitiesOps
}
