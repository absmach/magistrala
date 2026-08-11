// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/pkg/policies"
)

type fakePolicyClient struct {
	authorized          AuthorizedObjectIDs
	queries             []AuthorizedObjectIDsQuery
	directPolicyQueries []DirectPolicyQuery
	capID               string
	capIDs              map[string]string
	blocks              []CreatePermissionBlock
	created             []CreateDirectPolicy
	policies            []DirectPolicy
	deleted             []string
}

func (f *fakePolicyClient) AuthorizedObjectIDs(_ context.Context, q AuthorizedObjectIDsQuery) (AuthorizedObjectIDs, error) {
	f.queries = append(f.queries, q)
	return f.authorized, nil
}

func (f *fakePolicyClient) CheckAuthz(context.Context, AuthzRequest) (AuthzResponse, error) {
	return AuthzResponse{Allowed: true}, nil
}

func (f *fakePolicyClient) CapabilityID(_ context.Context, name string) (string, error) {
	if f.capIDs != nil && f.capIDs[name] != "" {
		return f.capIDs[name], nil
	}
	if f.capID == "" {
		return "cap-publish", nil
	}
	return f.capID, nil
}

func (f *fakePolicyClient) CreatePermissionBlock(_ context.Context, block CreatePermissionBlock) (PermissionBlock, error) {
	f.blocks = append(f.blocks, block)
	return PermissionBlock{
		ID:         "block-1",
		TenantID:   block.TenantID,
		ScopeMode:  block.ScopeMode,
		ObjectKind: block.ObjectKind,
		ObjectType: block.ObjectType,
		ObjectID:   block.ObjectID,
		Effect:     block.Effect,
		Conditions: block.Conditions,
		Actions:    []Capability{{ID: block.ActionIDs[0]}},
	}, nil
}

func (f *fakePolicyClient) CreateDirectPolicy(_ context.Context, policy CreateDirectPolicy) (DirectPolicy, error) {
	f.created = append(f.created, policy)
	return DirectPolicy{ID: "policy-1", PermissionBlockID: policy.PermissionBlockID}, nil
}

func (f *fakePolicyClient) ListDirectPolicies(_ context.Context, q DirectPolicyQuery) (DirectPolicyList, error) {
	f.directPolicyQueries = append(f.directPolicyQueries, q)
	total := uint64(len(f.policies))
	start := q.Offset
	if start > total {
		start = total
	}
	end := total
	if q.Limit > 0 && start+q.Limit < total {
		end = start + q.Limit
	}
	return DirectPolicyList{Items: f.policies[start:end], Total: total}, nil
}

func (f *fakePolicyClient) DeleteDirectPolicy(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	for i, p := range f.policies {
		if p.ID == id {
			f.policies = append(f.policies[:i], f.policies[i+1:]...)
			break
		}
	}
	return nil
}

func TestPolicyServiceListAllObjectsUsesAtomAuthorizedObjectIds(t *testing.T) {
	client := &fakePolicyClient{
		authorized: AuthorizedObjectIDs{IDs: []string{"client-2"}, Total: 1},
	}
	svc := NewPolicyService(client)

	page, err := svc.ListAllObjects(context.Background(), policies.Policy{
		SubjectType: policies.UserType,
		Subject:     testDomainID + "_user-1",
		Domain:      testDomainID,
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	})
	if err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	if len(page.Policies) != 1 || page.Policies[0] != "client-2" {
		t.Fatalf("unexpected policies: %+v", page.Policies)
	}
	if len(client.queries) != 1 {
		t.Fatalf("unexpected authorized object queries: %d", len(client.queries))
	}
	query := client.queries[0]
	if query.SubjectID != "user-1" ||
		query.Action != atomActionRead ||
		query.ObjectKind != atomObjectKindEntity ||
		query.ObjectType != atomObjectTypeEntityDevice ||
		query.TenantID != testDomainID {
		t.Fatalf("unexpected authorized object query: %+v", query)
	}
}

