// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
	"strings"
	"time"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"golang.org/x/exp/slices"
)

// PolicyTypes contains a list of the available policy types currently supported
// They are arranged in the following order based on their priority:
//
// Group policies
//  1. g_add - adds a member to a group
//  2. g_delete - delete a group
//  3. g_update - update a group
//  4. g_list - list groups and their members
//
// Client policies
//  5. c_delete - delete a client
//  6. c_update - update a client
//  8. c_list - list clients
//
// Message policies
//  9. m_write - write a message
//  10. m_read - read a message
//
// Sharing policies
//  11. c_share - share a client - allows a user to add another user to a group.
var PolicyTypes = []string{"g_add", "g_delete", "g_update", "g_list", "c_delete", "c_update", "c_list", "m_write", "m_read", "c_share"}

// Policy represents an argument struct for making a policy related function calls.
type Policy struct {
	OwnerID   string    `json:"owner_id"`
	Subject   string    `json:"subject"`
	Object    string    `json:"object"`
	Actions   []string  `json:"actions"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	UpdatedBy string    `json:"updated_by,omitempty"`
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

// Repository specifies a policy persistence API.
type Repository interface {
	// Save creates a policy for the given Policy Subject and Object combination.
	// It returns an error if the policy already exists or the operation failed
	// otherwise it returns nil.
	Save(ctx context.Context, p Policy) error

	// CheckAdmin checks if the user is an admin user.
	// It returns an error if the user is not an admin user or the operation failed
	// otherwise it returns nil.
	CheckAdmin(ctx context.Context, id string) error

	// EvaluateUserAccess is used to evaluate if user has access to another user.
	// It returns an error and an empty policy if the user does not have access
	// otherwise it returns nil and the policy.
	EvaluateUserAccess(ctx context.Context, ar AccessRequest) (Policy, error)

	// EvaluateGroupAccess is used to evaluate if user has access to a group.
	// It returns an error and an empty policy if the user does not have access
	// otherwise it returns nil and the policy.
	EvaluateGroupAccess(ctx context.Context, ar AccessRequest) (Policy, error)

	// Update updates the policy type.
	// It overwrites the existing policy actions with the new policy actions.
	// It returns an error if the policy does not exist or the operation failed
	// otherwise it returns nil.
	Update(ctx context.Context, p Policy) error

	// RetrieveAll retrieves policies based on the given policy structure.
	// It returns an error with an empty policy page if the operation failed
	// otherwise it returns nil and the policy page.
	RetrieveAll(ctx context.Context, pm Page) (PolicyPage, error)

	// Delete deletes the policy for the given Policy Subject and Object combination.
	// It returns an error if the policy does not exist or the operation failed
	// otherwise it returns nil.
	Delete(ctx context.Context, p Policy) error
}

type Service interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed `action` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, ar AccessRequest) error

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	// AddPolicy adds a policy is added if:
	//
	//  1. The subject is admin
	//
	//  2. The subject has `g_add` action on the object or is the owner of the object.
	AddPolicy(ctx context.Context, token string, p Policy) error

	// UpdatePolicy updates policies based on the given policy structure.
	// UpdatePolicy updates a policy if:
	//
	//  1. The subject is admin.
	//
	//  2. The subject is the owner of the policy.
	UpdatePolicy(ctx context.Context, token string, p Policy) error

	// ListPolicies lists policies based on the given policy structure.
	ListPolicies(ctx context.Context, token string, pm Page) (PolicyPage, error)

	// DeletePolicy removes a policy.
	// DeletePolicy deletes a policy if:
	//
	//  1. The subject is admin.
	//
	//  2. The subject is the owner of the policy.
	DeletePolicy(ctx context.Context, token string, p Policy) error
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
		if ok := ValidateAction(p); !ok {
			return apiutil.ErrMalformedPolicyAct
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

// AddListAction adds list actions to the actions slice if c_ or g_ actions are present.
//
// 1. If c_<anything> actions are present, add c_list and g_list actions to the actions slice.
//
// 2. If g_<anything> actions are present, add g_list action to the actions slice.
func AddListAction(actions []string) []string {
	hasCAction := false
	hasGAction := false

	for _, action := range actions {
		if strings.HasPrefix(action, "c_") {
			hasCAction = true
		}
		if strings.HasPrefix(action, "g_") {
			hasGAction = true
		}
	}

	updatedActions := make([]string, 0)

	updatedActions = append(updatedActions, actions...)

	if hasCAction {
		updatedActions = append(updatedActions, "c_list", "g_list")
	}

	if hasGAction {
		updatedActions = append(updatedActions, "g_list")
	}

	return removeDuplicates(updatedActions)
}

func removeDuplicates(slice []string) []string {
	unique := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !unique[item] {
			unique[item] = true
			result = append(result, item)
		}
	}

	return result
}

// checkActions checks if the incoming actions are in the current actions.
func checkActions(currentActions, incomingActions []string) error {
	for _, action := range incomingActions {
		if !slices.Contains(currentActions, action) {
			return errors.ErrAuthorization
		}
	}
	return nil
}
