// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/supermq/api/http/util"
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

type retrieveEntityReq struct {
	ID string
}

func (req retrieveEntityReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
