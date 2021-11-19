// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
)

// PolicyReq represents an argument struct for making a policy related
// function calls.
type PolicyReq struct {
	Subject  string
	Object   string
	Relation string
}

type PolicyPage struct {
	Policies []string
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr PolicyReq) error

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error

	// ListPolicies lists policies based on the given PolicyReq structure.
	ListPolicies(ctx context.Context, pr PolicyReq) (PolicyPage, error)
}

// PolicyAgent facilitates the communication to authorization
// services and implements Authz functionalities for certain
// authorization services (e.g. ORY Keto).
type PolicyAgent interface {
	// CheckPolicy checks if the subject has a relation on the object.
	// It returns a non-nil error if the subject has no relation on
	// the object (which simply means the operation is denied).
	CheckPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	RetrievePolicies(ctx context.Context, pr PolicyReq) ([]*acl.RelationTuple, error)
}
