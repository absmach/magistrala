// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/mainflux/mainflux/pkg/errors"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

var _ tpolicies.Service = (*mainfluxPolicies)(nil)

type mainfluxPolicies struct {
	mu          sync.Mutex
	auth        upolicies.AuthServiceClient
	connections map[string]tpolicies.Policy
}

// NewPoliciesService returns Mainflux Things Policies service mock.
// Only methods used by SDK are mocked.
func NewPoliciesService(auth upolicies.AuthServiceClient) tpolicies.Service {
	return &mainfluxPolicies{
		auth:        auth,
		connections: make(map[string]tpolicies.Policy),
	}
}

func (svc *mainfluxPolicies) AddPolicy(ctx context.Context, token string, p tpolicies.Policy) (tpolicies.Policy, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if _, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token}); err != nil {
		return tpolicies.Policy{}, errors.ErrAuthentication
	}
	svc.connections[fmt.Sprintf("%s:%s", p.Subject, p.Object)] = p

	return p, nil
}

func (svc *mainfluxPolicies) DeletePolicy(ctx context.Context, token string, p tpolicies.Policy) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if _, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token}); err != nil {
		return errors.ErrAuthentication
	}

	for _, pol := range svc.connections {
		if pol.Subject == p.Subject && pol.Object == p.Object {
			delete(svc.connections, fmt.Sprintf("%s:%s", p.Subject, p.Object))
		}
	}
	return nil
}

func (svc *mainfluxPolicies) UpdatePolicy(context.Context, string, tpolicies.Policy) (tpolicies.Policy, error) {
	panic("not implemented")
}

func (svc *mainfluxPolicies) Authorize(context.Context, tpolicies.AccessRequest) (tpolicies.Policy, error) {
	panic("not implemented")
}

func (svc *mainfluxPolicies) ListPolicies(context.Context, string, tpolicies.Page) (tpolicies.PolicyPage, error) {
	panic("not implemented")
}
