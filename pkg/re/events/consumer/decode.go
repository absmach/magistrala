// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/roles"
	rconsumer "github.com/absmach/magistrala/pkg/roles/rolemanager/events/consumer"
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re"
)

var (
	errDecodeAddRuleEvent            = errors.New("failed to decode rule add event")
	errDecodeUpdateRuleEvent         = errors.New("failed to decode rule update event")
	errDecodeUpdateRuleTagsEvent     = errors.New("failed to decode rule update tags event")
	errDecodeUpdateRuleScheduleEvent = errors.New("failed to decode rule update schedule event")
	errDecodeEnableRuleEvent         = errors.New("failed to decode rule enable event")
	errDecodeDisableRuleEvent        = errors.New("failed to decode rule disable event")
	errDecodeRemoveRuleEvent         = errors.New("failed to decode rule remove event")

	errID             = errors.New("missing or invalid 'id'")
	errName           = errors.New("missing or invalid 'name'")
	errTags           = errors.New("invalid 'tags'")
	errStatus         = errors.New("missing or invalid 'status'")
	errConvertStatus  = errors.New("failed to convert status")
	errCreatedBy      = errors.New("missing or invalid 'created_by'")
	errCreatedAt      = errors.New("failed to parse 'created_at' time")
	errUpdatedAt      = errors.New("failed to parse 'updated_at' time")
	errDecodeLogic    = errors.New("failed to decode 'logic'")
	errDecodeSchedule = errors.New("failed to decode 'schedule'")
)

// ToRule decodes a map[string]any event payload into a re.Rule.
func ToRule(data map[string]any) (re.Rule, error) {
	var r re.Rule

	id, ok := data["id"].(string)
	if !ok {
		return re.Rule{}, errID
	}
	r.ID = id

	name, ok := data["name"].(string)
	if !ok {
		return re.Rule{}, errName
	}
	r.Name = name

	stat, ok := data["status"].(string)
	if !ok {
		return re.Rule{}, errStatus
	}
	st, err := re.ToStatus(stat)
	if err != nil {
		return re.Rule{}, errors.Wrap(errConvertStatus, err)
	}
	r.Status = st

	cby, ok := data["created_by"].(string)
	if !ok {
		return re.Rule{}, errCreatedBy
	}
	r.CreatedBy = cby

	cat, ok := data["created_at"].(string)
	if !ok {
		return re.Rule{}, errCreatedAt
	}
	ct, err := time.Parse(re.TimeLayout, cat)
	if err != nil {
		return re.Rule{}, errors.Wrap(errCreatedAt, err)
	}
	r.CreatedAt = ct

	if domain, ok := data["domain"].(string); ok {
		r.DomainID = domain
	}

	if itags, ok := data["tags"].([]any); ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return re.Rule{}, errors.Wrap(errTags, err)
		}
		r.Tags = tags
	}

	if meta, ok := data["metadata"].(map[string]any); ok {
		r.Metadata = meta
	}

	if uby, ok := data["updated_by"].(string); ok {
		r.UpdatedBy = uby
	}

	if uat, ok := data["updated_at"].(string); ok {
		ut, err := time.Parse(re.TimeLayout, uat)
		if err != nil {
			return re.Rule{}, errors.Wrap(errUpdatedAt, err)
		}
		r.UpdatedAt = ut
	}

	if ic, ok := data["input_channel"].(string); ok {
		r.InputChannel = ic
	}

	if it, ok := data["input_topic"].(string); ok {
		r.InputTopic = it
	}

	if rawLogic, ok := data["logic"].(map[string]any); ok {
		b, err := json.Marshal(rawLogic)
		if err != nil {
			return re.Rule{}, errors.Wrap(errDecodeLogic, err)
		}
		if err := json.Unmarshal(b, &r.Logic); err != nil {
			return re.Rule{}, errors.Wrap(errDecodeLogic, err)
		}
	}

	if rawSched, ok := data["schedule"].(map[string]any); ok {
		b, err := json.Marshal(rawSched)
		if err != nil {
			return re.Rule{}, errors.Wrap(errDecodeSchedule, err)
		}
		var sched schedule.Schedule
		if err := json.Unmarshal(b, &sched); err != nil {
			return re.Rule{}, errors.Wrap(errDecodeSchedule, err)
		}
		r.Schedule = sched
	}

	return r, nil
}

func decodeAddRuleEvent(data map[string]any) (re.Rule, []roles.RoleProvision, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, nil, errors.Wrap(errDecodeAddRuleEvent, err)
	}

	var rps []roles.RoleProvision
	if irps, ok := data["roles_provisioned"].([]any); ok {
		rps, err = rconsumer.ToRoleProvisions(irps)
		if err != nil {
			return re.Rule{}, nil, errors.Wrap(errDecodeAddRuleEvent, err)
		}
	}

	return r, rps, nil
}

func decodeUpdateRuleEvent(data map[string]any) (re.Rule, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, errors.Wrap(errDecodeUpdateRuleEvent, err)
	}
	return r, nil
}

func decodeUpdateRuleTagsEvent(data map[string]any) (re.Rule, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, errors.Wrap(errDecodeUpdateRuleTagsEvent, err)
	}
	return r, nil
}

func decodeUpdateRuleScheduleEvent(data map[string]any) (re.Rule, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, errors.Wrap(errDecodeUpdateRuleScheduleEvent, err)
	}
	return r, nil
}

func decodeEnableRuleEvent(data map[string]any) (re.Rule, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, errors.Wrap(errDecodeEnableRuleEvent, err)
	}
	return r, nil
}

func decodeDisableRuleEvent(data map[string]any) (re.Rule, error) {
	r, err := ToRule(data)
	if err != nil {
		return re.Rule{}, errors.Wrap(errDecodeDisableRuleEvent, err)
	}
	return r, nil
}

func decodeRemoveRuleEvent(data map[string]any) (string, error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", errors.Wrap(errDecodeRemoveRuleEvent, errID)
	}
	return id, nil
}
