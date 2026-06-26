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
		TenantID:   pr.Domain,
		ScopeMode:  policyGrantScopeMode(pr),
		ObjectKind: policyGrantObjectKind(pr),
		ObjectType: policyGrantObjectType(pr),
		ObjectID:   policyGrantObjectID(pr),
		Effect:     "allow",
		Conditions: map[string]any{},
		ActionIDs:  []string{capID},
	})
	if err != nil {
		return err
	}
	_, err = writer.CreateDirectPolicy(ctx, CreateDirectPolicy{
		TenantID:          pr.Domain,
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
	page, err := writer.ListDirectPolicies(ctx, DirectPolicyQuery{
		TenantID:    pr.Domain,
		SubjectKind: policyGrantSubjectKind(pr),
		SubjectID:   policySubjectID(pr),
		Limit:       policyPageLimit,
	})
	if err != nil {
		return err
	}
	for _, policy := range page.Items {
		if !directPolicyMatches(policy, capID, pr) {
			continue
		}
		if err := writer.DeleteDirectPolicy(ctx, policy.ID); err != nil {
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
			ObjectType: entityKind(KindClient),
			TenantID:   pr.Domain,
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
	return pr.SubjectType == policies.UserType &&
		pr.Subject != "" &&
		pr.ObjectType == policies.ClientType &&
		pr.Permission == policies.ViewPermission
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
	case policies.DomainType:
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
	if pr.ObjectType == policies.ClientType {
		return atomObjectKindEntity + ":" + entityKind(KindClient)
	}
	switch pr.ObjectType {
	case policies.ChannelType:
		return "resource:" + KindChannel
	case policies.RulesType:
		return "resource:" + KindRule
	case policies.ReportsType:
		return "resource:" + KindReport
	case policies.AlarmsType:
		return "resource:" + KindAlarm
	case policies.GroupType:
		return ""
	default:
		return ""
	}
}

func policyGrantObjectID(pr policies.Policy) string {
	if policyGrantScopeMode(pr) != "object" {
		return ""
	}
	return policyResourceID(pr)
}

func directPolicyMatches(policy DirectPolicy, actionID string, pr policies.Policy) bool {
	block := policy.PermissionBlock
	if block.ID == "" || block.ScopeMode != policyGrantScopeMode(pr) {
		return false
	}
	if block.ObjectKind != policyGrantObjectKind(pr) ||
		block.ObjectType != policyGrantObjectType(pr) ||
		block.ObjectID != policyGrantObjectID(pr) {
		return false
	}
	for _, action := range block.Actions {
		if action.ID == actionID {
			return true
		}
	}
	return false
}
