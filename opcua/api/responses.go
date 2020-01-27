// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/opcua"
)

var _ mainflux.Response = (*browseRes)(nil)

type browseRes struct {
	Nodes []opcua.BrowsedNode `json:"nodes"`
}

func (res browseRes) Code() int {
	return http.StatusOK
}

func (res browseRes) Headers() map[string]string {
	return map[string]string{}
}

func (res browseRes) Empty() bool {
	return false
}
