// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
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
	created      bool
	deleted      bool
}

func (res alarmRes) Headers() map[string]string {
	switch {
	case res.created:
		return map[string]string{
			"Location": fmt.Sprintf("/%s/alarms/%s", res.DomainID, res.ID),
		}
	default:
		return map[string]string{}
	}
}

func (res alarmRes) Code() int {
	switch {
	case res.created:
		return http.StatusCreated
	case res.deleted:
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}

func (res alarmRes) Empty() bool {
	switch {
	case res.deleted:
		return true
	default:
		return false
	}
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
