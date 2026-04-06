// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/magistrala/api/http/util"
)

type retrieveUsersReq struct {
	ids    []string
	offset uint64
	limit  uint64
}

func (req retrieveUsersReq) validate() error {
	if len(req.ids) == 0 {
		return apiutil.ErrMissingUserID
	}

	return nil
}
