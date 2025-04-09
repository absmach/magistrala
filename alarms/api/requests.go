// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"

	"github.com/absmach/magistrala/alarms"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
)

type alarmReq struct {
	alarms.Alarm `json:",inline"`
}

func (req alarmReq) validate() error {
	if req.Alarm.ID == "" {
		return errors.New("missing alarm id")
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
	if req.Alarm.Measurement == "" {
		return errors.New("missing measurement")
	}
	if req.Alarm.Value == "" {
		return errors.New("missing measurement value")
	}
	if req.Alarm.Unit == "" {
		return errors.New("missing measurement unit")
	}
	if req.Alarm.Cause == "" {
		return errors.New("missing measurement cause")
	}

	return nil
}

type listAlarmsReq struct {
	alarms.PageMetadata
}

func (req listAlarmsReq) validate() error {
	if req.Limit > api.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}
