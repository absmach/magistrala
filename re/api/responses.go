// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/re"
)

var (
	_ magistrala.Response = (*viewRuleRes)(nil)
	_ magistrala.Response = (*addRuleRes)(nil)
	_ magistrala.Response = (*changeRuleStatusRes)(nil)
	_ magistrala.Response = (*rulesPageRes)(nil)
	_ magistrala.Response = (*updateRuleRes)(nil)
	_ magistrala.Response = (*changeRoleStatusRes)(nil)
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
	pageRes
	Rules []viewRuleRes `json:"rules"`
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

type changeRuleStatusRes struct {
	re.Rule `json:",inline"`
}

func (res changeRuleStatusRes) Code() int {
	return http.StatusOK
}

func (res changeRuleStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeRuleStatusRes) Empty() bool {
	return false
}

type changeRoleStatusRes struct {
	deleted bool
}

func (res changeRoleStatusRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusOK
}

func (res changeRoleStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeRoleStatusRes) Empty() bool {
	return true
}
