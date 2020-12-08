// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/mainflux/mainflux/authz"

// AuthZReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write)
type AuthZReq struct {
	Sub string
	Obj string
	Act string
}

func (req AuthZReq) validate() error {
	if req.Sub == "" {
		return authz.ErrInvalidReq
	}

	if req.Obj == "" {
		return authz.ErrInvalidReq
	}

	if req.Act == "" {
		return authz.ErrInvalidReq
	}

	return nil
}
