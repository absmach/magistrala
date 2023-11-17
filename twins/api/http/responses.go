// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/twins"
)

var (
	_ magistrala.Response = (*twinRes)(nil)
	_ magistrala.Response = (*viewTwinRes)(nil)
	_ magistrala.Response = (*viewStateRes)(nil)
	_ magistrala.Response = (*twinsPageRes)(nil)
	_ magistrala.Response = (*statesPageRes)(nil)
	_ magistrala.Response = (*removeRes)(nil)
)

type twinRes struct {
	id      string
	created bool
}

func (res twinRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res twinRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/twins/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res twinRes) Empty() bool {
	return true
}

type viewTwinRes struct {
	Owner       string                 `json:"owner,omitempty"`
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Revision    int                    `json:"revision"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
	Definitions []twins.Definition     `json:"definitions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (res viewTwinRes) Code() int {
	return http.StatusOK
}

func (res viewTwinRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewTwinRes) Empty() bool {
	return false
}

type viewStateRes struct {
	TwinID     string                 `json:"twin_id"`
	ID         int64                  `json:"id"`
	Definition int                    `json:"definition"`
	Created    time.Time              `json:"created"`
	Payload    map[string]interface{} `json:"payload"`
}

func (res viewStateRes) Code() int {
	return http.StatusOK
}

func (res viewStateRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewStateRes) Empty() bool {
	return false
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type twinsPageRes struct {
	pageRes
	Twins []viewTwinRes `json:"twins"`
}

func (res twinsPageRes) Code() int {
	return http.StatusOK
}

func (res twinsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res twinsPageRes) Empty() bool {
	return false
}

type statesPageRes struct {
	pageRes
	States []viewStateRes `json:"states"`
}

func (res statesPageRes) Code() int {
	return http.StatusOK
}

func (res statesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res statesPageRes) Empty() bool {
	return false
}

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}
