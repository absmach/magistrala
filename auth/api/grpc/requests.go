// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

type identityReq struct {
	token string
	kind  uint32
}

func (req identityReq) validate() error {
	if req.token == "" {
		return errors.ErrMalformedEntity
	}
	if req.kind != auth.LoginKey &&
		req.kind != auth.APIKey &&
		req.kind != auth.RecoveryKey {
		return errors.ErrMalformedEntity
	}

	return nil
}

type issueReq struct {
	id      string
	email   string
	keyType uint32
}

func (req issueReq) validate() error {
	if req.email == "" {
		return errors.ErrAuthentication
	}
	if req.keyType != auth.LoginKey &&
		req.keyType != auth.APIKey &&
		req.keyType != auth.RecoveryKey {
		return errors.ErrMalformedEntity
	}

	return nil
}

type assignReq struct {
	token     string
	groupID   string
	memberID  string
	groupType string
}

func (req assignReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}
	if req.groupID == "" || req.memberID == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

type membersReq struct {
	token      string
	groupID    string
	offset     uint64
	limit      uint64
	memberType string
}

func (req membersReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}
	if req.groupID == "" {
		return errors.ErrMalformedEntity
	}
	if req.memberType == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

// authReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write)
type authReq struct {
	Sub string
	Obj string
	Act string
}

func (req authReq) validate() error {
	if req.Sub == "" {
		return errors.ErrMalformedEntity
	}

	if req.Obj == "" {
		return errors.ErrMalformedEntity
	}

	if req.Act == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type addPolicyReq struct {
	Sub string
	Obj string
	Act string
}

func (req addPolicyReq) validate() error {
	if req.Sub == "" || req.Obj == "" || req.Act == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

type deletePolicyReq struct {
	Sub string
	Obj string
	Act string
}

func (req deletePolicyReq) validate() error {
	if req.Sub == "" || req.Obj == "" || req.Act == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

type listPoliciesReq struct {
	Sub string
	Obj string
	Act string
}
