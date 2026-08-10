// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

type gqlPayload struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func decodePayload(t *testing.T, r *http.Request) gqlPayload {
	t.Helper()
	var payload gqlPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return payload
}

func groupJSON(id, name, tenantID, parentID string) map[string]any {
	return map[string]any{
		"id":         id,
		"name":       name,
		"tenant_id":  tenantID,
		"parent_id":  parentID,
		"status":     "active",
		"created_at": "2026-08-10T00:00:00Z",
	}
}

func entityJSON(id, tenantID string) map[string]any {
	return map[string]any{
		"id":         id,
		"kind":       "device",
		"name":       id,
		"tenant_id":  tenantID,
		"status":     "active",
		"created_at": "2026-08-10T00:00:00Z",
	}
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestObjectGroupMembersAddAndRemove covers creating an object group, adding
// three members, listing them, then removing one and confirming the other
// two remain.
func TestObjectGroupMembersAddAndRemove(t *testing.T) {
	const groupID = "group-1"
	members := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON(groupID, "devices", testTenantID, "")},
			})
		case strings.Contains(payload.Query, "addGroupMember"):
			if payload.Variables["groupId"] != groupID {
				t.Fatalf("unexpected group id: %+v", payload.Variables)
			}
			members[payload.Variables["entityId"].(string)] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"addGroupMember": true}})
		case strings.Contains(payload.Query, "removeGroupMember"):
			delete(members, payload.Variables["entityId"].(string))
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"removeGroupMember": true}})
		case strings.Contains(payload.Query, "groupMembers"):
			items := []map[string]any{}
			for id := range members {
				items = append(items, entityJSON(id, testTenantID))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMembers": items}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	group, err := client.CreateObjectGroup(ctx, Group{Name: "devices", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create object group failed: %v", err)
	}
	if group.ID != groupID {
		t.Fatalf("unexpected group: %+v", group)
	}

	for _, deviceID := range []string{"device-1", "device-2", "device-3"} {
		if err := client.AddGroupMember(ctx, groupID, deviceID); err != nil {
			t.Fatalf("add member %s failed: %v", deviceID, err)
		}
	}

	list, err := client.GroupMembers(ctx, groupID)
	if err != nil {
		t.Fatalf("list members failed: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 members, got %d: %+v", len(list), list)
	}

	if err := client.RemoveGroupMember(ctx, groupID, "device-2"); err != nil {
		t.Fatalf("remove member failed: %v", err)
	}

	list, err = client.GroupMembers(ctx, groupID)
	if err != nil {
		t.Fatalf("list members after removal failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 members after removal, got %d: %+v", len(list), list)
	}
	var ids []string
	for _, e := range list {
		ids = append(ids, e.ID)
	}
	sort.Strings(ids)
	if want := []string{"device-1", "device-3"}; !equalStrings(ids, want) {
		t.Fatalf("unexpected remaining members: got %v want %v", ids, want)
	}
}

