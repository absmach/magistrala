// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
)

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
	userID   string
	domainID string // optional
	keyType  auth.KeyType
}

func (req issueReq) validate() error {
	if req.keyType != auth.AccessKey &&
		req.keyType != auth.APIKey &&
		req.keyType != auth.RecoveryKey &&
		req.keyType != auth.InvitationKey {
		return apiutil.ErrInvalidAuthKey
	}

	return nil
}

type refreshReq struct {
	refreshToken string
	domainID     string // optional
}

func (req refreshReq) validate() error {
	if req.refreshToken == "" {
		return apiutil.ErrMissingSecret
	}

	return nil
}

// authReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write).
type authReq struct {
	Domain      string
	SubjectType string
	SubjectKind string
	Subject     string
	Relation    string
	Permission  string
	ObjectType  string
	Object      string
}

func (req authReq) validate() error {
	if req.Subject == "" || req.SubjectType == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Object == "" || req.ObjectType == "" {
		return apiutil.ErrMissingPolicyObj
	}

	if req.Permission == "" {
		return apiutil.ErrMalformedPolicyPer
	}

	return nil
}

type deleteUserPoliciesReq struct {
	ID string
}

func (req deleteUserPoliciesReq) validate() error {
	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
