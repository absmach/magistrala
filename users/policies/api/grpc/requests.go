// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/mainflux/mainflux/internal/apiutil"
)

// authReq represents authorization request. It contains:
// 1. subject - an action invoker (client)
// 2. object - an entity over which action will be executed (client, group, computation, dataset)
// 3. action - type of action that will be executed (read/write)
// 4. entity_type - type of entity (client, group, computation, dataset).
type authReq struct {
	subject    string
	object     string
	action     string
	entityType string
}

func (req authReq) validate() error {
	if req.subject == "" {
		return apiutil.ErrMissingPolicySub
	}
	if req.object == "" {
		return apiutil.ErrMissingPolicyObj
	}
	if req.action == "" {
		return apiutil.ErrMalformedPolicyAct
	}
	if req.entityType == "" {
		return apiutil.ErrMissingPolicyEntityType
	}

	return nil
}

type identifyReq struct {
	token string
}

func (req identifyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}