// TestEntityGroupsReturnsAllMemberships is acceptance criterion 3: EntityGroups
// for a member returns every group it belongs to.
func TestEntityGroupsReturnsAllMemberships(t *testing.T) {
	const entityID = "device-1"
	membership := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			id := payload.Variables["input"].(map[string]any)["name"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON(id, id, testTenantID, "")},
			})
		case strings.Contains(payload.Query, "addGroupMember"):
			membership[payload.Variables["groupId"].(string)] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"addGroupMember": true}})
		case strings.Contains(payload.Query, "entityGroups"):
			if payload.Variables["entityId"] != entityID {
				t.Fatalf("unexpected entity id: %+v", payload.Variables)
			}
			var ids []string
			for id := range membership {
				ids = append(ids, id)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"entityGroups": ids}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	groupA, err := client.CreateObjectGroup(ctx, Group{Name: "group-a", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group a failed: %v", err)
	}
	groupB, err := client.CreateObjectGroup(ctx, Group{Name: "group-b", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group b failed: %v", err)
	}

	if err := client.AddGroupMember(ctx, groupA.ID, entityID); err != nil {
		t.Fatalf("add member to group a failed: %v", err)
	}
	if err := client.AddGroupMember(ctx, groupB.ID, entityID); err != nil {
		t.Fatalf("add member to group b failed: %v", err)
	}

	groups, err := client.EntityGroups(ctx, entityID)
	if err != nil {
		t.Fatalf("entity groups failed: %v", err)
	}
	if !equalStrings(sortedStrings(groups), sortedStrings([]string{groupA.ID, groupB.ID})) {
		t.Fatalf("unexpected entity groups: got %v want [%s %s]", groups, groupA.ID, groupB.ID)
	}
}

// TestGroupMembershipIsAdditiveNotMove is the most important test in this
// file: it guards against a regression back to single-membership/move
// semantics. A device added to two groups must appear in both member
// listings, and removing it from one group must leave the other membership
// fully intact.
func TestGroupMembershipIsAdditiveNotMove(t *testing.T) {
	const deviceID = "device-1"
	members := map[string]map[string]bool{
		"group-a": {},
		"group-b": {},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			id := payload.Variables["input"].(map[string]any)["name"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON("group-"+id, id, testTenantID, "")},
			})
		case strings.Contains(payload.Query, "addGroupMember"):
			groupID := payload.Variables["groupId"].(string)
			members[groupID][payload.Variables["entityId"].(string)] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"addGroupMember": true}})
		case strings.Contains(payload.Query, "removeGroupMember"):
			groupID := payload.Variables["groupId"].(string)
			delete(members[groupID], payload.Variables["entityId"].(string))
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"removeGroupMember": true}})
		case strings.Contains(payload.Query, "groupMembers"):
			groupID := payload.Variables["groupId"].(string)
			items := []map[string]any{}
			for id := range members[groupID] {
				items = append(items, entityJSON(id, testTenantID))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"groupMembers": items}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	groupA, err := client.CreateObjectGroup(ctx, Group{Name: "a", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group a failed: %v", err)
	}
	groupB, err := client.CreateObjectGroup(ctx, Group{Name: "b", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group b failed: %v", err)
	}

	if err := client.AddGroupMember(ctx, groupA.ID, deviceID); err != nil {
		t.Fatalf("add to group a failed: %v", err)
	}
	// Adding to a second group must not move the device out of the first.
	if err := client.AddGroupMember(ctx, groupB.ID, deviceID); err != nil {
		t.Fatalf("add to group b failed: %v", err)
	}

	membersA, err := client.GroupMembers(ctx, groupA.ID)
	if err != nil {
		t.Fatalf("list group a members failed: %v", err)
	}
	if len(membersA) != 1 || membersA[0].ID != deviceID {
		t.Fatalf("device missing from group a after joining group b: %+v", membersA)
	}

	membersB, err := client.GroupMembers(ctx, groupB.ID)
	if err != nil {
		t.Fatalf("list group b members failed: %v", err)
	}
	if len(membersB) != 1 || membersB[0].ID != deviceID {
		t.Fatalf("device missing from group b: %+v", membersB)
	}

	// Re-adding to a group it is already in must be idempotent.
	if err := client.AddGroupMember(ctx, groupA.ID, deviceID); err != nil {
		t.Fatalf("idempotent re-add failed: %v", err)
	}
	membersA, err = client.GroupMembers(ctx, groupA.ID)
	if err != nil || len(membersA) != 1 {
		t.Fatalf("re-add produced a duplicate: %+v (err %v)", membersA, err)
	}

	// Removing from group A must leave group B's membership fully intact.
	if err := client.RemoveGroupMember(ctx, groupA.ID, deviceID); err != nil {
		t.Fatalf("remove from group a failed: %v", err)
	}

	membersA, err = client.GroupMembers(ctx, groupA.ID)
	if err != nil {
		t.Fatalf("list group a members after removal failed: %v", err)
	}
	if len(membersA) != 0 {
		t.Fatalf("expected group a empty after removal, got %+v", membersA)
	}

	membersB, err = client.GroupMembers(ctx, groupB.ID)
	if err != nil {
		t.Fatalf("list group b members after removal from group a failed: %v", err)
	}
	if len(membersB) != 1 || membersB[0].ID != deviceID {
		t.Fatalf("group b membership was affected by removal from group a: %+v", membersB)
	}
}

// TestCreateNestedGroupAndChildGroups covers acceptance criterion 5: creating
// a nested group by passing ParentID at creation time, and ChildGroups on
// the parent returning it.
func TestCreateNestedGroupAndChildGroups(t *testing.T) {
	const parentID = "group-parent"
	const childID = "group-child"
	parents := map[string]string{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			input := payload.Variables["input"].(map[string]any)
			if _, ok := input["parentId"]; ok {
				t.Fatalf("createObjectGroup input must not carry parentId: %+v", input)
			}
			name := input["name"].(string)
			id := parentID
			if name == "child" {
				id = childID
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON(id, name, testTenantID, "")},
			})
		case strings.Contains(payload.Query, "setObjectGroupParent"):
			if strings.Contains(payload.Query, "id: $id") || strings.Contains(payload.Query, "parentId: $parentId") {
				t.Fatalf("setObjectGroupParent query used generic argument names: %s", payload.Query)
			}
			id := payload.Variables["objectGroupId"].(string)
			parent := payload.Variables["parentGroupId"].(string)
			parents[id] = parent
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"setObjectGroupParent": groupJSON(id, "child", testTenantID, parent)},
			})
		case strings.Contains(payload.Query, "childGroups"):
			if payload.Variables["parentId"] != parentID {
				t.Fatalf("unexpected parent id: %+v", payload.Variables)
			}
			items := []map[string]any{}
			for id, parent := range parents {
				if parent == parentID {
					items = append(items, groupJSON(id, "child", testTenantID, parent))
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"childGroups": map[string]any{"total": len(items), "items": items}},
			})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	parent, err := client.CreateObjectGroup(ctx, Group{Name: "parent", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create parent group failed: %v", err)
	}
	if parent.ID != parentID {
		t.Fatalf("unexpected parent group: %+v", parent)
	}

	child, err := client.CreateObjectGroup(ctx, Group{Name: "child", TenantID: testTenantID, ParentID: parent.ID})
	if err != nil {
		t.Fatalf("create nested group failed: %v", err)
	}
	if child.ID != childID || child.ParentID != parentID {
		t.Fatalf("unexpected nested group: %+v", child)
	}

	children, err := client.ChildGroups(ctx, parent.ID, 0, 0)
	if err != nil {
		t.Fatalf("child groups failed: %v", err)
	}
	if len(children.Items) != 1 || children.Items[0].ID != childID {
		t.Fatalf("unexpected child groups: %+v", children)
	}
}

