//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/users"

type apiReq interface {
	validate() error
}

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate()
}
