// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
)

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
