// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/things/policies"
)

type authorizeReq struct {
	entityType string
	clientID   string
	groupID    string
	action     string
}

func (req authorizeReq) validate() error {
	if req.clientID == "" {
		return apiutil.ErrMissingPolicySub
	}
	if req.groupID == "" {
		return apiutil.ErrMissingPolicyObj
	}
	if ok := policies.ValidateAction(req.action); !ok {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}

type identifyReq struct {
	key string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}
