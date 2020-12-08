// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/mainflux/mainflux/authz"
)

type addPolicyReq struct {
	token   string
	Subject string
	Object  string
	Action  string
}

func (req addPolicyReq) validate() error {
	if req.token == "" {
		return authz.ErrUnauthorizedAccess
	}

	if req.Subject == "" || req.Object == "" || req.Action == "" {
		return authz.ErrMalformedEntity
	}

	return nil
}

type removePolicyReq struct {
	token   string
	Subject string
	Object  string
	Action  string
}

func (req removePolicyReq) validate() error {
	if req.token == "" {
		return authz.ErrUnauthorizedAccess
	}

	if req.Subject == "" || req.Object == "" || req.Action == "" {
		return authz.ErrMalformedEntity
	}

	return nil
}
