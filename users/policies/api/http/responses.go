// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*authorizeRes)(nil)
	_ mainflux.Response = (*addPolicyRes)(nil)
	_ mainflux.Response = (*viewPolicyRes)(nil)
	_ mainflux.Response = (*listPolicyRes)(nil)
	_ mainflux.Response = (*updatePolicyRes)(nil)
	_ mainflux.Response = (*deletePolicyRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type authorizeRes struct {
	Authorized bool `json:"authorized"`
}

func (res authorizeRes) Code() int {
	return http.StatusOK
}

func (res authorizeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res authorizeRes) Empty() bool {
	return false
}

type addPolicyRes struct {
	created bool
}

func (res addPolicyRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res addPolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addPolicyRes) Empty() bool {
	return true
}

type viewPolicyRes struct {
	OwnerID   string    `json:"owner_id"`
	Subject   string    `json:"subject"`
	Object    string    `json:"object"`
	Actions   []string  `json:"actions"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (res viewPolicyRes) Code() int {
	return http.StatusOK
}

func (res viewPolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewPolicyRes) Empty() bool {
	return false
}

type updatePolicyRes struct {
	updated bool
}

func (res updatePolicyRes) Code() int {
	return http.StatusNoContent
}

func (res updatePolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updatePolicyRes) Empty() bool {
	return true
}

type listPolicyRes struct {
	pageRes
	Policies []viewPolicyRes `json:"policies"`
}

func (res listPolicyRes) Code() int {
	return http.StatusOK
}

func (res listPolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listPolicyRes) Empty() bool {
	return false
}

type deletePolicyRes struct{}

func (res deletePolicyRes) Code() int {
	return http.StatusNoContent
}

func (res deletePolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deletePolicyRes) Empty() bool {
	return true
}
