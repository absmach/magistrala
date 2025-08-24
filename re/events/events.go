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
	RuleCreate         = rulePrefix + "create"
	RuleList           = rulePrefix + "list"
	RuleView           = rulePrefix + "view"
	RuleUpdate         = rulePrefix + "update"
	RuleUpdateTags     = rulePrefix + "update_tags"
	RuleUpdateSchedule = rulePrefix + "update_schedule"
	RuleEnable         = rulePrefix + "enable"
	RuleDisable        = rulePrefix + "disable"
	RuleRemove         = rulePrefix + "remove"
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

// AllOperations is a list of all rule operations.
var AllOperations = [...]string{
	RuleCreate,
	RuleList,
	RuleView,
	RuleUpdate,
	RuleUpdateTags,
	RuleUpdateSchedule,
	RuleEnable,
	RuleDisable,
	RuleRemove,
}

type baseRuleEvent struct {
	authn.Session
	requestID string
}

func newBaseRuleEvent(session authn.Session, requestID string) baseRuleEvent {
	return baseRuleEvent{
		Session:   session,
		requestID: requestID,
	}
}

func (bre baseRuleEvent) Encode() map[string]interface{} {
	return map[string]interface{}{
		"domain":      bre.Session.DomainID,
		"user_id":     bre.Session.UserID,
		"token_type":  bre.Session.Type.String(),
		"super_admin": bre.SuperAdmin,
		"request_id":  bre.requestID,
	}
}

type createRuleEvent struct {
	re.Rule
	baseRuleEvent
}

func (cre createRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleCreate,
	}
	rule, err := cre.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, cre.baseRuleEvent.Encode())

	return val, nil
}

type listRuleEvent struct {
	re.PageMeta
	authn.Session
	requestID string
}

// Encode implements the events.Event interface for listRuleEvent.
func (lre listRuleEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": RuleList,
	}, nil
}

type updateRuleEvent struct {
	re.Rule
	operation string
	baseRuleEvent
}

func (ure updateRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ure.operation,
	}

	rule, err := ure.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, ure.baseRuleEvent.Encode())

	return val, nil
}

type viewRuleEvent struct {
	re.Rule
	baseRuleEvent
}

func (vre viewRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleView,
	}
	rule, err := vre.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, vre.baseRuleEvent.Encode())
	return val, nil
}

type updateRuleTagsEvent struct {
	re.Rule
	baseRuleEvent
}

func (urte updateRuleTagsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleUpdateTags,
	}
	rule, err := urte.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, urte.baseRuleEvent.Encode())
	return val, nil
}

type updateRuleScheduleEvent struct {
	re.Rule
	baseRuleEvent
}

func (urse updateRuleScheduleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleUpdateSchedule,
	}
	rule, err := urse.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, urse.baseRuleEvent.Encode())
	return val, nil
}

type disableRuleEvent struct {
	re.Rule
	baseRuleEvent
}

func (dre disableRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleDisable,
	}
	rule, err := dre.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, dre.baseRuleEvent.Encode())
	return val, nil
}

type enableRuleEvent struct {
	re.Rule
	baseRuleEvent
}

func (ere enableRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleEnable,
	}
	rule, err := ere.Rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, ere.baseRuleEvent.Encode())
	return val, nil
}

type removeRuleEvent struct {
	id string
	baseRuleEvent
}

func (rre removeRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": RuleRemove,
		"id":        rre.id,
	}
	maps.Copy(val, rre.baseRuleEvent.Encode())
	return val, nil
}
