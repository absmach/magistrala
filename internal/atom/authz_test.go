// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/stretchr/testify/assert"
)

type authzClient struct {
	req atom.AuthzRequest
	res atom.AuthzResponse
	err error
}

func (c *authzClient) CheckAuthz(_ context.Context, req atom.AuthzRequest) (atom.AuthzResponse, error) {
	c.req = req
	return c.res, c.err
}

func TestAuthorizeBuildsResourceRequest(t *testing.T) {
	client := &authzClient{res: atom.AuthzResponse{Allowed: true}}
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}

	err := atom.Authorize(context.Background(), client, session, "view", policies.RulesType, "rule-1", atom.KindRule)

	assert.NoError(t, err)
	assert.Equal(t, atom.AuthzRequest{
		SubjectID:  "user-1",
		Action:     "read",
		ResourceID: "rule-1",
		ObjectKind: "resource",
		ObjectID:   "rule-1",
		Context: map[string]any{
			"domain_id":          "domain-1",
			"legacy_object_type": policies.RulesType,
		},
	}, client.req)
}

func TestAuthorizeBuildsTenantRequest(t *testing.T) {
	client := &authzClient{res: atom.AuthzResponse{Allowed: true}}
	session := authn.Session{UserID: "user-1", DomainID: "domain-1"}

	err := atom.Authorize(context.Background(), client, session, "create", policies.DomainType, "domain-1", atom.KindRule)

	assert.NoError(t, err)
	assert.Equal(t, "tenant", client.req.ObjectKind)
	assert.Equal(t, "domain-1", client.req.ObjectID)
	assert.Empty(t, client.req.ResourceID)
}

func TestAuthorizeDenied(t *testing.T) {
	client := &authzClient{res: atom.AuthzResponse{Allowed: false}}

	err := atom.Authorize(context.Background(), client, authn.Session{UserID: "user-1"}, "view", policies.RulesType, "rule-1", atom.KindRule)

	assert.True(t, errors.Contains(err, errors.ErrAuthorization))
}
