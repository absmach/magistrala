// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/stretchr/testify/assert"
)

func TestPolicyEvaluatorCheckPolicy(t *testing.T) {
	client := &authzClient{res: atom.AuthzResponse{Allowed: true}}
	evaluator := atom.NewPolicyEvaluator(client)

	err := evaluator.CheckPolicy(context.Background(), policies.Policy{
		Domain:     "domain-1",
		Subject:    "domain-1_user-1",
		Permission: policies.ViewPermission,
		ObjectType: policies.RulesType,
		Object:     "rule-1",
	})

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
			"legacy_relation":    "",
		},
	}, client.req)
}

func TestPolicyEvaluatorDenied(t *testing.T) {
	client := &authzClient{res: atom.AuthzResponse{Allowed: false}}
	evaluator := atom.NewPolicyEvaluator(client)

	err := evaluator.CheckPolicy(context.Background(), policies.Policy{
		Subject:    "user-1",
		Permission: policies.AdminPermission,
		ObjectType: policies.PlatformType,
		Object:     policies.MagistralaObject,
	})

	assert.True(t, errors.Contains(err, errors.ErrAuthorization))
}