func TestPolicyServiceAddPolicyCreatesInternalCapabilityPolicy(t *testing.T) {
	client := &fakePolicyClient{capID: "cap-publish"}
	svc := NewPolicyService(client)

	err := svc.AddPolicy(context.Background(), policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_client-1",
		SubjectType: policies.ClientType,
		Object:      "channel-1",
		ObjectType:  policies.ChannelType,
		Permission:  policies.PublishPermission,
	})
	if err != nil {
		t.Fatalf("add policy failed: %v", err)
	}
	if len(client.blocks) != 1 || len(client.created) != 1 {
		t.Fatalf("expected one permission block and direct policy, got %d/%d", len(client.blocks), len(client.created))
	}
	block := client.blocks[0]
	if block.TenantID != testDomainID ||
		block.ScopeMode != atomScopeModeObject ||
		block.ObjectKind != atomObjectKindResource ||
		block.ObjectType != "resource:channel" ||
		block.ObjectID != "channel-1" ||
		block.Effect != "allow" ||
		len(block.ActionIDs) != 1 ||
		block.ActionIDs[0] != "cap-publish" {
		t.Fatalf("unexpected permission block: %+v", block)
	}
	created := client.created[0]
	if created.TenantID != testDomainID ||
		created.SubjectKind != atomObjectKindEntity ||
		created.SubjectID != "client-1" ||
		created.PermissionBlockID != "block-1" {
		t.Fatalf("unexpected direct policy: %+v", created)
	}
}

func TestPolicyServiceAddPolicyCreatesGroupCapabilityPolicy(t *testing.T) {
	client := &fakePolicyClient{capID: "cap-read"}
	svc := NewPolicyService(client)

	err := svc.AddPolicy(context.Background(), policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_user-1",
		SubjectType: policies.UserType,
		Object:      "group-1",
		ObjectType:  policies.GroupType,
		Permission:  policies.ViewPermission,
	})
	if err != nil {
		t.Fatalf("add policy failed: %v", err)
	}
	if len(client.blocks) != 1 || len(client.created) != 1 {
		t.Fatalf("expected one permission block and direct policy, got %d/%d", len(client.blocks), len(client.created))
	}
	block := client.blocks[0]
	if block.TenantID != testDomainID ||
		block.ScopeMode != atomScopeModeObject ||
		block.ObjectKind != atomObjectKindGroup ||
		block.ObjectType != "" ||
		block.ObjectID != "group-1" ||
		block.Effect != "allow" ||
		len(block.ActionIDs) != 1 ||
		block.ActionIDs[0] != "cap-read" {
		t.Fatalf("unexpected permission block: %+v", block)
	}
	created := client.created[0]
	if created.TenantID != testDomainID ||
		created.SubjectKind != atomObjectKindEntity ||
		created.SubjectID != "user-1" ||
		created.PermissionBlockID != "block-1" {
		t.Fatalf("unexpected direct policy: %+v", created)
	}
}

