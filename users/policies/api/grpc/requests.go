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
	Sub        string
	Obj        string
	Act        string
	EntityType string
}

func (req authReq) validate() error {
	if req.Sub == "" {
		return apiutil.ErrMissingPolicySub
	}
	if req.Obj == "" {
		return apiutil.ErrMissingPolicyObj
	}
	if req.Act == "" {
		return apiutil.ErrMalformedPolicyAct
	}
	if req.EntityType == "" {
		return apiutil.ErrMissingPolicyEntityType
	}

	return nil
}

type identityReq struct {
	token string
}

func (req identityReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type issueReq struct {
	email    string
	password string
}

func (req issueReq) validate() error {
	if req.email == "" {
		return apiutil.ErrMissingEmail
	}
	if req.password == "" {
		return apiutil.ErrMissingPass
	}
	return nil
}

type addPolicyReq struct {
	Token string
	Sub   string
	Obj   string
	Act   []string
}

func (req addPolicyReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Sub == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Obj == "" {
		return apiutil.ErrMissingPolicyObj
	}

	if len(req.Act) == 0 {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}

type policyReq struct {
	Token string
	Sub   string
	Obj   string
	Act   string
}

func (req policyReq) validate() error {
	if req.Sub == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Obj == "" {
		return apiutil.ErrMissingPolicyObj
	}

	if req.Act == "" {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}

type listPoliciesReq struct {
	Token string
	Sub   string
	Obj   string
	Act   string
}
