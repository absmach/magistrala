// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
)

var _ mainflux.Response = (*pageRes)(nil)

type pageRes struct {
	Total    uint64            `json:"total"`
	Offset   uint64            `json:"offset"`
	Limit    uint64            `json:"limit"`
	Messages []readers.Message `json:"messages,omitempty"`
}

func (res pageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res pageRes) Code() int {
	return http.StatusOK
}

func (res pageRes) Empty() bool {
	return false
}

type errorRes struct {
	Err string `json:"error"`
}
