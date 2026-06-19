// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
)

type PolicyEvaluator struct {
	client Authorizer
}

func NewPolicyEvaluator(client Authorizer) PolicyEvaluator {
	return PolicyEvaluator{client: client}
}

func (pe PolicyEvaluator) CheckPolicy(ctx context.Context, pr policies.Policy) error {
	res, err := pe.client.CheckAuthz(ctx, AuthzRequest{
		SubjectID:  policySubjectID(pr),
		Action:     policyAction(pr),
		ResourceID: policyResourceID(pr),
		ObjectKind: policyObjectKind(pr),
		ObjectID:   pr.Object,
		Context: map[string]any{
			"domain_id":          pr.Domain,
			"legacy_object_type": pr.ObjectType,
			"legacy_relation":    pr.Relation,
		},
	})
	if err != nil {
		return err
	}
	if !res.Allowed {
		return errors.ErrAuthorization
	}
	return nil
}

func policySubjectID(pr policies.Policy) string {
	if pr.Domain != "" {
		return strings.TrimPrefix(pr.Subject, pr.Domain+"_")
	}
	return pr.Subject
}

func policyAction(pr policies.Policy) string {
	if pr.Permission != "" {
		return pr.Permission
	}
	return pr.Relation
}

func policyObjectKind(pr policies.Policy) string {
	switch pr.ObjectType {
	case policies.DomainType:
		return atomObjectKindTenant
	case policies.PlatformType:
		return policies.PlatformType
	case policies.ClientType:
		return atomObjectKindEntity
	case policies.GroupType:
		return atomObjectKindGroup
	case policies.ChannelType:
		return atomObjectKindResource
	case policies.RulesType:
		return atomObjectKindResource
	case policies.ReportsType:
		return atomObjectKindResource
	case policies.AlarmsType:
		return atomObjectKindResource
	default:
		return pr.ObjectType
	}
}

func policyResourceID(pr policies.Policy) string {
	if pr.ObjectType == policies.DomainType || pr.ObjectType == policies.PlatformType {
		return ""
	}
	return pr.Object
}
