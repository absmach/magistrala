// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/callout"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var _ re.Service = (*calloutMiddleware)(nil)

type calloutMiddleware struct {
	svc     re.Service
	callout callout.Callout
}

const entityType = "rule"

func NewCallout(svc re.Service, callout callout.Callout) (re.Service, error) {
	return &calloutMiddleware{
		svc:     svc,
		callout: callout,
	}, nil
}

func (cm *calloutMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	params := map[string]any{
		"entities": r,
		"count":    1,
	}

	if err := cm.callOut(ctx, session, re.OpAddRuleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.AddRule(ctx, session, r)
}

func (cm *calloutMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, re.OpViewRuleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.ViewRule(ctx, session, id)
}

func (cm *calloutMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := cm.callOut(ctx, session, re.OpUpdateRuleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.UpdateRule(ctx, session, r)
}

func (cm *calloutMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := cm.callOut(ctx, session, re.OpUpdateRuleTagsStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.UpdateRuleTags(ctx, session, r)
}

func (cm *calloutMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := cm.callOut(ctx, session, re.OpUpdateRuleScheduleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.UpdateRuleSchedule(ctx, session, r)
}

func (cm *calloutMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	params := map[string]any{
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, re.OpListRulesStr, params); err != nil {
		return re.Page{}, err
	}

	return cm.svc.ListRules(ctx, session, pm)
}

func (cm *calloutMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, re.OpRemoveRuleStr, params); err != nil {
		return err
	}

	return cm.svc.RemoveRule(ctx, session, id)
}

func (cm *calloutMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, re.OpEnableRuleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.EnableRule(ctx, session, id)
}

func (cm *calloutMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, re.OpDisableRuleStr, params); err != nil {
		return re.Rule{}, err
	}

	return cm.svc.DisableRule(ctx, session, id)
}

func (cm *calloutMiddleware) StartScheduler(ctx context.Context) error {
	return cm.svc.StartScheduler(ctx)
}

func (cm *calloutMiddleware) Handle(msg *messaging.Message) error {
	return cm.svc.Handle(msg)
}

func (cm *calloutMiddleware) Cancel() error {
	return cm.svc.Cancel()
}

func (cm *calloutMiddleware) callOut(ctx context.Context, session authn.Session, op string, pld map[string]any) error {
	var entityID string
	if id, ok := pld["entity_id"].(string); ok {
		entityID = id
	}

	req := callout.Request{
		BaseRequest: callout.BaseRequest{
			Operation:  op,
			EntityType: entityType,
			EntityID:   entityID,
			CallerID:   session.UserID,
			CallerType: policies.UserType,
			DomainID:   session.DomainID,
			Time:       time.Now().UTC(),
		},
		Payload: pld,
	}

	if err := cm.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
