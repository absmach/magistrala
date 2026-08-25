// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/policies"
)

func writeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

// groupGrantAtom is a minimal in-memory stand-in for the slice of Atom that
// group-scoped grants depend on: capability lookup, permission blocks,
// direct policies, group membership and hierarchy, and authorization
// decisions for the group_direct_objects / group_descendant_objects scope
// modes. It exists to prove the Go client wiring reaches Atom correctly, not
// to reimplement Atom's engine in full.
type groupGrantAtom struct {
	mu sync.Mutex

	nextID int

	capsByName map[string]string // capability name -> id
	capsByID   map[string]string // capability id -> name

	groups map[string]*Group // groupID -> group (ParentID mutated in place)

	blocks      map[string]PermissionBlock
	policies    map[string]DirectPolicy
	policyOrder []string // insertion order, for stable pagination

	members map[string]map[string]bool // groupID -> entityID -> direct member

	mutations map[string]int // GraphQL field name -> times invoked
}

func newGroupGrantAtom() *groupGrantAtom {
	return &groupGrantAtom{
		capsByName: map[string]string{"read": "cap-read", "write": "cap-write"},
		capsByID:   map[string]string{"cap-read": "read", "cap-write": "write"},
		groups:     map[string]*Group{},
		blocks:     map[string]PermissionBlock{},
		policies:   map[string]DirectPolicy{},
		members:    map[string]map[string]bool{},
		mutations:  map[string]int{},
	}
}

func (a *groupGrantAtom) newID(prefix string) string {
	a.nextID++
	return fmt.Sprintf("%s-%d", prefix, a.nextID)
}

func (a *groupGrantAtom) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload := decodePayload(t, r)
		assertNoLegacyGroupMembershipQuery(t, payload.Query)

		a.mu.Lock()
		defer a.mu.Unlock()

		switch {
		case strings.Contains(payload.Query, "createObjectGroup"):
			a.mutations["createObjectGroup"]++
			a.handleCreateObjectGroup(t, w, payload)
		case strings.Contains(payload.Query, "setObjectGroupParent"):
			a.mutations["setObjectGroupParent"]++
			a.handleSetObjectGroupParent(t, w, payload)
		case strings.Contains(payload.Query, "addEntityToObjectGroup"):
			a.mutations["addEntityToObjectGroup"]++
			a.handleAddGroupMember(t, w, payload)
		case strings.Contains(payload.Query, "removeEntityFromObjectGroup"):
			a.mutations["removeEntityFromObjectGroup"]++
			a.handleRemoveGroupMember(t, w, payload)
		case strings.Contains(payload.Query, "query Actions"):
			a.handleActions(w)
		case strings.Contains(payload.Query, "createPermissionBlock"):
			a.mutations["createPermissionBlock"]++
			a.handleCreatePermissionBlock(t, w, payload)
		case strings.Contains(payload.Query, "createDirectPolicy"):
			a.mutations["createDirectPolicy"]++
			a.handleCreateDirectPolicy(t, w, payload)
		case strings.Contains(payload.Query, "directPolicies"):
			a.handleDirectPolicies(w, payload)
		case strings.Contains(payload.Query, "deleteDirectPolicy"):
			a.mutations["deleteDirectPolicy"]++
			a.handleDeleteDirectPolicy(t, w, payload)
		case strings.Contains(payload.Query, "authzCheck"):
			a.handleAuthzCheck(t, w, payload)
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
}

