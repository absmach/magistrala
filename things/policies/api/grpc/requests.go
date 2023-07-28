// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/things/policies"
)

type authorizeReq struct {
	subject    string
	object     string
	action     string
	entityType string
}

func (req authorizeReq) validate() error {
	if req.subject == "" {
		return apiutil.ErrMissingPolicySub
	}
	if req.object == "" {
		return apiutil.ErrMissingPolicyObj
	}
	if ok := policies.ValidateAction(req.action); !ok {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}

type identifyReq struct {
	secret string
}

func (req identifyReq) validate() error {
	if req.secret == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}
