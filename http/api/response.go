// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/supermq"
)

var _ supermq.Response = (*publishMessageRes)(nil)

type publishMessageRes struct{}

func (res publishMessageRes) Code() int {
	return http.StatusAccepted
}

func (res publishMessageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res publishMessageRes) Empty() bool {
	return true
}