func TestPolicyServiceDeletePolicyFilterRemovesMatchingCapabilityPolicy(t *testing.T) {
	client := &fakePolicyClient{
		capID: "cap-subscribe",
		policies: []DirectPolicy{
			{
				ID: "keep",
				PermissionBlock: PermissionBlock{
					ID:         "keep-block",
					ScopeMode:  "object",
					ObjectKind: atomObjectKindResource,
					ObjectType: "resource:channel",
					ObjectID:   "channel-1",
					Actions:    []Capability{{ID: "cap-other"}},
				},
			},
			{
				ID: "delete",
				PermissionBlock: PermissionBlock{
					ID:         "delete-block",
					ScopeMode:  "object",
					ObjectKind: atomObjectKindResource,
					ObjectType: "resource:channel",
					ObjectID:   "channel-1",
					Actions:    []Capability{{ID: "cap-subscribe"}},
				},
			},
		},
	}
	svc := NewPolicyService(client)

	err := svc.DeletePolicyFilter(context.Background(), policies.Policy{
		Domain:      testDomainID,
		Subject:     "domain-1_client-1",
		SubjectType: policies.ClientType,
		Object:      "channel-1",
		ObjectType:  policies.ChannelType,
		Permission:  policies.SubscribePermission,
	})
	if err != nil {
		t.Fatalf("delete policy failed: %v", err)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "delete" {
		t.Fatalf("unexpected deleted policies: %+v", client.deleted)
	}
}

func TestPolicyServiceUnsupportedOperation(t *testing.T) {
	svc := NewPolicyService(&fakePolicyClient{})

	_, err := svc.ListAllObjects(context.Background(), policies.Policy{
		SubjectType: policies.UserType,
		Subject:     "user-1",
		ObjectType:  policies.ChannelType,
		Permission:  policies.ViewPermission,
	})
	if err == nil {
		t.Fatal("expected unsupported operation error")
	}
}

func TestPolicyServiceObjectTypeMatchesOnWriteAndReadPaths(t *testing.T) {
	client := &fakePolicyClient{capID: "cap-read"}
	svc := NewPolicyService(client)

	writePolicy := policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_user-1",
		SubjectType: policies.UserType,
		Object:      "device-1",
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	}
	if err := svc.AddPolicy(context.Background(), writePolicy); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}
	if len(client.blocks) != 1 {
		t.Fatalf("expected one permission block, got %d", len(client.blocks))
	}
	writtenObjectType := client.blocks[0].ObjectType

	readPolicy := policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_user-1",
		SubjectType: policies.UserType,
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	}
	if _, err := svc.ListAllObjects(context.Background(), readPolicy); err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	if len(client.queries) != 1 {
		t.Fatalf("expected one authorized object query, got %d", len(client.queries))
	}
	readObjectType := client.queries[0].ObjectType

	if writtenObjectType != readObjectType {
		t.Fatalf("write and read object types diverged: write=%q read=%q", writtenObjectType, readObjectType)
	}
	if writtenObjectType != "entity:device" {
		t.Fatalf("expected namespaced object type, got %q", writtenObjectType)
	}
}

// namespacedObjectStore simulates Atom storing policy grants keyed by the
// namespaced object type, only returning matches on an exact match. This
// exposes the pre-fix bug where the write and read paths disagreed.
type namespacedObjectStore struct {
	grantedObjectType string
	grantedObjectID   string
}

func (n *namespacedObjectStore) CheckAuthz(context.Context, AuthzRequest) (AuthzResponse, error) {
	return AuthzResponse{}, nil
}

func (n *namespacedObjectStore) AuthorizedObjectIDs(_ context.Context, q AuthorizedObjectIDsQuery) (AuthorizedObjectIDs, error) {
	if q.ObjectType != n.grantedObjectType {
		return AuthorizedObjectIDs{}, nil
	}
	return AuthorizedObjectIDs{IDs: []string{n.grantedObjectID}, Total: 1}, nil
}

func TestPolicyServiceListAllObjectsFindsObjectScopedDeviceGrant(t *testing.T) {
	store := &namespacedObjectStore{grantedObjectType: "entity:device", grantedObjectID: "device-1"}
	svc := NewPolicyService(store)

	page, err := svc.ListAllObjects(context.Background(), policies.Policy{
		SubjectType: policies.UserType,
		Subject:     testDomainID + "_user-1",
		Domain:      testDomainID,
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	})
	if err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	if len(page.Policies) != 1 || page.Policies[0] != "device-1" {
		t.Fatalf("expected object-scoped device grant to be returned, got %+v", page.Policies)
	}
}

