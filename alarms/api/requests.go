// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"

	"github.com/absmach/magistrala/alarms"
	sapi "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
)

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

type listAlarmsReq struct {
	alarms.PageMetadata
}

func (req listAlarmsReq) validate() error {
	if req.Limit > sapi.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}
