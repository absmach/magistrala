// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/mocks"
	"github.com/absmach/magistrala/alarms/operations"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingAtomAuthorizer struct {
	allowed    bool
	allow      func(atom.AuthzRequest) bool
	authorized atom.AuthorizedObjectIDs
	reqs       []atom.AuthzRequest
	queries    []atom.AuthorizedObjectIDsQuery
}

func (a *recordingAtomAuthorizer) CheckAuthz(_ context.Context, req atom.AuthzRequest) (atom.AuthzResponse, error) {
	a.reqs = append(a.reqs, req)
	if a.allow != nil {
		return atom.AuthzResponse{Allowed: a.allow(req)}, nil
	}
	return atom.AuthzResponse{Allowed: a.allowed}, nil
}

func (a *recordingAtomAuthorizer) AuthorizedObjectIDs(_ context.Context, q atom.AuthorizedObjectIDsQuery) (atom.AuthorizedObjectIDs, error) {
	a.queries = append(a.queries, q)
	return a.authorized, nil
}

func TestListAlarmsAuthorizesTenantAlarmReader(t *testing.T) {
	svc := mocks.NewService(t)
	pm := alarms.PageMetadata{Limit: 10}
	expectedPM := pm
	expectedPM.DomainID = "domain-1"
	session := authn.Session{UserID: "user-1", DomainID: "domain-1", DomainUserID: "domain-1_user-1"}
	authz := &recordingAtomAuthorizer{allowed: true}
	wrapped, err := NewAtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListAlarms", mock.Anything, session, expectedPM).Return(alarms.AlarmsPage{Limit: 10}, nil).Once()
	page, err := wrapped.ListAlarms(context.Background(), session, pm)

	require.NoError(t, err)
	assert.Equal(t, uint64(10), page.Limit)
	require.Len(t, authz.reqs, 1)
	assert.Equal(t, atom.AuthzRequest{
		SubjectID:  "user-1",
		Action:     "alarm_read",
		ResourceID: "",
		ObjectKind: "tenant",
		ObjectID:   "domain-1",
		Context: map[string]any{
			"domain_id":          "domain-1",
			"legacy_object_type": "domain",
		},
	}, authz.reqs[0])
}

func TestListAlarmsFiltersToReadableRulesWhenTenantAlarmReadDenied(t *testing.T) {
	svc := mocks.NewService(t)
	pm := alarms.PageMetadata{Limit: 10}
	expectedPM := pm
	expectedPM.DomainID = "domain-1"
	expectedPM.RuleIDs = []string{"rule-1", "rule-2"}
	authz := &recordingAtomAuthorizer{
		allowed:    false,
		authorized: atom.AuthorizedObjectIDs{IDs: []string{"rule-1", "rule-2"}, Total: 2},
	}
	wrapped, err := NewAtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListAlarms", mock.Anything, authn.Session{UserID: "user-1", DomainID: "domain-1"}, expectedPM).Return(alarms.AlarmsPage{Limit: 10}, nil).Once()
	_, err = wrapped.ListAlarms(context.Background(), authn.Session{UserID: "user-1", DomainID: "domain-1"}, pm)

	require.NoError(t, err)
	require.Len(t, authz.reqs, 1)
	assert.Equal(t, "alarm_read", authz.reqs[0].Action)
	assert.Equal(t, "tenant", authz.reqs[0].ObjectKind)
	require.Len(t, authz.queries, 1)
	assert.Equal(t, atom.AuthorizedObjectIDsQuery{
		SubjectID:  "user-1",
		Action:     "alarm_read",
		ObjectKind: "resource",
		ObjectType: "resource:rule",
		TenantID:   "domain-1",
		Limit:      100,
	}, authz.queries[0])
}

func TestListAlarmsWithRuleFilterAuthorizesRuleRead(t *testing.T) {
	svc := mocks.NewService(t)
	pm := alarms.PageMetadata{Limit: 10, RuleID: "rule-1"}
	expectedPM := pm
	expectedPM.DomainID = "domain-1"
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	authz := &recordingAtomAuthorizer{
		allow: func(req atom.AuthzRequest) bool {
			return req.Action == "alarm_read" && req.ObjectKind == "resource" && req.ObjectID == "rule-1"
		},
	}
	wrapped, err := NewAtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListAlarms", mock.Anything, session, expectedPM).Return(alarms.AlarmsPage{Limit: 10}, nil).Once()
	_, err = wrapped.ListAlarms(context.Background(), session, pm)

	require.NoError(t, err)
	require.Len(t, authz.reqs, 2)
	assert.Equal(t, "alarm_read", authz.reqs[0].Action)
	assert.Equal(t, "alarm_read", authz.reqs[1].Action)
	assert.Equal(t, "resource", authz.reqs[1].ObjectKind)
	assert.Equal(t, "rules", authz.reqs[1].Context["legacy_object_type"])
}

