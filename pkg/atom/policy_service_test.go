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