func TestCreateNestedGroupDeletesCreatedGroupWhenReparentFails(t *testing.T) {
	const childID = "group-child"
	const invalidParentID = "missing-parent"
	created := map[string]bool{}
	deleted := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			input := payload.Variables["input"].(map[string]any)
			if _, ok := input["parentId"]; ok {
				t.Fatalf("createObjectGroup input must not carry parentId: %+v", input)
			}
			created[childID] = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON(childID, input["name"].(string), testTenantID, "")},
			})
		case strings.Contains(payload.Query, "setObjectGroupParent"):
			if strings.Contains(payload.Query, "id: $id") || strings.Contains(payload.Query, "parentId: $parentId") {
				t.Fatalf("setObjectGroupParent query used generic argument names: %s", payload.Query)
			}
			if payload.Variables["objectGroupId"] != childID || payload.Variables["parentGroupId"] != invalidParentID {
				t.Fatalf("unexpected setObjectGroupParent variables: %+v", payload.Variables)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{"message": "parent group not found"}},
			})
		case strings.Contains(payload.Query, "deleteGroup"):
			if payload.Variables["id"] != childID {
				t.Fatalf("unexpected deleteGroup variables: %+v", payload.Variables)
			}
			deleted[childID] = true
			delete(created, childID)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"deleteGroup": true}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	_, err := client.CreateObjectGroup(context.Background(), Group{
		Name:     "child",
		TenantID: testTenantID,
		ParentID: invalidParentID,
	})
	if err == nil {
		t.Fatal("expected create nested group to fail")
	}
	if !strings.Contains(err.Error(), "set parent for created group") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted[childID] {
		t.Fatal("created group was not deleted after reparent failure")
	}
	if created[childID] {
		t.Fatal("created group was left behind after reparent failure")
	}
}

