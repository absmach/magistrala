// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/policy"
)

type service struct {
	policy magistrala.PolicyServiceClient
}

func NewService(policyClient magistrala.PolicyServiceClient) policy.PolicyService {
	return &service{
		policy: policyClient,
	}
}

func (svc service) AddPolicy(ctx context.Context, req *magistrala.AddPolicyReq) (bool, error) {
	res, err := svc.policy.AddPolicy(ctx, req)
	if err != nil {
		return false, err
	}

	return res.GetAdded(), nil
}

func (svc service) AddPolicies(ctx context.Context, req *magistrala.AddPoliciesReq) (bool, error) {
	res, err := svc.policy.AddPolicies(ctx, req)
	if err != nil {
		return false, err
	}

	return res.GetAdded(), nil
}

func (svc service) DeletePolicyFilter(ctx context.Context, req *magistrala.DeletePolicyFilterReq) (bool, error) {
	res, err := svc.policy.DeletePolicyFilter(ctx, req)
	if err != nil {
		return false, err
	}
	return res.GetDeleted(), nil
}

func (svc service) DeletePolicies(ctx context.Context, req *magistrala.DeletePoliciesReq) (bool, error) {
	res, err := svc.policy.DeletePolicies(ctx, req)
	if err != nil {
		return false, err
	}
	return res.GetDeleted(), nil
}

func (svc service) ListObjects(ctx context.Context, req *magistrala.ListObjectsReq) ([]string, error) {
	res, err := svc.policy.ListObjects(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Policies, nil
}

func (svc service) ListAllObjects(ctx context.Context, req *magistrala.ListObjectsReq) ([]string, error) {
	res, err := svc.policy.ListAllObjects(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Policies, nil
}

func (svc service) CountObjects(ctx context.Context, req *magistrala.CountObjectsReq) (uint64, error) {
	res, err := svc.policy.CountObjects(ctx, req)
	if err != nil {
		return 0, err
	}

	return res.Count, nil
}

func (svc service) ListSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) ([]string, error) {
	res, err := svc.policy.ListSubjects(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Policies, nil
}

func (svc service) ListAllSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) ([]string, error) {
	res, err := svc.policy.ListAllSubjects(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Policies, nil
}

func (svc service) CountSubjects(ctx context.Context, req *magistrala.CountSubjectsReq) (uint64, error) {
	res, err := svc.policy.CountSubjects(ctx, req)
	if err != nil {
		return 0, err
	}

	return res.Count, nil
}

func (svc service) ListPermissions(ctx context.Context, req *magistrala.ListPermissionsReq) ([]string, error) {
	res, err := svc.policy.ListPermissions(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.GetPermissions(), nil
}

func (svc service) DeleteEntityPolicies(ctx context.Context, req *magistrala.DeleteEntityPoliciesReq) (bool, error) {
	res, err := svc.policy.DeleteEntityPolicies(ctx, req)
	if err != nil {
		return false, err
	}

	return res.GetDeleted(), nil
}
