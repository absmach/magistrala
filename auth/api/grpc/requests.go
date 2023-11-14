// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
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
	id      string
	keyType auth.KeyType
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
	value string
}

func (req refreshReq) validate() error {
	if req.value == "" {
		return apiutil.ErrMissingSecret
	}

	return nil
}

// authReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write).
type authReq struct {
	Namespace   string
	SubjectType string
	SubjectKind string
	Subject     string
	Relation    string
	Permission  string
	ObjectType  string
	Object      string
}

func (req authReq) validate() error {
	if req.Subject == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Object == "" {
		return apiutil.ErrMissingPolicyObj
	}

	// if req.SubjectKind == "" {
	// 	return apiutil.ErrMissingPolicySub
	// }

	// if req.Permission == "" {
	// 	return apiutil.ErrMalformedPolicyAct
	// }

	return nil
}

type policyReq struct {
	Namespace   string
	SubjectType string
	Subject     string
	Relation    string
	Permission  string
	ObjectType  string
	Object      string
}

func (req policyReq) validate() error {
	if req.Subject == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Object == "" {
		return apiutil.ErrMissingPolicyObj
	}

	if req.Relation == "" && req.Permission == "" {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}

type listObjectsReq struct {
	Namespace     string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
	Limit         int32
}

type countObjectsReq struct {
	Namespace     string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
}

type listSubjectsReq struct {
	Namespace     string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
	Limit         int32
}

type countSubjectsReq struct {
	Namespace     string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
}
