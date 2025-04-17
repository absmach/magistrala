// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq"
)

var (
	_ supermq.Response = (*viewRuleRes)(nil)
	_ supermq.Response = (*addRuleRes)(nil)
	_ supermq.Response = (*updateRuleStatusRes)(nil)
	_ supermq.Response = (*rulesPageRes)(nil)
	_ supermq.Response = (*updateRuleRes)(nil)
	_ supermq.Response = (*deleteRuleRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type addRuleRes struct {
	re.Rule
	created bool
}

func (res addRuleRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res addRuleRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/rules/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res addRuleRes) Empty() bool {
	return false
}

type updateRuleRes struct {
	re.Rule `json:",inline"`
}

func (res updateRuleRes) Code() int {
	return http.StatusOK
}

func (res updateRuleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateRuleRes) Empty() bool {
	return false
}

type viewRuleRes struct {
	re.Rule `json:",inline"`
}

func (res viewRuleRes) Code() int {
	return http.StatusOK
}

func (res viewRuleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewRuleRes) Empty() bool {
	return false
}

type rulesPageRes struct {
	re.Page `json:",inline"`
}

func (res rulesPageRes) Code() int {
	return http.StatusOK
}

func (res rulesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res rulesPageRes) Empty() bool {
	return false
}

type updateRuleStatusRes struct {
	re.Rule `json:",inline"`
}

func (res updateRuleStatusRes) Code() int {
	return http.StatusOK
}

func (res updateRuleStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateRuleStatusRes) Empty() bool {
	return false
}

type deleteRuleRes struct {
	deleted bool
}

func (res deleteRuleRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusOK
}

func (res deleteRuleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRuleRes) Empty() bool {
	return true
}
