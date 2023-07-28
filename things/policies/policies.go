// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
	"time"

	"github.com/mainflux/mainflux/internal/apiutil"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

// PolicyTypes contains a list of the available policy types currently supported.
var PolicyTypes = []string{WriteAction, ReadAction}

// Policy represents an argument struct for making a policy related function calls.
type Policy struct {
	OwnerID   string    `json:"owner_id"`
	Subject   string    `json:"subject"`
	Object    string    `json:"object"`
	Actions   []string  `json:"actions"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// AccessRequest represents an access control request for Authorization.
type AccessRequest struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
	Entity  string `json:"entity"`
}

// PolicyPage contains a page of policies.
type PolicyPage struct {
	Page
	Policies []Policy `json:"policies"`
}

// Repository specifies an account persistence API.
type Repository interface {
	// Save creates a policy for the given Subject, so that, after
	// Save, `Subject` has a `relation` on `group_id`. Returns a non-nil
	// error in case of failures.
	Save(ctx context.Context, p Policy) (Policy, error)

	// EvaluateMessagingAccess is used to evaluate if thing has access to channel.
	EvaluateMessagingAccess(ctx context.Context, ar AccessRequest) (Policy, error)

	// EvaluateThingAccess is used to evaluate if user has access to a thing.
	EvaluateThingAccess(ctx context.Context, ar AccessRequest) (Policy, error)

	// EvaluateGroupAccess is used to evaluate if user has access to a group.
	EvaluateGroupAccess(ctx context.Context, ar AccessRequest) (Policy, error)

	// Update updates the policy type.
	Update(ctx context.Context, p Policy) (Policy, error)

	// Retrieve retrieves policy for a given input.
	Retrieve(ctx context.Context, pm Page) (PolicyPage, error)

	// Delete deletes the policy
	Delete(ctx context.Context, p Policy) error
}

// Service represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Service interface {
	// Authorize is used to check if subject has access to object with the specified action.
	// Authorize returns non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	// Authorize is used to check if user has access to thing and group.
	// Authorize is also used to check if things has access to group i.e they are connected.
	Authorize(ctx context.Context, ar AccessRequest) (Policy, error)

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, token string, p Policy) (Policy, error)

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, token string, p Policy) error

	// UpdatePolicy updates an existing policy
	UpdatePolicy(ctx context.Context, token string, p Policy) (Policy, error)

	// ListPolicies lists existing policies
	ListPolicies(ctx context.Context, token string, p Page) (PolicyPage, error)
}

// Cache contains channel-thing connection caching interface.
type Cache interface {
	// Put adds policy to cahce.
	Put(ctx context.Context, policy Policy) error

	// Get retrieves policy from cache.
	Get(ctx context.Context, policy Policy) (Policy, error)

	// Remove deletes a policy from cache.
	Remove(ctx context.Context, policy Policy) error
}

// Validate returns an error if policy representation is invalid.
func (p Policy) Validate() error {
	if p.Subject == "" {
		return apiutil.ErrMissingPolicySub
	}
	if p.Object == "" {
		return apiutil.ErrMissingPolicyObj
	}
	if len(p.Actions) == 0 {
		return apiutil.ErrMalformedPolicyAct
	}
	for _, p := range p.Actions {
		// Validate things policies first
		if ok := ValidateAction(p); !ok {
			// Validate users policies for clients connected to a group
			if ok := upolicies.ValidateAction(p); !ok {
				return apiutil.ErrMalformedPolicyAct
			}
		}
	}
	return nil
}

// ValidateAction check if the action is in policies.
func ValidateAction(act string) bool {
	for _, v := range PolicyTypes {
		if v == act {
			return true
		}
	}
	return false
}
