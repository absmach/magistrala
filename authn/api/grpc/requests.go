// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/mainflux/mainflux/authn"

type identityReq struct {
	token string
	kind  uint32
}

func (req identityReq) validate() error {
	if req.token == "" {
		return authn.ErrMalformedEntity
	}
	if req.kind != authn.UserKey &&
		req.kind != authn.APIKey &&
		req.kind != authn.RecoveryKey {
		return authn.ErrMalformedEntity
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
		return authn.ErrUnauthorizedAccess
	}
	if req.keyType != authn.UserKey &&
		req.keyType != authn.APIKey &&
		req.keyType != authn.RecoveryKey {
		return authn.ErrMalformedEntity
	}

	return nil
}