func TestPolicyServiceDeletePolicyFilterPaginatesAcrossAllMatches(t *testing.T) {
	const total = 250
	allPolicies := make([]DirectPolicy, 0, total)
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("policy-%d", i)
		allPolicies = append(allPolicies, DirectPolicy{
			ID: id,
			PermissionBlock: PermissionBlock{
				ID:         "block-" + id,
				ScopeMode:  atomScopeModeObject,
				ObjectKind: atomObjectKindResource,
				ObjectType: "resource:channel",
				ObjectID:   "channel-1",
				Actions:    []Capability{{ID: "cap-subscribe"}},
			},
		})
	}
	client := &fakePolicyClient{capID: "cap-subscribe", policies: allPolicies}
	svc := NewPolicyService(client)

	pr := policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_client-1",
		SubjectType: policies.ClientType,
		Object:      "channel-1",
		ObjectType:  policies.ChannelType,
		Permission:  policies.SubscribePermission,
	}
	if err := svc.DeletePolicyFilter(context.Background(), pr); err != nil {
		t.Fatalf("delete policy failed: %v", err)
	}

	if len(client.deleted) != total {
		t.Fatalf("expected %d deletions, got %d", total, len(client.deleted))
	}
	deleted := make(map[string]bool, total)
	for _, id := range client.deleted {
		deleted[id] = true
	}
	for _, p := range allPolicies {
		if !deleted[p.ID] {
			t.Fatalf("policy %s was not deleted", p.ID)
		}
	}
	if len(client.directPolicyQueries) < 3 {
		t.Fatalf("expected pagination across at least 3 pages, got %d queries", len(client.directPolicyQueries))
	}

	remaining, err := client.ListDirectPolicies(context.Background(), DirectPolicyQuery{
		TenantID:    pr.Domain,
		SubjectKind: policyGrantSubjectKind(pr),
		SubjectID:   policySubjectID(pr),
		Limit:       policyPageLimit,
	})
	if err != nil {
		t.Fatalf("re-query after deletion failed: %v", err)
	}
	if remaining.Total != 0 || len(remaining.Items) != 0 {
		t.Fatalf("expected no policies left after deletion, got %+v", remaining)
	}
}

// TestPolicyServiceAddPolicyCreatesTenantScopedPolicy is a regression test
// for acceptance criterion 9: the "tenant" scope mode, used for domain-level
// grants, must be unaffected by adding group-scoped support alongside it.
func TestPolicyServiceAddPolicyCreatesTenantScopedPolicy(t *testing.T) {
	client := &fakePolicyClient{capID: "cap-manage"}
	svc := NewPolicyService(client)

	err := svc.AddPolicy(context.Background(), policies.Policy{
		Domain:      testDomainID,
		Subject:     testDomainID + "_user-1",
		SubjectType: policies.UserType,
		Object:      testDomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.AdminPermission,
	})
	if err != nil {
		t.Fatalf("add policy failed: %v", err)
	}
	if len(client.blocks) != 1 || len(client.created) != 1 {
		t.Fatalf("expected one permission block and direct policy, got %d/%d", len(client.blocks), len(client.created))
	}
	block := client.blocks[0]
	if block.ScopeMode != atomObjectKindTenant ||
		block.ObjectKind != "" ||
		block.ObjectType != "" ||
		block.ObjectID != "" ||
		block.GroupID != "" {
		t.Fatalf("unexpected tenant-scoped permission block: %+v", block)
	}
}

// TestGrantGroupAccessCreatesOneBlockAndOnePolicy is acceptance criterion 1:
// a group grant is a single permission block and a single direct policy.
func TestGrantGroupAccessCreatesOneBlockAndOnePolicy(t *testing.T) {
	client := &fakePolicyClient{capIDs: map[string]string{"read": "cap-read"}}
	svc := NewPolicyService(client)

	grant := GroupGrant{
		TenantID:    testDomainID,
		GroupID:     "group-1",
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "user-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.ClientType,
		Actions:     []string{"read"},
	}
	if err := svc.GrantGroupAccess(context.Background(), grant); err != nil {
		t.Fatalf("grant group access failed: %v", err)
	}
	if len(client.blocks) != 1 || len(client.created) != 1 {
		t.Fatalf("expected one permission block and direct policy, got %d/%d", len(client.blocks), len(client.created))
	}

	block := client.blocks[0]
	if block.TenantID != testDomainID ||
		block.ScopeMode != atomScopeModeGroupDirectObjects ||
		block.ObjectKind != atomObjectKindEntity ||
		block.ObjectType != "entity:device" ||
		block.ObjectID != "" ||
		block.GroupID != "group-1" ||
		block.Effect != atomDecisionAllow ||
		len(block.ActionIDs) != 1 ||
		block.ActionIDs[0] != "cap-read" {
		t.Fatalf("unexpected permission block: %+v", block)
	}

	created := client.created[0]
	if created.TenantID != testDomainID ||
		created.SubjectKind != atomObjectKindEntity ||
		created.SubjectID != "user-1" ||
		created.PermissionBlockID != "block-1" {
		t.Fatalf("unexpected direct policy: %+v", created)
	}
}