func strVar(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func (a *groupGrantAtom) handleCreateObjectGroup(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	input, _ := payload.Variables[atomInputKeyInput].(map[string]any)
	id := strVar(input, "id")
	if id == "" {
		id = a.newID("group")
	}
	group := &Group{
		ID:       id,
		Name:     strVar(input, "name"),
		TenantID: strVar(input, "tenantId"),
		Status:   atomStatusActive,
	}
	a.groups[id] = group
	a.members[id] = map[string]bool{}
	writeGraphQLData(t, w, "createObjectGroup", *group)
}

func (a *groupGrantAtom) handleSetObjectGroupParent(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	id, _ := payload.Variables["objectGroupId"].(string)
	parentID, _ := payload.Variables["parentGroupId"].(string)
	group, ok := a.groups[id]
	if !ok {
		t.Fatalf("setObjectGroupParent: unknown group %q", id)
	}
	group.ParentID = parentID
	writeGraphQLData(t, w, "setObjectGroupParent", *group)
}

func (a *groupGrantAtom) handleAddGroupMember(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	groupID, _ := payload.Variables[atomInputKeyObjectGroupID].(string)
	entityID, _ := payload.Variables[atomInputKeyEntityID].(string)
	if a.members[groupID] == nil {
		a.members[groupID] = map[string]bool{}
	}
	a.members[groupID][entityID] = true
	writeGraphQLData(t, w, "addEntityToObjectGroup", map[string]any{"id": entityID})
}

func (a *groupGrantAtom) handleRemoveGroupMember(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	groupID, _ := payload.Variables[atomInputKeyObjectGroupID].(string)
	entityID, _ := payload.Variables[atomInputKeyEntityID].(string)
	delete(a.members[groupID], entityID)
	writeGraphQLData(t, w, "removeEntityFromObjectGroup", map[string]any{"id": entityID})
}

func (a *groupGrantAtom) handleActions(w http.ResponseWriter) {
	items := make([]Capability, 0, len(a.capsByName))
	for name, id := range a.capsByName {
		items = append(items, Capability{ID: id, Name: name})
	}
	_ = writeJSON(w, map[string]any{
		"data": map[string]any{
			"actions": CapabilityList{Items: items, Total: int64(len(items))},
		},
	})
}

func (a *groupGrantAtom) handleCreatePermissionBlock(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	input, _ := payload.Variables[atomInputKeyInput].(map[string]any)
	id := a.newID("block")

	var actions []Capability
	if raw, ok := input["actionIds"].([]any); ok {
		for _, v := range raw {
			capID, _ := v.(string)
			actions = append(actions, Capability{ID: capID, Name: a.capsByID[capID]})
		}
	}

	block := PermissionBlock{
		ID:         id,
		TenantID:   strVar(input, "tenantId"),
		ScopeMode:  strVar(input, "scopeMode"),
		ObjectKind: strVar(input, "objectKind"),
		ObjectType: strVar(input, "objectType"),
		ObjectID:   strVar(input, "objectId"),
		GroupID:    strVar(input, "groupId"),
		Effect:     strVar(input, "effect"),
		Actions:    actions,
	}
	a.blocks[id] = block
	writeGraphQLData(t, w, "createPermissionBlock", block)
}

func (a *groupGrantAtom) handleCreateDirectPolicy(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	input, _ := payload.Variables[atomInputKeyInput].(map[string]any)
	id := a.newID("policy")
	blockID := strVar(input, "permissionBlockId")

	policy := DirectPolicy{
		ID:                id,
		TenantID:          strVar(input, "tenantId"),
		SubjectKind:       strVar(input, "subjectKind"),
		SubjectID:         strVar(input, atomInputKeySubjectID),
		PermissionBlockID: blockID,
		PermissionBlock:   a.blocks[blockID],
	}
	a.policies[id] = policy
	a.policyOrder = append(a.policyOrder, id)
	writeGraphQLData(t, w, "createDirectPolicy", policy)
}

func (a *groupGrantAtom) handleDirectPolicies(w http.ResponseWriter, payload gqlPayload) {
	tenantID, hasTenant := payload.Variables["tenantId"].(string)
	subjectKind, hasSubjectKind := payload.Variables["subjectKind"].(string)
	subjectID, hasSubjectID := payload.Variables[atomInputKeySubjectID].(string)

	var matched []DirectPolicy
	for _, id := range a.policyOrder {
		policy, ok := a.policies[id]
		if !ok {
			continue
		}
		if hasTenant && policy.TenantID != tenantID {
			continue
		}
		if hasSubjectKind && policy.SubjectKind != subjectKind {
			continue
		}
		if hasSubjectID && policy.SubjectID != subjectID {
			continue
		}
		matched = append(matched, policy)
	}

	offset := 0
	if v, ok := payload.Variables["offset"].(float64); ok {
		offset = int(v)
	}
	limit := len(matched)
	if v, ok := payload.Variables["limit"].(float64); ok {
		limit = int(v)
	}

	start := offset
	if start > len(matched) {
		start = len(matched)
	}
	end := start + limit
	if end > len(matched) {
		end = len(matched)
	}

	_ = writeJSON(w, map[string]any{
		"data": map[string]any{
			"directPolicies": DirectPolicyList{
				Items: matched[start:end],
				Total: uint64(len(matched)),
			},
		},
	})
}

func (a *groupGrantAtom) handleDeleteDirectPolicy(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	id, _ := payload.Variables["id"].(string)
	delete(a.policies, id)
	for i, existing := range a.policyOrder {
		if existing == id {
			a.policyOrder = append(a.policyOrder[:i], a.policyOrder[i+1:]...)
			break
		}
	}
	writeGraphQLData(t, w, "deleteDirectPolicy", true)
}

func (a *groupGrantAtom) handleAuthzCheck(t *testing.T, w http.ResponseWriter, payload gqlPayload) {
	t.Helper()
	input, _ := payload.Variables[atomInputKeyInput].(map[string]any)
	subjectID := strVar(input, atomInputKeySubjectID)
	action := strVar(input, atomInputKeyAction)
	objectID := strVar(input, "objectId")

	allowed := a.authorized(subjectID, action, objectID)
	writeGraphQLData(t, w, "authzCheck", AuthzResponse{Allowed: allowed})
}

func (a *groupGrantAtom) authorized(subjectID, action, objectID string) bool {
	capID, ok := a.capsByName[action]
	if !ok {
		return false
	}
	for _, id := range a.policyOrder {
		policy, ok := a.policies[id]
		if !ok || policy.SubjectID != subjectID {
			continue
		}
		block := policy.PermissionBlock
		if block.Effect != atomDecisionAllow || !blockGrantsCapability(block, capID) {
			continue
		}
		if a.blockCoversObject(block, objectID) {
			return true
		}
	}
	return false
}

func blockGrantsCapability(block PermissionBlock, capID string) bool {
	for _, action := range block.Actions {
		if action.ID == capID {
			return true
		}
	}
	return false
}

func (a *groupGrantAtom) blockCoversObject(block PermissionBlock, objectID string) bool {
	switch block.ScopeMode {
	case atomScopeModeGroupDirectObjects:
		return a.members[block.GroupID][objectID]
	case atomScopeModeGroupDescendantObjects:
		for groupID, members := range a.members {
			if members[objectID] && a.isStrictDescendant(groupID, block.GroupID) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// isStrictDescendant reports whether groupID is a strict descendant of
// ancestorID by walking groupID's parent chain. This mirrors Atom's actual
// group_descendant_objects semantics: a group's own direct members are not
// covered by a descendant-scoped grant on that same group, only members of
// groups nested underneath it.
func (a *groupGrantAtom) isStrictDescendant(groupID, ancestorID string) bool {
	group, ok := a.groups[groupID]
	if !ok {
		return false
	}
	current := group.ParentID
	for current != "" {
		if current == ancestorID {
			return true
		}
		parent, ok := a.groups[current]
		if !ok {
			return false
		}
		current = parent.ParentID
	}
	return false
}

func writeGraphQLData(t *testing.T, w http.ResponseWriter, field string, value any) {
	t.Helper()
	if err := writeJSON(w, map[string]any{"data": map[string]any{field: value}}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func checkAuthz(t *testing.T, ctx context.Context, client *Client, subjectID, action, objectID string) bool {
	t.Helper()
	res, err := client.CheckAuthz(ctx, AuthzRequest{
		SubjectID:  subjectID,
		Action:     action,
		ResourceID: objectID,
		ObjectKind: atomObjectKindEntity,
		ObjectID:   objectID,
	})
	if err != nil {
		t.Fatalf("authz check failed: %v", err)
	}
	return res.Allowed
}

// TestGroupGrantAuthorizationFollowsMembership covers acceptance criteria
// 2-5: a grant over a group authorizes its members, denies non-members, and
// tracks membership changes with no further policy writes.
func TestGroupGrantAuthorizationFollowsMembership(t *testing.T) {
	fake := newGroupGrantAtom()
	srv := fake.server(t)
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	svc := NewPolicyService(client)
	ctx := context.Background()

	group, err := client.CreateObjectGroup(ctx, Group{Name: "meters", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	err = svc.GrantGroupAccess(ctx, GroupGrant{
		TenantID:    testTenantID,
		GroupID:     group.ID,
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "customer-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.DeviceType,
		Actions:     []string{"read"},
	})
	if err != nil {
		t.Fatalf("grant group access failed: %v", err)
	}

	if err := client.AddGroupMember(ctx, group.ID, "meter-a"); err != nil {
		t.Fatalf("add member failed: %v", err)
	}

	// Criterion 2: a member of the granted group is authorized.
	if !checkAuthz(t, ctx, client, "customer-1", "read", "meter-a") {
		t.Fatal("expected member of granted group to be authorized")
	}

	// Criterion 3: a device that was never added to the group is denied.
	if checkAuthz(t, ctx, client, "customer-1", "read", "meter-b") {
		t.Fatal("expected non-member to be denied")
	}

	// Criterion 4: adding a member via AddGroupMember alone, with no further
	// call into the policy API, is enough to authorize it.
	if err := client.AddGroupMember(ctx, group.ID, "meter-b"); err != nil {
		t.Fatalf("add second member failed: %v", err)
	}
	if !checkAuthz(t, ctx, client, "customer-1", "read", "meter-b") {
		t.Fatal("expected newly added member to be authorized without a further policy write")
	}
	if fake.mutations["createPermissionBlock"] != 1 || fake.mutations["createDirectPolicy"] != 1 {
		t.Fatalf("expected membership changes to not write policy, got blocks=%d policies=%d",
			fake.mutations["createPermissionBlock"], fake.mutations["createDirectPolicy"])
	}

	// Criterion 5: removing a member revokes its access through this group.
	if err := client.RemoveGroupMember(ctx, group.ID, "meter-a"); err != nil {
		t.Fatalf("remove member failed: %v", err)
	}
	if checkAuthz(t, ctx, client, "customer-1", "read", "meter-a") {
		t.Fatal("expected removed member to be denied")
	}
}

// TestGroupGrantIncludeDescendantsReachesChildGroupMembers is acceptance
// criterion 6: IncludeDescendants:true reaches members of a child group;
// IncludeDescendants:false does not.
func TestGroupGrantIncludeDescendantsReachesChildGroupMembers(t *testing.T) {
	fake := newGroupGrantAtom()
	srv := fake.server(t)
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	svc := NewPolicyService(client)
	ctx := context.Background()

	parent, err := client.CreateObjectGroup(ctx, Group{Name: "site", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create parent group failed: %v", err)
	}
	child, err := client.CreateObjectGroup(ctx, Group{Name: "wing", TenantID: testTenantID, ParentID: parent.ID})
	if err != nil {
		t.Fatalf("create child group failed: %v", err)
	}

	if err := client.AddGroupMember(ctx, parent.ID, "meter-in-parent"); err != nil {
		t.Fatalf("add member to parent failed: %v", err)
	}
	if err := client.AddGroupMember(ctx, child.ID, "meter-in-child"); err != nil {
		t.Fatalf("add member to child failed: %v", err)
	}

	directGrant := GroupGrant{
		TenantID:    testTenantID,
		GroupID:     parent.ID,
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "direct-customer",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.DeviceType,
		Actions:     []string{"read"},
	}
	if err := svc.GrantGroupAccess(ctx, directGrant); err != nil {
		t.Fatalf("grant direct access failed: %v", err)
	}

	descendantGrant := GroupGrant{
		TenantID:           testTenantID,
		GroupID:            parent.ID,
		SubjectKind:        atomObjectKindEntity,
		SubjectID:          "descendant-customer",
		ObjectKind:         atomObjectKindEntity,
		ObjectType:         policies.DeviceType,
		Actions:            []string{"read"},
		IncludeDescendants: true,
	}
	if err := svc.GrantGroupAccess(ctx, descendantGrant); err != nil {
		t.Fatalf("grant descendant access failed: %v", err)
	}

	if !checkAuthz(t, ctx, client, "direct-customer", "read", "meter-in-parent") {
		t.Fatal("direct grant should authorize a direct member of the granted group")
	}
	if checkAuthz(t, ctx, client, "direct-customer", "read", "meter-in-child") {
		t.Fatal("direct grant (IncludeDescendants: false) must not reach a child group's members")
	}

	if !checkAuthz(t, ctx, client, "descendant-customer", "read", "meter-in-child") {
		t.Fatal("descendant grant (IncludeDescendants: true) should reach a child group's members")
	}
}

// TestRevokeGroupAccessDeniesOnlyThroughThatGroup is acceptance criterion 7,
// plus the semantic point that revoking access granted through one group
// does not touch access granted through a different group the same member
// happens to also belong to.
func TestRevokeGroupAccessDeniesOnlyThroughThatGroup(t *testing.T) {
	fake := newGroupGrantAtom()
	srv := fake.server(t)
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	svc := NewPolicyService(client)
	ctx := context.Background()

	groupA, err := client.CreateObjectGroup(ctx, Group{Name: "group-a", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group a failed: %v", err)
	}
	groupB, err := client.CreateObjectGroup(ctx, Group{Name: "group-b", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group b failed: %v", err)
	}

	grantA := GroupGrant{
		TenantID:    testTenantID,
		GroupID:     groupA.ID,
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "customer-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.DeviceType,
		Actions:     []string{"read"},
	}
	grantB := grantA
	grantB.GroupID = groupB.ID

	if err := svc.GrantGroupAccess(ctx, grantA); err != nil {
		t.Fatalf("grant access to group a failed: %v", err)
	}
	if err := svc.GrantGroupAccess(ctx, grantB); err != nil {
		t.Fatalf("grant access to group b failed: %v", err)
	}

	// meter-only-a is reachable only through group A; meter-both is a member
	// of both groups.
	for _, member := range []struct{ group, entity string }{
		{groupA.ID, "meter-only-a"},
		{groupA.ID, "meter-both"},
		{groupB.ID, "meter-both"},
	} {
		if err := client.AddGroupMember(ctx, member.group, member.entity); err != nil {
			t.Fatalf("add member %s to %s failed: %v", member.entity, member.group, err)
		}
	}

	if !checkAuthz(t, ctx, client, "customer-1", "read", "meter-only-a") ||
		!checkAuthz(t, ctx, client, "customer-1", "read", "meter-both") {
		t.Fatal("expected both meters to be authorized before revocation")
	}

	if err := svc.RevokeGroupAccess(ctx, grantA); err != nil {
		t.Fatalf("revoke access to group a failed: %v", err)
	}

	// Criterion 7: a member reachable only through the revoked group is now denied.
	if checkAuthz(t, ctx, client, "customer-1", "read", "meter-only-a") {
		t.Fatal("expected meter reachable only through the revoked group to be denied")
	}
	// Revocation through group A must not touch access granted through group B.
	if !checkAuthz(t, ctx, client, "customer-1", "read", "meter-both") {
		t.Fatal("expected meter still reachable through group b to remain authorized")
	}

	// RevokeGroupAccess, like the legacy DeletePolicyFilter path, removes the
	// direct policy but leaves the permission block behind.
	if len(fake.policies) != 1 {
		t.Fatalf("expected exactly one surviving direct policy (group b's), got %d", len(fake.policies))
	}
}

// TestGrantGroupAccessWriteCountIsConstantRegardlessOfMemberCount is
// acceptance criterion 8: granting access to a group with many members costs
// a fixed number of policy-write mutations, not one per member.
func TestGrantGroupAccessWriteCountIsConstantRegardlessOfMemberCount(t *testing.T) {
	fake := newGroupGrantAtom()
	srv := fake.server(t)
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	svc := NewPolicyService(client)
	ctx := context.Background()

	group, err := client.CreateObjectGroup(ctx, Group{Name: "fleet", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	if err := svc.GrantGroupAccess(ctx, GroupGrant{
		TenantID:    testTenantID,
		GroupID:     group.ID,
		SubjectKind: atomObjectKindEntity,
		SubjectID:   "customer-1",
		ObjectKind:  atomObjectKindEntity,
		ObjectType:  policies.DeviceType,
		Actions:     []string{"read"},
	}); err != nil {
		t.Fatalf("grant group access failed: %v", err)
	}
	if fake.mutations["createPermissionBlock"] != 1 || fake.mutations["createDirectPolicy"] != 1 {
		t.Fatalf("expected exactly one block and one policy from the grant itself, got blocks=%d policies=%d",
			fake.mutations["createPermissionBlock"], fake.mutations["createDirectPolicy"])
	}

	const memberCount = 500
	for i := 0; i < memberCount; i++ {
		if err := client.AddGroupMember(ctx, group.ID, fmt.Sprintf("device-%d", i)); err != nil {
			t.Fatalf("add member %d failed: %v", i, err)
		}
	}

	if fake.mutations["addEntityToObjectGroup"] != memberCount {
		t.Fatalf("expected %d addEntityToObjectGroup mutations, got %d", memberCount, fake.mutations["addEntityToObjectGroup"])
	}
	// The write count from the grant itself must not have grown with member count.
	if fake.mutations["createPermissionBlock"] != 1 || fake.mutations["createDirectPolicy"] != 1 {
		t.Fatalf("expected policy-write mutations to stay constant, got blocks=%d policies=%d",
			fake.mutations["createPermissionBlock"], fake.mutations["createDirectPolicy"])
	}
}
