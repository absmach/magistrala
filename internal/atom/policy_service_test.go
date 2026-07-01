// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
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
	return DirectPolicyList{Items: f.policies, Total: uint64(len(f.policies))}, nil
}

func (f *fakePolicyClient) DeleteDirectPolicy(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
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
		query.ObjectType != entityKind(KindClient) ||
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