// TestGrantGroupAccessIncludeDescendantsUsesDescendantScopeMode is the write
// side of acceptance criterion 6.
func TestGrantGroupAccessIncludeDescendantsUsesDescendantScopeMode(t *testing.T) {
	client := &fakePolicyClient{capIDs: map[string]string{"read": "cap-read"}}
	svc := NewPolicyService(client)

	grant := GroupGrant{
		TenantID:           testDomainID,
		GroupID:            "group-1",
		SubjectKind:        atomObjectKindEntity,
		SubjectID:          "user-1",
		ObjectKind:         atomObjectKindEntity,
		ObjectType:         policies.ClientType,
		Actions:            []string{"read"},
		IncludeDescendants: true,
	}
	if err := svc.GrantGroupAccess(context.Background(), grant); err != nil {
		t.Fatalf("grant group access failed: %v", err)
	}
	if len(client.blocks) != 1 {
		t.Fatalf("expected one permission block, got %d", len(client.blocks))
	}
	if client.blocks[0].ScopeMode != atomScopeModeGroupDescendantObjects {
		t.Fatalf("expected descendant scope mode, got %q", client.blocks[0].ScopeMode)
	}
}

// TestGrantGroupAccessResolvesMultipleActions covers a grant naming more
// than one action: still one block, carrying every resolved capability ID.
func TestGrantGroupAccessResolvesMultipleActions(t *testing.T) {
	client := &fakePolicyClient{capIDs: map[string]string{"read": "cap-read", "write": "cap-write"}}
	svc := NewPolicyService(client)

	grant := GroupGrant{
		TenantID:    testDomainID,
		GroupID:     "group-1",
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "user-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.ClientType,
		Actions:     []string{"read", "write"},
	}
	if err := svc.GrantGroupAccess(context.Background(), grant); err != nil {
		t.Fatalf("grant group access failed: %v", err)
	}
	if len(client.blocks) != 1 || len(client.created) != 1 {
		t.Fatalf("expected one permission block and direct policy, got %d/%d", len(client.blocks), len(client.created))
	}
	ids := client.blocks[0].ActionIDs
	if len(ids) != 2 || ids[0] != "cap-read" || ids[1] != "cap-write" {
		t.Fatalf("unexpected action ids: %+v", ids)
	}
}

