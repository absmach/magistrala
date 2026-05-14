// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/pkg/policies"
)

type fakePolicyClient struct {
	entities []Entity
	allowed  map[string]bool
	checks   []AuthzRequest
}

func (f *fakePolicyClient) ListEntities(context.Context, Query) (EntityList, error) {
	return EntityList{
		Items: f.entities,
		Total: uint64(len(f.entities)),
	}, nil
}

func (f *fakePolicyClient) CheckAuthz(_ context.Context, req AuthzRequest) (AuthzResponse, error) {
	f.checks = append(f.checks, req)
	return AuthzResponse{Allowed: f.allowed[req.ObjectID]}, nil
}

func TestPolicyServiceListAllObjectsFiltersByAtomAuthz(t *testing.T) {
	client := &fakePolicyClient{
		entities: []Entity{
			{ID: "client-1", Kind: entityKind(KindClient)},
			{ID: "client-2", Kind: entityKind(KindClient)},
		},
		allowed: map[string]bool{"client-2": true},
	}
	svc := NewPolicyService(client)

	page, err := svc.ListAllObjects(context.Background(), policies.Policy{
		SubjectType: policies.UserType,
		Subject:     "domain-1_user-1",
		Domain:      "domain-1",
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	})
	if err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	if len(page.Policies) != 1 || page.Policies[0] != "client-2" {
		t.Fatalf("unexpected policies: %+v", page.Policies)
	}
	if len(client.checks) != 2 {
		t.Fatalf("unexpected authz checks: %d", len(client.checks))
	}
	if client.checks[0].SubjectID != "user-1" || client.checks[0].Action != policies.ViewPermission || client.checks[0].ObjectKind != "entity" {
		t.Fatalf("unexpected authz request: %+v", client.checks[0])
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