func TestListAlarmsSuperAdminSkipsListAuthorization(t *testing.T) {
	svc := mocks.NewService(t)
	pm := alarms.PageMetadata{Limit: 10}
	expectedPM := pm
	expectedPM.DomainID = "domain-1"
	session := authn.Session{UserID: "admin-1", DomainID: "domain-1", Role: authn.SuperAdminRole}
	authz := &recordingAtomAuthorizer{allowed: true}
	wrapped, err := NewAtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ListAlarms", mock.Anything, mock.MatchedBy(func(s authn.Session) bool {
		return s.SuperAdmin
	}), expectedPM).Return(alarms.AlarmsPage{Limit: 10}, nil).Once()
	_, err = wrapped.ListAlarms(context.Background(), session, pm)

	require.NoError(t, err)
	require.Len(t, authz.reqs, 1)
	assert.Equal(t, "manage", authz.reqs[0].Action)
}

func TestAcknowledgeAlarmAuthorizesRuleAlarmActionWhenTenantDenied(t *testing.T) {
	svc := mocks.NewService(t)
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}
	current := alarms.Alarm{ID: "alarm-1", RuleID: "rule-1", DomainID: "domain-1"}
	update := alarms.Alarm{ID: "alarm-1", AcknowledgedBy: "user-1"}
	authz := &recordingAtomAuthorizer{
		allow: func(req atom.AuthzRequest) bool {
			return req.Action == "alarm_acknowledge" && req.ObjectKind == "resource" && req.ObjectID == "rule-1"
		},
	}
	wrapped, err := NewAtomAuthorizationMiddleware(svc, authz, testEntitiesOps(t))
	require.NoError(t, err)

	svc.On("ViewAlarm", mock.Anything, session, "alarm-1").Return(current, nil).Once()
	svc.On("UpdateAlarm", mock.Anything, session, update).Return(update, nil).Once()
	_, err = wrapped.UpdateAlarm(context.Background(), session, update)

	require.NoError(t, err)
	require.Len(t, authz.reqs, 2)
	assert.Equal(t, atom.AuthzRequest{
		SubjectID:  "user-1",
		Action:     "alarm_acknowledge",
		ResourceID: "",
		ObjectKind: "tenant",
		ObjectID:   "domain-1",
		Context: map[string]any{
			"domain_id":          "domain-1",
			"legacy_object_type": "domain",
		},
	}, authz.reqs[0])
	assert.Equal(t, atom.AuthzRequest{
		SubjectID:  "user-1",
		Action:     "alarm_acknowledge",
		ResourceID: "rule-1",
		ObjectKind: "resource",
		ObjectID:   "rule-1",
		Context: map[string]any{
			"domain_id":          "domain-1",
			"legacy_object_type": "rules",
		},
	}, authz.reqs[1])
}

func testEntitiesOps(t *testing.T) permissions.EntitiesOperations[permissions.Operation] {
	t.Helper()
	details := operations.OperationDetails()
	perms := make(map[string]permissions.Permission, len(details))
	for op, detail := range details {
		if detail.PermissionRequired {
			perms[detail.Name] = testPermission(op, detail.Name)
		}
	}
	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{operations.EntityType: perms},
		permissions.EntitiesOperationDetails[permissions.Operation]{operations.EntityType: details},
	)
	require.NoError(t, err)
	return entitiesOps
}

func testPermission(op permissions.Operation, fallback string) permissions.Permission {
	switch op {
	case operations.OpViewAlarm, operations.OpListAlarms:
		return "alarm_read_permission"
	case operations.OpUpdateAlarm:
		return "alarm_update_permission"
	case operations.OpDeleteAlarm:
		return "alarm_delete_permission"
	case operations.OpAssignAlarm:
		return "alarm_assign_permission"
	case operations.OpAcknowledgeAlarm:
		return "alarm_acknowledge_permission"
	case operations.OpResolveAlarm:
		return "alarm_resolve_permission"
	default:
		return permissions.Permission(fallback)
	}
}
