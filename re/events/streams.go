// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/messaging"
	rmEvents "github.com/absmach/supermq/pkg/roles/rolemanager/events"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	supermqPrefix        = "supermq."
	CreateStream         = supermqPrefix + ruleCreate
	ListStream           = supermqPrefix + ruleList
	ViewStream           = supermqPrefix + ruleView
	UpdateStream         = supermqPrefix + ruleUpdate
	UpdateTagsStream     = supermqPrefix + ruleUpdateTags
	UpdateScheduleStream = supermqPrefix + ruleUpdateSchedule
	EnableStream         = supermqPrefix + ruleEnable
	DisableStream        = supermqPrefix + ruleDisable
	RemoveStream         = supermqPrefix + ruleRemove
)

var _ re.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc re.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around rules service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc re.Service, url string) (re.Service, error) {
	publisher, err := store.NewPublisher(ctx, url)
	if err != nil {
		return nil, err
	}

	res := rmEvents.NewRoleManagerEventStore("alarms", supermqPrefix, svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	rule, err := es.svc.AddRule(ctx, session, r)
	if err != nil {
		return rule, err
	}
	event := createRuleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, CreateStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	page, err := es.svc.ListRules(ctx, session, pm)
	if err != nil {
		return page, err
	}
	event := listRuleEvent{
		PageMeta:      pm,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, ListStream, event); err != nil {
		return page, err
	}
	return page, nil
}

func (es *eventStore) ViewRule(ctx context.Context, session authn.Session, id string, withRoles bool) (re.Rule, error) {
	rule, err := es.svc.ViewRule(ctx, session, id, withRoles)
	if err != nil {
		return rule, err
	}
	event := viewRuleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, ViewStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	rule, err := es.svc.UpdateRule(ctx, session, r)
	if err != nil {
		return rule, err
	}
	event := updateRuleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, UpdateStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	rule, err := es.svc.UpdateRuleTags(ctx, session, r)
	if err != nil {
		return rule, err
	}
	event := updateRuleTagsEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, UpdateTagsStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	rule, err := es.svc.UpdateRuleSchedule(ctx, session, r)
	if err != nil {
		return rule, err
	}
	event := updateRuleScheduleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, UpdateScheduleStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	err := es.svc.RemoveRule(ctx, session, id)
	if err != nil {
		return err
	}
	event := removeRuleEvent{
		id:            id,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, RemoveStream, event); err != nil {
		return err
	}
	return nil
}

func (es *eventStore) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	rule, err := es.svc.EnableRule(ctx, session, id)
	if err != nil {
		return rule, err
	}
	event := enableRuleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, EnableStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	rule, err := es.svc.DisableRule(ctx, session, id)
	if err != nil {
		return rule, err
	}
	event := disableRuleEvent{
		rule:          rule,
		baseRuleEvent: newBaseRuleEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, DisableStream, event); err != nil {
		return rule, err
	}
	return rule, nil
}

func (es *eventStore) StartScheduler(ctx context.Context) error {
	return es.svc.StartScheduler(ctx)
}

func (es *eventStore) Handle(msg *messaging.Message) error {
	return es.svc.Handle(msg)
}

func (es *eventStore) Cancel() error {
	return es.svc.Cancel()
}
