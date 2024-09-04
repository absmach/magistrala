// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"

	"github.com/absmach/magistrala"
)

//go:generate mockery --name PolicyService --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type PolicyService interface {
	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, req *magistrala.AddPolicyReq) (bool, error)

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, req *magistrala.AddPoliciesReq) (bool, error)

	// DeletePolicyFilter removes policy for given policy filter request.
	DeletePolicyFilter(ctx context.Context, req *magistrala.DeletePolicyFilterReq) (bool, error)

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, req *magistrala.DeletePoliciesReq) (bool, error)

	// ListObjects lists policies based on the given PolicyReq structure.
	ListObjects(ctx context.Context, req *magistrala.ListObjectsReq) ([]string, error)

	// ListAllObjects lists all policies based on the given PolicyReq structure.
	ListAllObjects(ctx context.Context, req *magistrala.ListObjectsReq) ([]string, error)

	// CountObjects count policies based on the given PolicyReq structure.
	CountObjects(ctx context.Context, req *magistrala.CountObjectsReq) (uint64, error)

	// ListSubjects lists subjects based on the given PolicyReq structure.
	ListSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) ([]string, error)

	// ListAllSubjects lists all subjects based on the given PolicyReq structure.
	ListAllSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) ([]string, error)

	// CountSubjects count policies based on the given PolicyReq structure.
	CountSubjects(ctx context.Context, req *magistrala.CountSubjectsReq) (uint64, error)

	// ListPermissions lists permission betweeen given subject and object .
	ListPermissions(ctx context.Context, req *magistrala.ListPermissionsReq) ([]string, error)

	// DeleteEntityPolicies deletes all policies for the given entity.
	DeleteEntityPolicies(ctx context.Context, req *magistrala.DeleteEntityPoliciesReq) (bool, error)
}