// TestSetAndRemoveGroupParent covers acceptance criterion 6: reparenting an
// already-existing group and then detaching it.
func TestSetAndRemoveGroupParent(t *testing.T) {
	const groupAID = "group-a"
	const groupBID = "group-b"
	parent := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "setGroupParent"):
			if payload.Variables["id"] != groupBID || payload.Variables["parentId"] != groupAID {
				t.Fatalf("unexpected setGroupParent variables: %+v", payload.Variables)
			}
			parent = groupAID
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"setGroupParent": groupJSON(groupBID, "b", testTenantID, parent)},
			})
		case strings.Contains(payload.Query, "removeGroupParent"):
			if payload.Variables["id"] != groupBID {
				t.Fatalf("unexpected removeGroupParent variables: %+v", payload.Variables)
			}
			parent = ""
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"removeGroupParent": true}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	reparented, err := client.SetGroupParent(ctx, groupBID, groupAID)
	if err != nil {
		t.Fatalf("set group parent failed: %v", err)
	}
	if reparented.ParentID != groupAID {
		t.Fatalf("unexpected parent after set: %+v", reparented)
	}
	if parent != groupAID {
		t.Fatalf("server-side parent state not updated: %q", parent)
	}

	if err := client.RemoveGroupParent(ctx, groupBID); err != nil {
		t.Fatalf("remove group parent failed: %v", err)
	}
	if parent != "" {
		t.Fatalf("expected parent cleared, got %q", parent)
	}
}

func TestSetAndRemoveObjectGroupParentUseObjectArguments(t *testing.T) {
	const parentID = "object-group-a"
	const childID = "object-group-b"
	parent := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "setObjectGroupParent"):
			if strings.Contains(payload.Query, "id: $id") || strings.Contains(payload.Query, "parentId: $parentId") {
				t.Fatalf("setObjectGroupParent query used generic argument names: %s", payload.Query)
			}
			if payload.Variables["objectGroupId"] != childID || payload.Variables["parentGroupId"] != parentID {
				t.Fatalf("unexpected setObjectGroupParent variables: %+v", payload.Variables)
			}
			parent = parentID
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"setObjectGroupParent": groupJSON(childID, "b", testTenantID, parent)},
			})
		case strings.Contains(payload.Query, "removeObjectGroupParent"):
			if strings.Contains(payload.Query, "id: $id") {
				t.Fatalf("removeObjectGroupParent query used generic argument names: %s", payload.Query)
			}
			if payload.Variables["objectGroupId"] != childID {
				t.Fatalf("unexpected removeObjectGroupParent variables: %+v", payload.Variables)
			}
			parent = ""
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"removeObjectGroupParent": true}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	reparented, err := client.SetObjectGroupParent(ctx, childID, parentID)
	if err != nil {
		t.Fatalf("set object group parent failed: %v", err)
	}
	if reparented.ParentID != parentID {
		t.Fatalf("unexpected parent after set: %+v", reparented)
	}
	if parent != parentID {
		t.Fatalf("server-side object parent state not updated: %q", parent)
	}

	if err := client.RemoveObjectGroupParent(ctx, childID); err != nil {
		t.Fatalf("remove object group parent failed: %v", err)
	}
	if parent != "" {
		t.Fatalf("expected object parent cleared, got %q", parent)
	}
}

