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
	ListEntities(ctx context.Context, q Query) (EntityList, error)
}

type PolicyService struct {
	client policyClient
}

func NewPolicyService(client policyClient) PolicyService {
	return PolicyService{client: client}
}

func (ps PolicyService) AddPolicy(context.Context, policies.Policy) error {
	return errUnsupportedPolicyOperation
}

func (ps PolicyService) AddPolicies(context.Context, []policies.Policy) error {
	return errUnsupportedPolicyOperation
}

func (ps PolicyService) DeletePolicyFilter(context.Context, policies.Policy) error {
	return errUnsupportedPolicyOperation
}

func (ps PolicyService) DeletePolicies(context.Context, []policies.Policy) error {
	return errUnsupportedPolicyOperation
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
		page, err := ps.client.ListEntities(ctx, Query{
			Kind:   entityKind(KindClient),
			Limit:  policyPageLimit,
			Offset: offset,
		})
		if err != nil {
			return policies.PolicyPage{}, err
		}

		for _, entity := range page.Items {
			allowed, err := ps.client.CheckAuthz(ctx, AuthzRequest{
				SubjectID:  policySubjectID(pr),
				Action:     pr.Permission,
				ObjectKind: policyObjectKind(pr),
				ObjectID:   entity.ID,
				Context: map[string]any{
					"legacy_object_type": pr.ObjectType,
				},
			})
			if err != nil {
				return policies.PolicyPage{}, err
			}
			if allowed.Allowed {
				ids = append(ids, entity.ID)
			}
		}

		if uint64(len(page.Items)) < policyPageLimit || offset+uint64(len(page.Items)) >= page.Total {
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
