// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq"
)

var (
	_ supermq.Response = (*alarmRes)(nil)
	_ supermq.Response = (*alarmsPageRes)(nil)
)

type alarmRes struct {
	alarms.Alarm `json:",inline"`
}

func (res alarmRes) Headers() map[string]string {
	return map[string]string{}
}

func (res alarmRes) Code() int {
	return http.StatusOK
}

func (res alarmRes) Empty() bool {
	return false
}

type alarmsPageRes struct {
	alarms.AlarmsPage `json:",inline"`
}

func (res alarmsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res alarmsPageRes) Code() int {
	return http.StatusOK
}

func (res alarmsPageRes) Empty() bool {
	return false
}