// TestObjectAndPrincipalGroupsNamespacesDoNotCrossContaminate covers
// acceptance criterion 7.
func TestObjectAndPrincipalGroupsNamespacesDoNotCrossContaminate(t *testing.T) {
	type storedGroup struct {
		id, name, groupType string
	}
	var stored []storedGroup

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			input := payload.Variables["input"].(map[string]any)
			name := input["name"].(string)
			stored = append(stored, storedGroup{id: "obj-" + name, name: name, groupType: "object"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createObjectGroup": groupJSON("obj-"+name, name, testTenantID, "")},
			})
		case strings.Contains(payload.Query, "createPrincipalGroup"):
			input := payload.Variables["input"].(map[string]any)
			name := input["name"].(string)
			stored = append(stored, storedGroup{id: "principal-" + name, name: name, groupType: "principal"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createPrincipalGroup": groupJSON("principal-"+name, name, testTenantID, "")},
			})
		case strings.Contains(payload.Query, "objectGroups"):
			items := []map[string]any{}
			for _, g := range stored {
				if g.groupType == "object" {
					items = append(items, groupJSON(g.id, g.name, testTenantID, ""))
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"objectGroups": map[string]any{"total": len(items), "items": items}},
			})
		case strings.Contains(payload.Query, "principalGroups"):
			items := []map[string]any{}
			for _, g := range stored {
				if g.groupType == "principal" {
					items = append(items, groupJSON(g.id, g.name, testTenantID, ""))
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"principalGroups": map[string]any{"total": len(items), "items": items}},
			})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	if _, err := client.CreateObjectGroup(ctx, Group{Name: "devices", TenantID: testTenantID}); err != nil {
		t.Fatalf("create object group failed: %v", err)
	}
	if _, err := client.CreatePrincipalGroup(ctx, Group{Name: "operators", TenantID: testTenantID}); err != nil {
		t.Fatalf("create principal group failed: %v", err)
	}

	objectGroups, err := client.ObjectGroups(ctx, Query{TenantID: testTenantID})
	if err != nil {
		t.Fatalf("list object groups failed: %v", err)
	}
	if len(objectGroups.Items) != 1 || objectGroups.Items[0].Name != "devices" {
		t.Fatalf("unexpected object groups: %+v", objectGroups)
	}

	principalGroups, err := client.PrincipalGroups(ctx, Query{TenantID: testTenantID})
	if err != nil {
		t.Fatalf("list principal groups failed: %v", err)
	}
	if len(principalGroups.Items) != 1 || principalGroups.Items[0].Name != "operators" {
		t.Fatalf("unexpected principal groups: %+v", principalGroups)
	}
}

func TestObjectGroupsUsesCurrentAtomSchemaArguments(t *testing.T) {
	const parentID = "customer-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		if strings.Contains(payload.Query, "includeDescendants") {
			t.Fatalf("objectGroups query includes unsupported includeDescendants argument: %s", payload.Query)
		}
		if _, ok := payload.Variables["includeDescendants"]; ok {
			t.Fatalf("objectGroups variables include unsupported includeDescendants: %+v", payload.Variables)
		}
		if payload.Variables["parentId"] != parentID {
			t.Fatalf("parentId variable = %v, want %s", payload.Variables["parentId"], parentID)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"objectGroups": map[string]any{
				"total": 1,
				"items": []map[string]any{groupJSON("site-1", "Site 1", testTenantID, parentID)},
			}},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	groups, err := client.ObjectGroups(context.Background(), Query{
		TenantID: testTenantID,
		ParentID: parentID,
	})
	if err != nil {
		t.Fatalf("list object groups failed: %v", err)
	}
	if len(groups.Items) != 1 || groups.Items[0].ID != "site-1" {
		t.Fatalf("unexpected object groups: %+v", groups)
	}
}

// TestAddGroupMemberRejectsCrossTenant covers acceptance criterion 8: Atom
// rejects adding a member from a different tenant, and the client must
// surface that failure rather than swallow it.
func TestAddGroupMemberRejectsCrossTenant(t *testing.T) {
	const groupID = "group-1"
	members := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := decodePayload(t, r)
		switch {
		case strings.Contains(payload.Query, "addGroupMember"):
			entityID := payload.Variables["entityId"].(string)
			if entityID == "foreign-device" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]string{{"message": "forbidden: entity belongs to a different tenant than the group"}},
				})
				return
			}
			members[entityID] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"addGroupMember": true}})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	err := client.AddGroupMember(ctx, groupID, "foreign-device")
	if err == nil {
		t.Fatalf("expected cross-tenant membership to be rejected")
	}
	atomErr, ok := err.(Error)
	if !ok {
		t.Fatalf("expected atom.Error, got %T: %v", err, err)
	}
	if atomErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-tenant membership, got %d", atomErr.StatusCode)
	}
	if members["foreign-device"] {
		t.Fatalf("cross-tenant member must not have been recorded")
	}
}
