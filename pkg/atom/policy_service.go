// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	stderrors "errors"

	"github.com/absmach/magistrala/pkg/policies"
)

const policyPageLimit uint64 = 100

var errUnsupportedPolicyOperation = stderrors.New("atom policy service: unsupported policy operation")

var errInvalidGroupGrant = stderrors.New("atom policy service: invalid group grant")

type policyClient interface {
	Authorizer
	AuthorizedObjectIDs(ctx context.Context, q AuthorizedObjectIDsQuery) (AuthorizedObjectIDs, error)
}

type policyWriter interface {
	CapabilityID(ctx context.Context, name string) (string, error)
	CreatePermissionBlock(ctx context.Context, block CreatePermissionBlock) (PermissionBlock, error)
	CreateDirectPolicy(ctx context.Context, policy CreateDirectPolicy) (DirectPolicy, error)
	ListDirectPolicies(ctx context.Context, q DirectPolicyQuery) (DirectPolicyList, error)
	DeleteDirectPolicy(ctx context.Context, id string) error
}

type PolicyService struct {
	client policyClient
}

func NewPolicyService(client policyClient) PolicyService {
	return PolicyService{client: client}
}

func (ps PolicyService) AddPolicy(ctx context.Context, pr policies.Policy) error {
	writer, ok := ps.client.(policyWriter)
	if !ok {
		return errUnsupportedPolicyOperation
	}
	capID, err := writer.CapabilityID(ctx, CapabilityName(pr.Permission))
	if err != nil {
		return err
	}
	block, err := writer.CreatePermissionBlock(ctx, CreatePermissionBlock{
		TenantID:   pr.Workspace,
		ScopeMode:  policyGrantScopeMode(pr),
		ObjectKind: policyGrantObjectKind(pr),
		ObjectType: policyGrantObjectType(pr),
		ObjectID:   policyGrantObjectID(pr),
		Effect:     atomDecisionAllow,
		Conditions: map[string]any{},
		ActionIDs:  []string{capID},
	})
	if err != nil {
		return err
	}
	_, err = writer.CreateDirectPolicy(ctx, CreateDirectPolicy{
		TenantID:          pr.Workspace,
		SubjectKind:       policyGrantSubjectKind(pr),
		SubjectID:         policySubjectID(pr),
		PermissionBlockID: block.ID,
	})
	return err
}

func (ps PolicyService) AddPolicies(ctx context.Context, prs []policies.Policy) error {
	for _, pr := range prs {
		if err := ps.AddPolicy(ctx, pr); err != nil {
			return err
		}
	}
	return nil
}

