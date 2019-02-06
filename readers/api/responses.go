//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var _ mainflux.Response = (*listMessagesRes)(nil)

type listMessagesRes struct {
	Messages []mainflux.Message `json:"messages"`
}

func (res listMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMessagesRes) Code() int {
	return http.StatusOK
}

func (res listMessagesRes) Empty() bool {
	return false
}