func TestGrantGroupAccessRejectsInvalidGrant(t *testing.T) {
	cases := []struct {
		name  string
		grant GroupGrant
	}{
		{
			name:  "missing tenant",
			grant: GroupGrant{GroupID: "group-1", SubjectKind: atomObjectKindEntity, SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "missing group",
			grant: GroupGrant{TenantID: testDomainID, SubjectKind: atomObjectKindEntity, SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "missing subject",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectKind: atomObjectKindEntity, ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "missing subject kind",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "unsupported subject kind",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectKind: atomObjectKindResource, SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "no actions",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectKind: atomObjectKindEntity, SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: policies.ClientType},
		},
		{
			name:  "object kind is a group, not entity/resource",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectKind: atomObjectKindEntity, SubjectID: "user-1", ObjectKind: atomObjectKindGroup, ObjectType: policies.ClientType, Actions: []string{"read"}},
		},
		{
			name:  "unmappable object type",
			grant: GroupGrant{TenantID: testDomainID, GroupID: "group-1", SubjectKind: atomObjectKindEntity, SubjectID: "user-1", ObjectKind: atomObjectKindEntity, ObjectType: "unknown", Actions: []string{"read"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakePolicyClient{capIDs: map[string]string{"read": "cap-read"}}
			svc := NewPolicyService(client)
			if err := svc.GrantGroupAccess(context.Background(), tc.grant); err == nil {
				t.Fatal("expected invalid group grant to be rejected")
			}
			if len(client.blocks) != 0 {
				t.Fatalf("expected no permission block to be written, got %+v", client.blocks)
			}
		})
	}
}

// TestDirectPolicyMatchesComparesGroupID is a dedicated regression test for
// the highest-risk change in this file: two permission blocks can be
// identical in every other respect (tenant, object kind/type, scope mode)
// and differ only by which group they grant access through. Without
// comparing GroupID, directPolicyMatches cannot tell them apart.
func TestDirectPolicyMatchesComparesGroupID(t *testing.T) {
	policy := DirectPolicy{
		PermissionBlock: PermissionBlock{
			ID:         "block-a",
			ScopeMode:  atomScopeModeGroupDirectObjects,
			ObjectKind: atomObjectKindEntity,
			ObjectType: "entity:device",
			GroupID:    "group-a",
		},
	}

	matchSameGroup := blockMatch{
		ScopeMode:  atomScopeModeGroupDirectObjects,
		ObjectKind: atomObjectKindEntity,
		ObjectType: "entity:device",
		GroupID:    "group-a",
	}
	if !directPolicyMatches(policy, matchSameGroup) {
		t.Fatal("expected match against the same group")
	}

	matchDifferentGroup := matchSameGroup
	matchDifferentGroup.GroupID = "group-b"
	if directPolicyMatches(policy, matchDifferentGroup) {
		t.Fatal("block for group-a must not match a query for group-b")
	}
}

// TestRevokeGroupAccessOnlyRemovesTargetedGroupsBlock exercises the same
// risk at the PolicyService level: two grants share every field except
// GroupID, and revoking one must not touch the other.
func TestRevokeGroupAccessOnlyRemovesTargetedGroupsBlock(t *testing.T) {
	client := &fakePolicyClient{
		capIDs: map[string]string{"read": "cap-read"},
		policies: []DirectPolicy{
			{
				ID:          "policy-a",
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-1",
				PermissionBlock: PermissionBlock{
					ID:         "block-a",
					ScopeMode:  atomScopeModeGroupDirectObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-a",
					Actions:    []Capability{{ID: "cap-read", Name: "read"}},
				},
			},
			{
				ID:          "policy-b",
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-1",
				PermissionBlock: PermissionBlock{
					ID:         "block-b",
					ScopeMode:  atomScopeModeGroupDirectObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-b",
					Actions:    []Capability{{ID: "cap-read", Name: "read"}},
				},
			},
		},
	}
	svc := NewPolicyService(client)

	err := svc.RevokeGroupAccess(context.Background(), GroupGrant{
		TenantID:    testDomainID,
		GroupID:     "group-a",
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "user-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.ClientType,
		Actions:     []string{"read"},
	})
	if err != nil {
		t.Fatalf("revoke group access failed: %v", err)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "policy-a" {
		t.Fatalf("expected only policy-a to be deleted, got %+v", client.deleted)
	}
}

func TestRevokeGroupAccessOnlyRemovesMatchingActionSet(t *testing.T) {
	client := &fakePolicyClient{
		capIDs: map[string]string{"read": "cap-read", "write": "cap-write"},
		policies: []DirectPolicy{
			{
				ID:          "policy-read",
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-1",
				PermissionBlock: PermissionBlock{
					ID:         "block-read",
					ScopeMode:  atomScopeModeGroupDirectObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-a",
					Actions:    []Capability{{ID: "cap-read", Name: "read"}},
				},
			},
			{
				ID:          "policy-write",
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-1",
				PermissionBlock: PermissionBlock{
					ID:         "block-write",
					ScopeMode:  atomScopeModeGroupDirectObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-a",
					Actions:    []Capability{{ID: "cap-write", Name: "write"}},
				},
			},
		},
	}
	svc := NewPolicyService(client)

	err := svc.RevokeGroupAccess(context.Background(), GroupGrant{
		TenantID:    testDomainID,
		GroupID:     "group-a",
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "user-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.ClientType,
		Actions:     []string{"read"},
	})
	if err != nil {
		t.Fatalf("revoke group access failed: %v", err)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "policy-read" {
		t.Fatalf("expected only policy-read to be deleted, got %+v", client.deleted)
	}
}

// TestListGroupGrantsFiltersByGroupID covers the read side of the group
// grant API: only grants recorded against the requested group come back.
func TestListGroupGrantsFiltersByGroupID(t *testing.T) {
	client := &fakePolicyClient{
		capIDs: map[string]string{"read": "cap-read"},
		policies: []DirectPolicy{
			{
				ID:          "policy-a",
				TenantID:    testDomainID,
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-1",
				PermissionBlock: PermissionBlock{
					ID:         "block-a",
					ScopeMode:  atomScopeModeGroupDirectObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-a",
					Actions:    []Capability{{ID: "cap-read", Name: "read"}},
				},
			},
			{
				ID:          "policy-b",
				TenantID:    testDomainID,
				SubjectKind: atomObjectKindEntity,
				SubjectID:   "user-2",
				PermissionBlock: PermissionBlock{
					ID:         "block-b",
					ScopeMode:  atomScopeModeGroupDescendantObjects,
					ObjectKind: atomObjectKindEntity,
					ObjectType: "entity:device",
					GroupID:    "group-b",
					Actions:    []Capability{{ID: "cap-read", Name: "read"}},
				},
			},
		},
	}
	svc := NewPolicyService(client)

	grants, err := svc.ListGroupGrants(context.Background(), "group-a")
	if err != nil {
		t.Fatalf("list group grants failed: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("expected one grant for group-a, got %+v", grants)
	}
	got := grants[0]
	if got.GroupID != "group-a" || got.SubjectID != "user-1" || got.IncludeDescendants {
		t.Fatalf("unexpected grant: %+v", got)
	}
	if got.ObjectType != policies.ClientType {
		t.Fatalf("expected revocable object type %q, got %q", policies.ClientType, got.ObjectType)
	}
	if len(got.Actions) != 1 || got.Actions[0] != "read" {
		t.Fatalf("unexpected grant actions: %+v", got.Actions)
	}
	if err := svc.RevokeGroupAccess(context.Background(), got); err != nil {
		t.Fatalf("revoke listed grant failed: %v", err)
	}
	if len(client.deleted) != 1 || client.deleted[0] != "policy-a" {
		t.Fatalf("expected listed grant to revoke policy-a, got %+v", client.deleted)
	}
}

func TestIsSupportedObjectList(t *testing.T) {
	cases := []struct {
		name string
		pr   policies.Policy
		want bool
	}{
		{
			name: "user view on client is supported",
			pr:   policies.Policy{SubjectType: policies.UserType, Subject: "user-1", ObjectType: policies.ClientType, Permission: policies.ViewPermission},
			want: true,
		},
		{
			name: "user read on client (entity:device) is supported",
			pr:   policies.Policy{SubjectType: policies.UserType, Subject: "user-1", ObjectType: policies.ClientType, Permission: atomActionRead},
			want: true,
		},
		{
			name: "non-user subject is unsupported",
			pr:   policies.Policy{SubjectType: policies.ClientType, Subject: "client-1", ObjectType: policies.ClientType, Permission: policies.ViewPermission},
			want: false,
		},
		{
			name: "empty subject is unsupported",
			pr:   policies.Policy{SubjectType: policies.UserType, Subject: "", ObjectType: policies.ClientType, Permission: policies.ViewPermission},
			want: false,
		},
		{
			name: "non-client object type is unsupported",
			pr:   policies.Policy{SubjectType: policies.UserType, Subject: "user-1", ObjectType: policies.ChannelType, Permission: policies.ViewPermission},
			want: false,
		},
		{
			name: "unsupported permission on client is rejected",
			pr:   policies.Policy{SubjectType: policies.UserType, Subject: "user-1", ObjectType: policies.ClientType, Permission: policies.EditPermission},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSupportedObjectList(tc.pr); got != tc.want {
				t.Fatalf("isSupportedObjectList(%+v) = %v, want %v", tc.pr, got, tc.want)
			}
		})
	}
}
