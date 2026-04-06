// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/magistrala/api/http/util"
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

type retrieveStatusReq struct {
	ID string
}

func (req retrieveStatusReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type retrieveIDByRouteReq struct {
	Route string
}

func (req retrieveIDByRouteReq) validate() error {
	if req.Route == "" {
		return apiutil.ErrMissingRoute
	}

	return nil
}
