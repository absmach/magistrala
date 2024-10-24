// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/absmach/magistrala/pkg/apiutil"
)

type deleteUserPoliciesReq struct {
	ID string
}

func (req deleteUserPoliciesReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
