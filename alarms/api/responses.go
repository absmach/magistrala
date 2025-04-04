// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq"
)

var (
	_ supermq.Response = (*ruleRes)(nil)
	_ supermq.Response = (*rulesPageRes)(nil)
	_ supermq.Response = (*alarmRes)(nil)
	_ supermq.Response = (*alarmsPageRes)(nil)
)

type ruleRes struct {
	alarms.Rule `json:",inline"`
}

func (res ruleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res ruleRes) Code() int {
	return http.StatusOK
}

func (res ruleRes) Empty() bool {
	return false
}

type rulesPageRes struct {
	alarms.RulesPage `json:",inline"`
}

func (res rulesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res rulesPageRes) Code() int {
	return http.StatusOK
}

func (res rulesPageRes) Empty() bool {
	return false
}

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
