// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"maps"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
)

const (
	rulePrefix         = "rule."
	ruleCreate         = rulePrefix + "create"
	ruleList           = rulePrefix + "list"
	ruleView           = rulePrefix + "view"
	ruleUpdate         = rulePrefix + "update"
	ruleUpdateTags     = rulePrefix + "update_tags"
	ruleUpdateSchedule = rulePrefix + "update_schedule"
	ruleEnable         = rulePrefix + "enable"
	ruleDisable        = rulePrefix + "disable"
	ruleRemove         = rulePrefix + "remove"
)

var (
	_ events.Event = (*createRuleEvent)(nil)
	_ events.Event = (*listRuleEvent)(nil)
	_ events.Event = (*viewRuleEvent)(nil)
	_ events.Event = (*updateRuleEvent)(nil)
	_ events.Event = (*updateRuleTagsEvent)(nil)
	_ events.Event = (*updateRuleScheduleEvent)(nil)
	_ events.Event = (*enableRuleEvent)(nil)
	_ events.Event = (*disableRuleEvent)(nil)
	_ events.Event = (*removeRuleEvent)(nil)
)

type baseRuleEvent struct {
	session   authn.Session
	requestID string
}

func newBaseRuleEvent(session authn.Session, requestID string) baseRuleEvent {
	return baseRuleEvent{
		session:   session,
		requestID: requestID,
	}
}

func (bre baseRuleEvent) Encode() map[string]any {
	return map[string]any{
		"domain":      bre.session.DomainID,
		"user_id":     bre.session.UserID,
		"token_type":  bre.session.Type.String(),
		"super_admin": bre.session.SuperAdmin,
		"request_id":  bre.requestID,
	}
}

type createRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (cre createRuleEvent) Encode() (map[string]any, error) {
	val, err := cre.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, cre.baseRuleEvent.Encode())
	val["operation"] = ruleCreate
	return val, nil
}

type listRuleEvent struct {
	re.PageMeta
	baseRuleEvent
}

// Encode implements the events.Event interface for listRuleEvent.
func (lre listRuleEvent) Encode() (map[string]any, error) {
	val := lre.PageMeta.EventEncode()
	maps.Copy(val, lre.baseRuleEvent.Encode())
	val["operation"] = ruleList
	return val, nil
}

type updateRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

type viewRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (vre viewRuleEvent) Encode() (map[string]any, error) {
	val, err := vre.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, vre.baseRuleEvent.Encode())
	val["operation"] = ruleView
	return val, nil
}

func (ure updateRuleEvent) Encode() (map[string]any, error) {
	val, err := ure.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, ure.baseRuleEvent.Encode())
	val["operation"] = ruleUpdate
	return val, nil
}

type updateRuleTagsEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (urte updateRuleTagsEvent) Encode() (map[string]any, error) {
	val, err := urte.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, urte.baseRuleEvent.Encode())
	val["operation"] = ruleUpdateTags
	return val, nil
}

type updateRuleScheduleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (urse updateRuleScheduleEvent) Encode() (map[string]any, error) {
	val, err := urse.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, urse.baseRuleEvent.Encode())
	val["operation"] = ruleUpdateSchedule
	return val, nil
}

type disableRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (dre disableRuleEvent) Encode() (map[string]any, error) {
	val, err := dre.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, dre.baseRuleEvent.Encode())
	val["operation"] = ruleDisable
	return val, nil
}

type enableRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (ere enableRuleEvent) Encode() (map[string]any, error) {
	val, err := ere.rule.EventEncode()
	if err != nil {
		return map[string]any{}, err
	}
	maps.Copy(val, ere.baseRuleEvent.Encode())
	val["operation"] = ruleEnable
	return val, nil
}

type removeRuleEvent struct {
	id string
	baseRuleEvent
}

func (rre removeRuleEvent) Encode() (map[string]any, error) {
	val := rre.baseRuleEvent.Encode()
	val["id"] = rre.id
	val["operation"] = ruleRemove
	return val, nil
}
