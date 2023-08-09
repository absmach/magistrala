// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

const (
	// Messaging policy actions.
	ReadAction  = "m_read"
	WriteAction = "m_write"

	// Add policy actions to check if the client is admin.
	addPolicyAction = "g_add"

	sharePolicyAction = "c_share"

	// Entity types.
	ClientEntityType = "client"
	GroupEntityType  = "group"
	ThingEntityType  = "thing"

	thingsObjectKey = "things"
)

// ErrInvalidEntityType indicates that the entity type is invalid.
var ErrInvalidEntityType = errors.New("invalid entity type")

type service struct {
	auth        upolicies.AuthServiceClient
	policies    Repository
	policyCache Cache
	idProvider  mainflux.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(auth upolicies.AuthServiceClient, p Repository, ccache Cache, idp mainflux.IDProvider) Service {
	return service{
		auth:        auth,
		policies:    p,
		policyCache: ccache,
		idProvider:  idp,
	}
}

func (svc service) Authorize(ctx context.Context, ar AccessRequest) (Policy, error) {
	// Fetch from cache first.
	policy := Policy{
		Subject: ar.Subject,
		Object:  ar.Object,
	}
	policy, err := svc.policyCache.Get(ctx, policy)
	if err == nil {
		for _, action := range policy.Actions {
			if action == ar.Action {
				return policy, nil
			}
		}
		return Policy{}, errors.ErrAuthorization
	}
	if !errors.Contains(err, errors.ErrNotFound) {
		return Policy{}, err
	}

	// Fetch from repo as a fallback if not found in cache.
	switch ar.Entity {
	case GroupEntityType:
		policy, err = svc.policies.EvaluateGroupAccess(ctx, ar)
		if err != nil {
			return Policy{}, err
		}

	case ClientEntityType:
		policy, err = svc.policies.EvaluateThingAccess(ctx, ar)
		if err != nil {
			return Policy{}, err
		}

	case ThingEntityType:
		policy, err := svc.policies.EvaluateMessagingAccess(ctx, ar)
		if err != nil {
			return Policy{}, err
		}
		// Replace Subject since AccessRequest Subject is Thing Key,
		// and Policy subject is Thing ID.
		policy.Subject = ar.Subject
		if err := svc.policyCache.Put(ctx, policy); err != nil {
			return policy, err
		}

	default:
		return Policy{}, ErrInvalidEntityType
	}

	return policy, nil
}

// AddPolicy adds a policy is added if:
//
//  1. The client is admin
//
//  2. The client has `g_add` action on the object or is the owner of the object.
func (svc service) AddPolicy(ctx context.Context, token string, external bool, p Policy) (Policy, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return Policy{}, err
	}
	if err := p.validate(); err != nil {
		return Policy{}, err
	}
	if external {
		p.Actions = upolicies.AddListAction(p.Actions)
	}

	p.OwnerID = userID
	p.CreatedAt = time.Now()

	// Incase the policy exists, use these for update.
	p.UpdatedAt = time.Now()
	p.UpdatedBy = userID

	if err := svc.policyCache.Remove(ctx, p); err != nil {
		return Policy{}, err
	}

	// If the client is admin, add the policy
	if err := svc.checkAdmin(ctx, userID); err == nil {
		return svc.policies.Save(ctx, p)
	}

	// If the client has `g_add` action on the object or is the owner of the object, add the policy
	areq := AccessRequest{Subject: userID, Object: p.Object, Action: "g_add"}
	if pol, err := svc.policies.EvaluateGroupAccess(ctx, areq); err == nil {
		if err := svc.checkSubject(ctx, userID, external, p); err != nil {
			return Policy{}, err
		}

		// the client has `g_add` action on the object
		if len(pol.Actions) > 0 {
			if err := checkActions(pol.Actions, p.Actions); err != nil {
				return Policy{}, err
			}
		}

		// the client is the owner of the object
		return svc.policies.Save(ctx, p)
	}

	return Policy{}, errors.ErrAuthorization
}

// checkSubject is used to check if the subject is valid.
//
// 1. If the subject is a thing (external is false), check the following:
//   - The user is the owner of the thing
//   - The user has `c_share` action on the thing
//
// 2. If the subject is a user (external is true), check the following:
//   - The user is the owner of the user
//   - The user has `c_share` action on the user
func (svc service) checkSubject(ctx context.Context, userID string, external bool, p Policy) error {
	if external {
		return svc.usersAuthorize(ctx, userID, p.Subject, sharePolicyAction, ClientEntityType)
	}

	ar := AccessRequest{Subject: userID, Object: p.Subject, Action: sharePolicyAction}
	if _, err := svc.policies.EvaluateThingAccess(ctx, ar); err != nil {
		return err
	}

	return nil
}

func (svc service) UpdatePolicy(ctx context.Context, token string, p Policy) (Policy, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return Policy{}, err
	}

	if err := p.validate(); err != nil {
		return Policy{}, err
	}
	if err := svc.checkPolicy(ctx, userID, p); err != nil {
		return Policy{}, err
	}
	p.UpdatedAt = time.Now()
	p.UpdatedBy = userID

	if err := svc.policyCache.Remove(ctx, p); err != nil {
		return Policy{}, err
	}

	return svc.policies.Update(ctx, p)
}

func (svc service) ListPolicies(ctx context.Context, token string, pm Page) (PolicyPage, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return PolicyPage{}, err
	}
	if err := pm.Validate(); err != nil {
		return PolicyPage{}, err
	}
	// If the user is admin, return all policies.
	if err := svc.checkAdmin(ctx, userID); err == nil {
		return svc.policies.Retrieve(ctx, pm)
	}

	// If the user is not admin, return only the policies that they created.
	pm.OwnerID = userID

	return svc.policies.Retrieve(ctx, pm)
}

func (svc service) DeletePolicy(ctx context.Context, token string, p Policy) error {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.checkPolicy(ctx, userID, p); err != nil {
		return err
	}

	if err := svc.policyCache.Remove(ctx, p); err != nil {
		return err
	}
	return svc.policies.Delete(ctx, p)
}

// checkPolicy checks for the following:
//
//  1. Check if the client is admin
//  2. Check if the client is the owner of the policy
func (svc service) checkPolicy(ctx context.Context, clientID string, p Policy) error {
	if err := svc.checkAdmin(ctx, clientID); err == nil {
		return nil
	}

	pm := Page{Subject: p.Subject, Object: p.Object, OwnerID: clientID, Offset: 0, Limit: 1}
	page, err := svc.policies.Retrieve(ctx, pm)
	if err != nil {
		return err
	}
	if len(page.Policies) == 1 && page.Policies[0].OwnerID == clientID {
		return nil
	}

	return errors.ErrAuthorization
}

func (svc service) identify(ctx context.Context, token string) (string, error) {
	req := &upolicies.IdentifyReq{Token: token}
	res, err := svc.auth.Identify(ctx, req)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	return res.GetId(), nil
}

func (svc service) checkAdmin(ctx context.Context, id string) error {
	// For checking admin rights policy object, action and entity type are not important.
	return svc.usersAuthorize(ctx, id, thingsObjectKey, addPolicyAction, GroupEntityType)
}

func (svc service) usersAuthorize(ctx context.Context, subject, object, action, entity string) error {
	req := &upolicies.AuthorizeReq{
		Subject:    subject,
		Object:     object,
		Action:     action,
		EntityType: entity,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}
