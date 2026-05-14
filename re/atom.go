// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
)

type atomService struct {
	Service
	projector atom.Projector
}

func WithAtom(svc Service, projector atom.Projector) Service {
	if projector == nil {
		return svc
	}
	return atomService{Service: svc, projector: projector}
}

func (svc atomService) AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	rule, err := svc.Service.AddRule(ctx, session, r)
	if err != nil {
		return rule, err
	}
	if err := svc.projector.UpsertResource(ctx, ruleProjection(rule)); err != nil {
		return rule, nil
	}
	return rule, nil
}

func (svc atomService) UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	rule, err := svc.Service.UpdateRule(ctx, session, r)
	return svc.upsertAfterRuleChange(ctx, rule, err)
}

func (svc atomService) UpdateRuleTags(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	rule, err := svc.Service.UpdateRuleTags(ctx, session, r)
	return svc.upsertAfterRuleChange(ctx, rule, err)
}

func (svc atomService) UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	rule, err := svc.Service.UpdateRuleSchedule(ctx, session, r)
	return svc.upsertAfterRuleChange(ctx, rule, err)
}

func (svc atomService) EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	rule, err := svc.Service.EnableRule(ctx, session, id)
	return svc.upsertAfterRuleChange(ctx, rule, err)
}

func (svc atomService) DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	rule, err := svc.Service.DisableRule(ctx, session, id)
	return svc.upsertAfterRuleChange(ctx, rule, err)
}

func (svc atomService) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.RemoveRule(ctx, session, id); err != nil {
		return err
	}
	_ = svc.projector.DeleteResource(ctx, id)
	return nil
}

func (svc atomService) upsertAfterRuleChange(ctx context.Context, rule Rule, err error) (Rule, error) {
	if err != nil {
		return rule, err
	}
	if err := svc.projector.UpsertResource(ctx, ruleProjection(rule)); err != nil {
		return rule, nil
	}
	return rule, nil
}

func ruleProjection(r Rule) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID:        r.ID,
		Kind:      atom.KindRule,
		Name:      r.Name,
		TenantID:  r.DomainID,
		OwnerID:   r.CreatedBy,
		Status:    r.Status.String(),
		Tags:      r.Tags,
		Metadata:  map[string]any(r.Metadata),
		CreatedBy: r.CreatedBy,
		UpdatedBy: r.UpdatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	})
	res.Attributes["input_channel"] = r.InputChannel
	res.Attributes["input_topic"] = r.InputTopic
	res.Attributes["scheduled_at"] = r.Schedule.Time
	return res
}
