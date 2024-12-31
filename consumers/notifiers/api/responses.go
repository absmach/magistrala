// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/supermq"
)

var (
	_ supermq.Response = (*createSubRes)(nil)
	_ supermq.Response = (*viewSubRes)(nil)
	_ supermq.Response = (*listSubsRes)(nil)
	_ supermq.Response = (*removeSubRes)(nil)
)

type createSubRes struct {
	ID string
}

func (res createSubRes) Code() int {
	return http.StatusCreated
}

func (res createSubRes) Headers() map[string]string {
	return map[string]string{
		"Location": fmt.Sprintf("/subscriptions/%s", res.ID),
	}
}

func (res createSubRes) Empty() bool {
	return true
}

type viewSubRes struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
	Contact string `json:"contact"`
	Topic   string `json:"topic"`
}

func (res viewSubRes) Code() int {
	return http.StatusOK
}

func (res viewSubRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewSubRes) Empty() bool {
	return false
}

type listSubsRes struct {
	Offset        uint         `json:"offset"`
	Limit         int          `json:"limit"`
	Total         uint         `json:"total,omitempty"`
	Subscriptions []viewSubRes `json:"subscriptions,omitempty"`
}

func (res listSubsRes) Code() int {
	return http.StatusOK
}

func (res listSubsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listSubsRes) Empty() bool {
	return false
}

type removeSubRes struct{}

func (res removeSubRes) Code() int {
	return http.StatusNoContent
}

func (res removeSubRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeSubRes) Empty() bool {
	return true
}
