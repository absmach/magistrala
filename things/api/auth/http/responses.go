//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "net/http"

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

type canAccessByIDRes struct{}

func (res canAccessByIDRes) Code() int {
	return http.StatusOK
}

func (res canAccessByIDRes) Headers() map[string]string {
	return map[string]string{}
}

func (res canAccessByIDRes) Empty() bool {
	return true
}
