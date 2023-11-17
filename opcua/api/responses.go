// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/opcua"
)

var _ magistrala.Response = (*browseRes)(nil)

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
