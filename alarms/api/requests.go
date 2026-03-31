// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"

	"github.com/absmach/supermq/alarms"
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

type updateAlarmReq struct {
	alarms.Alarm `json:",inline"`
}

func (req updateAlarmReq) validate() error {
	if req.Alarm.ID == "" {
		return errors.New("missing alarm id")
	}
	if req.Alarm.AssigneeID == "" && req.Alarm.AcknowledgedBy == "" && req.Alarm.ResolvedBy == "" && len(req.Alarm.Metadata) == 0 {
		return errors.New("at least one of assignee_id, acknowledged_by, resolved_by, or metadata must be set")
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

	if req.Order != "" && req.Order != api.UpdatedAtOrder && req.Order != api.CreatedAtOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.Dir != api.AscDir && req.Dir != api.DescDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}
