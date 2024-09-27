// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/absmach/magistrala/auth"
)

type identityReq struct {
	token string
}

type issueReq struct {
	userID   string
	domainID string // optional
	keyType  auth.KeyType
}

type refreshReq struct {
	refreshToken string
	domainID     string // optional
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
