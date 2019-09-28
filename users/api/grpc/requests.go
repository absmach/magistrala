//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import "github.com/mainflux/mainflux/users"

type identityReq struct {
	token string
}

func (req identityReq) validate() error {
	if req.token == "" {
		return users.ErrMalformedEntity
	}
	return nil
}
