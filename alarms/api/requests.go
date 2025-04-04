// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"

	"github.com/absmach/magistrala/alarms"
	apiutil "github.com/absmach/supermq/api/http/util"
)

type createRuleReq struct {
	alarms.Rule `json:",inline"`
}

func (req createRuleReq) validate() error {
	if req.Rule.Name == "" {
		return apiutil.ErrMissingName
	}

	return nil
}

type entityReq struct {
	ID string
}

func (req entityReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type createAlarmReq struct {
	alarms.Alarm `json:",inline"`
}

func (req createAlarmReq) validate() error {
	if req.Alarm.RuleID == "" {
		return errors.New("missing rule id")
	}
	if req.Alarm.Message == "" {
		return errors.New("missing message")
	}

	return nil
}

type assignAlarmReq struct {
	alarms.Alarm `json:",inline"`
}

func (req assignAlarmReq) validate() error {
	if req.Alarm.ID == "" {
		return errors.New("missing alarm id")
	}
	if req.Alarm.AssigneeID == "" {
		return errors.New("missing assignee id")
	}

	return nil
}
