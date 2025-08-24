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

// AllOperations is a list of all rule operations.
var AllOperations = [...]string{
	ruleCreate,
	ruleList,
	ruleView,
	ruleUpdate,
	ruleUpdateTags,
	ruleUpdateSchedule,
	ruleEnable,
	ruleDisable,
	ruleRemove,
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
	rule re.Rule
	baseRuleEvent
}

func (cre createRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleCreate,
	}
	rule, err := cre.rule.EventEncode()
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
		"operation": ruleList,
	}, nil
}

type updateRuleEvent struct {
	rule      re.Rule
	operation string
	baseRuleEvent
}

func (ure updateRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ure.operation,
	}

	rule, err := ure.rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, ure.baseRuleEvent.Encode())

	return val, nil
}

type viewRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (vre viewRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleView,
	}
	rule, err := vre.rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, vre.baseRuleEvent.Encode())
	return val, nil
}

type updateRuleTagsEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (urte updateRuleTagsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleUpdateTags,
	}
	rule, err := urte.rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, urte.baseRuleEvent.Encode())
	return val, nil
}

type updateRuleScheduleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (urse updateRuleScheduleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleUpdateSchedule,
	}
	rule, err := urse.rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, urse.baseRuleEvent.Encode())
	return val, nil
}

type disableRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (dre disableRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleDisable,
	}
	rule, err := dre.rule.EventEncode()
	if err != nil {
		return map[string]interface{}{}, err
	}
	maps.Copy(val, rule)
	maps.Copy(val, dre.baseRuleEvent.Encode())
	return val, nil
}

type enableRuleEvent struct {
	rule re.Rule
	baseRuleEvent
}

func (ere enableRuleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": ruleEnable,
	}
	rule, err := ere.rule.EventEncode()
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
		"operation": ruleRemove,
		"id":        rre.id,
	}
	maps.Copy(val, rre.baseRuleEvent.Encode())
	return val, nil
}
