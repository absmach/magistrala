// Copyright (c) Abstract Machines
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

type policyReq struct {
	Domain      string
	SubjectType string
	Subject     string
	SubjectKind string
	Relation    string
	Permission  string
	ObjectType  string
	ObjectKind  string
	Object      string
}

func (req policyReq) validate() error {
	return nil
}

type policiesReq []policyReq

func (prs policiesReq) validate() error {
	for _, pr := range prs {
		if err := pr.validate(); err != nil {
			return nil
		}
	}
	return nil
}

type listObjectsReq struct {
	Domain        string
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
	Domain        string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
}

type listSubjectsReq struct {
	Domain        string
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
	Domain        string
	SubjectType   string
	Subject       string
	Relation      string
	Permission    string
	ObjectType    string
	Object        string
	NextPageToken string
}

type listPermissionsReq struct {
	Domain            string
	SubjectType       string
	Subject           string
	SubjectRelation   string
	ObjectType        string
	Object            string
	FilterPermissions []string
}