func (ps PolicyService) DeletePolicyFilter(ctx context.Context, pr policies.Policy) error {
	writer, ok := ps.client.(policyWriter)
	if !ok {
		return errUnsupportedPolicyOperation
	}
	capID, err := writer.CapabilityID(ctx, CapabilityName(pr.Permission))
	if err != nil {
		return err
	}

	// Collect matching IDs across all pages before deleting: deleting while
	// paginating would shift later pages under us and skip matches.
	match := policyBlockMatch(pr)
	var ids []string
	for offset := uint64(0); ; offset += policyPageLimit {
		page, err := writer.ListDirectPolicies(ctx, DirectPolicyQuery{
			TenantID:    pr.Workspace,
			SubjectKind: policyGrantSubjectKind(pr),
			SubjectID:   policySubjectID(pr),
			Limit:       policyPageLimit,
			Offset:      offset,
		})
		if err != nil {
			return err
		}
		for _, policy := range page.Items {
			if directPolicyMatches(policy, match) && blockHasAction(policy.PermissionBlock, capID) {
				ids = append(ids, policy.ID)
			}
		}
		if uint64(len(page.Items)) < policyPageLimit || offset+uint64(len(page.Items)) >= page.Total {
			break
		}
	}

	for _, id := range ids {
		if err := writer.DeleteDirectPolicy(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (ps PolicyService) DeletePolicies(ctx context.Context, prs []policies.Policy) error {
	for _, pr := range prs {
		if err := ps.DeletePolicyFilter(ctx, pr); err != nil {
			return err
		}
	}
	return nil
}

// GrantGroupAccess grants a subject the given actions over a group's
// members: direct members, or members of descendant groups if
// grant.IncludeDescendants is set. It writes exactly one permission block
// and one direct policy, regardless of how many members the group has now
// or gains later — membership changes (see the group membership client)
// grant or withdraw access on their own, with no further policy writes.
func (ps PolicyService) GrantGroupAccess(ctx context.Context, grant GroupGrant) error {
	writer, ok := ps.client.(policyWriter)
	if !ok {
		return errUnsupportedPolicyOperation
	}
	if err := validateGroupGrant(grant); err != nil {
		return err
	}

	actionIDs := make([]string, 0, len(grant.Actions))
	for _, action := range grant.Actions {
		capID, err := writer.CapabilityID(ctx, CapabilityName(action))
		if err != nil {
			return err
		}
		actionIDs = append(actionIDs, capID)
	}

	block, err := writer.CreatePermissionBlock(ctx, CreatePermissionBlock{
		TenantID:   grant.TenantID,
		ScopeMode:  groupGrantScopeMode(grant),
		ObjectKind: grant.ObjectKind,
		ObjectType: groupGrantObjectType(grant),
		GroupID:    grant.GroupID,
		Effect:     atomDecisionAllow,
		Conditions: map[string]any{},
		ActionIDs:  actionIDs,
	})
	if err != nil {
		return err
	}
	_, err = writer.CreateDirectPolicy(ctx, CreateDirectPolicy{
		TenantID:          grant.TenantID,
		SubjectKind:       grant.SubjectKind,
		SubjectID:         grant.SubjectID,
		PermissionBlockID: block.ID,
	})
	return err
}

// RevokeGroupAccess removes the permission block and direct policy that
// GrantGroupAccess created for this group. Group membership is many-to-many:
// this withdraws access granted through this specific group only. A member
// that also belongs to a different group with its own grant remains
// reachable through that grant afterwards — this method does not walk a
// member's other group memberships looking for other access to strip.
func (ps PolicyService) RevokeGroupAccess(ctx context.Context, grant GroupGrant) error {
	writer, ok := ps.client.(policyWriter)
	if !ok {
		return errUnsupportedPolicyOperation
	}
	if err := validateGroupGrant(grant); err != nil {
		return err
	}

	actionIDs := make([]string, 0, len(grant.Actions))
	for _, action := range grant.Actions {
		capID, err := writer.CapabilityID(ctx, CapabilityName(action))
		if err != nil {
			return err
		}
		actionIDs = append(actionIDs, capID)
	}

	match := groupGrantBlockMatch(grant)
	var ids []string
	for offset := uint64(0); ; offset += policyPageLimit {
		page, err := writer.ListDirectPolicies(ctx, DirectPolicyQuery{
			TenantID:    grant.TenantID,
			SubjectKind: grant.SubjectKind,
			SubjectID:   grant.SubjectID,
			Limit:       policyPageLimit,
			Offset:      offset,
		})
		if err != nil {
			return err
		}
		for _, policy := range page.Items {
			if directPolicyMatches(policy, match) && blockActionSetMatches(policy.PermissionBlock, actionIDs) {
				ids = append(ids, policy.ID)
			}
		}
		if uint64(len(page.Items)) < policyPageLimit || offset+uint64(len(page.Items)) >= page.Total {
			break
		}
	}

	for _, id := range ids {
		if err := writer.DeleteDirectPolicy(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// ListGroupGrants returns every group-scoped grant (direct or descendant)
// recorded against groupID, across all tenants and subjects.
func (ps PolicyService) ListGroupGrants(ctx context.Context, groupID string) ([]GroupGrant, error) {
	writer, ok := ps.client.(policyWriter)
	if !ok {
		return nil, errUnsupportedPolicyOperation
	}

	var grants []GroupGrant
	for offset := uint64(0); ; offset += policyPageLimit {
		page, err := writer.ListDirectPolicies(ctx, DirectPolicyQuery{
			Limit:  policyPageLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		for _, policy := range page.Items {
			if grant, ok := groupGrantFromDirectPolicy(policy, groupID); ok {
				grants = append(grants, grant)
			}
		}
		if uint64(len(page.Items)) < policyPageLimit || offset+uint64(len(page.Items)) >= page.Total {
			break
		}
	}
	return grants, nil
}

func (ps PolicyService) ListObjects(ctx context.Context, pr policies.Policy, _ string, limit uint64) (policies.PolicyPage, error) {
	page, err := ps.ListAllObjects(ctx, pr)
	if err != nil {
		return policies.PolicyPage{}, err
	}
	if limit == 0 || uint64(len(page.Policies)) <= limit {
		return page, nil
	}
	page.Policies = page.Policies[:limit]
	return page, nil
}

func (ps PolicyService) ListAllObjects(ctx context.Context, pr policies.Policy) (policies.PolicyPage, error) {
	if !isSupportedObjectList(pr) {
		return policies.PolicyPage{}, errUnsupportedPolicyOperation
	}

	var ids []string
	for offset := uint64(0); ; offset += policyPageLimit {
		page, err := ps.client.AuthorizedObjectIDs(ctx, AuthorizedObjectIDsQuery{
			SubjectID:  policySubjectID(pr),
			Action:     CapabilityName(pr.Permission),
			ObjectKind: policyObjectKind(pr),
			ObjectType: atomPolicyObjectType(pr.ObjectType),
			TenantID:   pr.Workspace,
			Limit:      policyPageLimit,
			Offset:     offset,
		})
		if err != nil {
			return policies.PolicyPage{}, err
		}

		ids = append(ids, page.IDs...)

		if uint64(len(page.IDs)) < policyPageLimit || offset+uint64(len(page.IDs)) >= page.Total {
			break
		}
	}

	return policies.PolicyPage{Policies: ids}, nil
}

func (ps PolicyService) CountObjects(ctx context.Context, pr policies.Policy) (uint64, error) {
	page, err := ps.ListAllObjects(ctx, pr)
	if err != nil {
		return 0, err
	}
	return uint64(len(page.Policies)), nil
}

func (ps PolicyService) ListSubjects(context.Context, policies.Policy, string, uint64) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, errUnsupportedPolicyOperation
}

func (ps PolicyService) ListAllSubjects(context.Context, policies.Policy) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, errUnsupportedPolicyOperation
}

func (ps PolicyService) CountSubjects(context.Context, policies.Policy) (uint64, error) {
	return 0, errUnsupportedPolicyOperation
}

func (ps PolicyService) ListPermissions(context.Context, policies.Policy, []string) (policies.Permissions, error) {
	return nil, errUnsupportedPolicyOperation
}

func isSupportedObjectList(pr policies.Policy) bool {
	if pr.SubjectType != policies.UserType || pr.Subject == "" || pr.ObjectType != policies.ClientType {
		return false
	}
	return pr.Permission == policies.ViewPermission || pr.Permission == atomActionRead
}

func policyGrantSubjectKind(pr policies.Policy) string {
	if pr.SubjectType == policies.GroupType || pr.SubjectKind == policies.GroupsKind {
		return atomObjectKindGroup
	}
	return atomObjectKindEntity
}

func policyGrantScopeMode(pr policies.Policy) string {
	switch pr.ObjectType {
	case policies.PlatformType:
		return "platform"
	case policies.WorkspaceType:
		return atomObjectKindTenant
	default:
		return atomScopeModeObject
	}
}

func policyGrantObjectKind(pr policies.Policy) string {
	if policyGrantScopeMode(pr) != atomScopeModeObject {
		return ""
	}
	switch pr.ObjectType {
	case policies.ClientType:
		return atomObjectKindEntity
	case policies.GroupType:
		return atomObjectKindGroup
	}
	return atomObjectKindResource
}

func policyGrantObjectType(pr policies.Policy) string {
	if policyGrantScopeMode(pr) != atomScopeModeObject {
		return ""
	}
	return atomPolicyObjectType(pr.ObjectType)
}

// atomPolicyObjectType is shared by the write and read paths so their object types cannot drift apart.
func atomPolicyObjectType(objectType string) string {
	switch objectType {
	case policies.ClientType:
		return atomObjectType(atomObjectKindEntity, entityKind(KindDevice))
	case policies.ChannelType:
		return atomObjectType(atomObjectKindResource, KindChannel)
	case policies.RulesType:
		return atomObjectType(atomObjectKindResource, KindRule)
	case policies.ReportsType:
		return atomObjectType(atomObjectKindResource, KindReport)
	case policies.AlarmsType:
		return atomObjectType(atomObjectKindResource, KindAlarm)
	default:
		return ""
	}
}

func policyGrantObjectID(pr policies.Policy) string {
	if policyGrantScopeMode(pr) != atomScopeModeObject {
		return ""
	}
	return policyResourceID(pr)
}

// blockMatch is the permission-block shape a direct policy's block must have
// to count as a match for a grant. It is shared by the legacy
// object/tenant/platform delete path and group-scoped revocation so both go
// through one comparison, including GroupID: two group-scoped blocks can
// otherwise be identical (same tenant, object kind/type, scope mode) and
// differ only in which group they grant access through.
type blockMatch struct {
	ScopeMode  string
	ObjectKind string
	ObjectType string
	ObjectID   string
	GroupID    string
}

func policyBlockMatch(pr policies.Policy) blockMatch {
	return blockMatch{
		ScopeMode:  policyGrantScopeMode(pr),
		ObjectKind: policyGrantObjectKind(pr),
		ObjectType: policyGrantObjectType(pr),
		ObjectID:   policyGrantObjectID(pr),
	}
}

func groupGrantBlockMatch(g GroupGrant) blockMatch {
	return blockMatch{
		ScopeMode:  groupGrantScopeMode(g),
		ObjectKind: g.ObjectKind,
		ObjectType: groupGrantObjectType(g),
		GroupID:    g.GroupID,
	}
}

func directPolicyMatches(policy DirectPolicy, match blockMatch) bool {
	block := policy.PermissionBlock
	if block.ID == "" {
		return false
	}
	return block.ScopeMode == match.ScopeMode &&
		block.ObjectKind == match.ObjectKind &&
		block.ObjectType == match.ObjectType &&
		block.ObjectID == match.ObjectID &&
		block.GroupID == match.GroupID
}

func blockHasAction(block PermissionBlock, actionID string) bool {
	for _, action := range block.Actions {
		if action.ID == actionID {
			return true
		}
	}
	return false
}

func blockActionSetMatches(block PermissionBlock, actionIDs []string) bool {
	if len(block.Actions) != len(actionIDs) {
		return false
	}
	for _, actionID := range actionIDs {
		if !blockHasAction(block, actionID) {
			return false
		}
	}
	return true
}

func groupGrantScopeMode(g GroupGrant) string {
	if g.IncludeDescendants {
		return atomScopeModeGroupDescendantObjects
	}
	return atomScopeModeGroupDirectObjects
}

// groupGrantObjectType reuses atomPolicyObjectType, the same mapping the
// object-scoped write and read paths share, so this cannot drift from them.
func groupGrantObjectType(g GroupGrant) string {
	return atomPolicyObjectType(g.ObjectType)
}

func groupGrantObjectKind(g GroupGrant) string {
	switch g.ObjectType {
	case policies.ClientType:
		return atomObjectKindEntity
	case policies.ChannelType, policies.RulesType, policies.ReportsType, policies.AlarmsType:
		return atomObjectKindResource
	default:
		return ""
	}
}

func groupGrantPolicyObjectType(objectType string) string {
	switch objectType {
	case atomPolicyObjectType(policies.ClientType):
		return policies.ClientType
	case atomPolicyObjectType(policies.ChannelType):
		return policies.ChannelType
	case atomPolicyObjectType(policies.RulesType):
		return policies.RulesType
	case atomPolicyObjectType(policies.ReportsType):
		return policies.ReportsType
	case atomPolicyObjectType(policies.AlarmsType):
		return policies.AlarmsType
	default:
		return ""
	}
}

func validateGroupGrant(g GroupGrant) error {
	if g.TenantID == "" || g.GroupID == "" || g.SubjectKind == "" || g.SubjectID == "" || len(g.Actions) == 0 {
		return errInvalidGroupGrant
	}
	if g.SubjectKind != atomObjectKindEntity && g.SubjectKind != atomObjectKindGroup {
		return errInvalidGroupGrant
	}
	if g.ObjectKind != atomObjectKindEntity && g.ObjectKind != atomObjectKindResource {
		return errInvalidGroupGrant
	}
	if groupGrantObjectType(g) == "" {
		return errInvalidGroupGrant
	}
	if g.ObjectKind != groupGrantObjectKind(g) {
		return errInvalidGroupGrant
	}
	return nil
}

func groupGrantFromDirectPolicy(policy DirectPolicy, groupID string) (GroupGrant, bool) {
	block := policy.PermissionBlock
	if block.GroupID != groupID {
		return GroupGrant{}, false
	}
	var includeDescendants bool
	switch block.ScopeMode {
	case atomScopeModeGroupDirectObjects:
		includeDescendants = false
	case atomScopeModeGroupDescendantObjects:
		includeDescendants = true
	default:
		return GroupGrant{}, false
	}
	actions := make([]string, 0, len(block.Actions))
	for _, action := range block.Actions {
		actions = append(actions, action.Name)
	}
	return GroupGrant{
		TenantID:           policy.TenantID,
		GroupID:            block.GroupID,
		SubjectKind:        policy.SubjectKind,
		SubjectID:          policy.SubjectID,
		ObjectKind:         block.ObjectKind,
		ObjectType:         groupGrantPolicyObjectType(block.ObjectType),
		Actions:            actions,
		IncludeDescendants: includeDescendants,
	}, true
}
