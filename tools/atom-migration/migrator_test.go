// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestGroupRoleBlockPlansUseAtomSupportedScopes(t *testing.T) {
	scope := roleScope{prefix: "groups", scopeMode: "group"}
	roleID := "role-1"
	groupID := "11111111-1111-1111-1111-111111111111"
	tenantID := "22222222-2222-2222-2222-222222222222"

	cases := []struct {
		name       string
		action     string
		scopeMode  string
		objectKind any
		objectType any
		objectID   any
		groupID    any
	}{
		{
			name: "direct device action", action: "client_read",
			scopeMode: "group_direct_objects", objectKind: "entity", objectType: "entity:device", groupID: groupID,
		},
		{
			name: "descendant device action", action: "subgroup_client_set_parent_group",
			scopeMode: "group_descendant_objects", objectKind: "entity", objectType: "entity:device", groupID: groupID,
		},
		{
			name: "direct channel action", action: "channel_publish",
			scopeMode: "group_direct_objects", objectKind: "resource", objectType: "resource:channel", groupID: groupID,
		},
		{
			name: "descendant channel action", action: "subgroup_channel_subscribe",
			scopeMode: "group_descendant_objects", objectKind: "resource", objectType: "resource:channel", groupID: groupID,
		},
		{
			name: "descendant group action", action: "subgroup_set_child",
			scopeMode: "group_descendant_groups", groupID: groupID,
		},
		{
			name: "self group action", action: "manage_role",
			scopeMode: "object", objectKind: "group", objectID: groupID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plans := scope.blockPlans(roleID, groupID, tenantID, tc.action)
			if len(plans) != 1 {
				t.Fatalf("expected one plan, got %d", len(plans))
			}
			got := plans[0]
			if got.ScopeMode != tc.scopeMode {
				t.Fatalf("scope mode = %q, want %q", got.ScopeMode, tc.scopeMode)
			}
			if got.TenantID != tenantID {
				t.Fatalf("tenant = %q, want %q", got.TenantID, tenantID)
			}
			if got.ObjectKind != tc.objectKind || got.ObjectType != tc.objectType || got.ObjectID != tc.objectID || got.GroupID != tc.groupID {
				t.Fatalf("plan = %+v, want objectKind=%v objectType=%v objectID=%v groupID=%v", got, tc.objectKind, tc.objectType, tc.objectID, tc.groupID)
			}
		})
	}
}

func TestMapActionPreservesChannelPublishSubscribeVariants(t *testing.T) {
	cases := map[string]string{
		"channel_publish":            actionPublish,
		"subgroup_channel_publish":   actionPublish,
		"channel_subscribe":          actionSubscribe,
		"subgroup_channel_subscribe": actionSubscribe,
	}
	for raw, want := range cases {
		got, ok := mapAction(raw)
		if !ok {
			t.Fatalf("mapAction(%q) returned not ok", raw)
		}
		if got != want {
			t.Fatalf("mapAction(%q) = %q, want %q", raw, got, want)
		}
	}
}
