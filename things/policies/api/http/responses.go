// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things/policies"
)

var (
	_ mainflux.Response = (*policyRes)(nil)
	_ mainflux.Response = (*listPolicyRes)(nil)
	_ mainflux.Response = (*identityRes)(nil)
	_ mainflux.Response = (*authorizeRes)(nil)
	_ mainflux.Response = (*deletePolicyRes)(nil)
)

type policyRes struct {
	Policies []policies.Policy `json:"policies"`
	created  bool
}

func (res policyRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res policyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res policyRes) Empty() bool {
	return false
}

type listPolicyRes struct {
	policies.PolicyPage
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

type identityRes struct {
	ID string `json:"id"`
}

func (res identityRes) Code() int {
	return http.StatusOK
}

func (res identityRes) Headers() map[string]string {
	return map[string]string{}
}

func (res identityRes) Empty() bool {
	return false
}

type authorizeRes struct {
	ThingID    string `json:"thing_id"`
	Authorized bool   `json:"authorized"`
}

func (res authorizeRes) Code() int {
	return http.StatusOK
}

func (res authorizeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res authorizeRes) Empty() bool {
	return true
}
