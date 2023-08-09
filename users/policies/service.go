// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/jwt"
)

const AccessToken = "access"

// ErrInvalidEntityType indicates that the entity type is invalid.
var ErrInvalidEntityType = errors.New("invalid entity type")

type service struct {
	policies   Repository
	idProvider mainflux.IDProvider
	tokens     jwt.Repository
}

// NewService returns a new Policies service implementation.
func NewService(p Repository, t jwt.Repository, idp mainflux.IDProvider) Service {
	return service{
		policies:   p,
		tokens:     t,
		idProvider: idp,
	}
}

func (svc service) Authorize(ctx context.Context, ar AccessRequest) error {
	if err := svc.policies.CheckAdmin(ctx, ar.Subject); err == nil {
		return nil
	}
	switch ar.Entity {
	case "client":
		if _, err := svc.policies.EvaluateUserAccess(ctx, ar); err != nil {
			return err
		}
	case "group":
		if _, err := svc.policies.EvaluateGroupAccess(ctx, ar); err != nil {
			return err
		}
	default:
		return ErrInvalidEntityType
	}
	return nil
}

// AddPolicy adds a policy is added if:
//
//  1. The client is admin
//
//  2. The client has `g_add` action on the object or is the owner of the object.
func (svc service) AddPolicy(ctx context.Context, token string, p Policy) error {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	p.Actions = AddListAction(p.Actions)

	p.OwnerID = id
	p.CreatedAt = time.Now()

	// incase the policy exists, use these for update.
	p.UpdatedAt = time.Now()
	p.UpdatedBy = id

	// check if the client is admin
	if err = svc.policies.CheckAdmin(ctx, id); err == nil {
		return svc.policies.Save(ctx, p)
	}

	// check if the client has `g_add` action on the object or is the owner of the object
	areq := AccessRequest{Subject: id, Object: p.Object, Action: "g_add", Entity: "group"}
	if pol, err := svc.policies.EvaluateGroupAccess(ctx, areq); err == nil {
		// the client has `g_add` action on the object
		if len(pol.Actions) > 0 {
			if err := checkActions(pol.Actions, p.Actions); err != nil {
				return err
			}
		}

		// the client is the owner of the object
		return svc.policies.Save(ctx, p)
	}

	return errors.ErrAuthorization
}

func (svc service) UpdatePolicy(ctx context.Context, token string, p Policy) error {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err := svc.checkPolicy(ctx, id, p); err != nil {
		return err
	}
	p.UpdatedAt = time.Now()
	p.UpdatedBy = id

	return svc.policies.Update(ctx, p)
}

func (svc service) ListPolicies(ctx context.Context, token string, pm Page) (PolicyPage, error) {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return PolicyPage{}, err
	}
	if err := pm.Validate(); err != nil {
		return PolicyPage{}, err
	}
	// If the user is admin, return all policies
	if err := svc.policies.CheckAdmin(ctx, id); err == nil {
		return svc.policies.RetrieveAll(ctx, pm)
	}

	// If the user is not admin, return only the policies that they created
	pm.OwnerID = id

	return svc.policies.RetrieveAll(ctx, pm)
}

func (svc service) DeletePolicy(ctx context.Context, token string, p Policy) error {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.checkPolicy(ctx, id, p); err != nil {
		return err
	}
	return svc.policies.Delete(ctx, p)
}

// checkPolicy checks for the following:
//
//  1. Check if the client is admin
//  2. Check if the client is the owner of the policy
func (svc service) checkPolicy(ctx context.Context, clientID string, p Policy) error {
	// Check if the client is admin
	if err := svc.policies.CheckAdmin(ctx, clientID); err == nil {
		return nil
	}

	// Check if the client is the owner of the policy
	pm := Page{Subject: p.Subject, Object: p.Object, OwnerID: clientID, Offset: 0, Limit: 1}
	page, err := svc.policies.RetrieveAll(ctx, pm)
	if err != nil {
		return err
	}
	if len(page.Policies) == 1 && page.Policies[0].OwnerID == clientID {
		return nil
	}

	return errors.ErrAuthorization
}

// identify returns the client ID associated with the provided token.
func (svc service) identify(ctx context.Context, token string) (string, error) {
	claims, err := svc.tokens.Parse(ctx, token)
	if err != nil {
		return "", err
	}
	if claims.Type != AccessToken {
		return "", errors.ErrAuthentication
	}

	return claims.ClientID, nil
}
