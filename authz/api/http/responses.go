// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*addPolicyRes)(nil)
	_ mainflux.Response = (*removePolicyRes)(nil)
)

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

type errorRes struct {
	Err string `json:"error"`
}

type removePolicyRes struct {
	removed bool
}

func (res removePolicyRes) Code() int {
	if res.removed {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res removePolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removePolicyRes) Empty() bool {
	return true
}
