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
	readers.PageMetadata
	Total    uint64            `json:"total"`
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
