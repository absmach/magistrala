// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"time"

	"github.com/mainflux/mainflux/authn"
)

type issueKeyReq struct {
	issuer   string
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// It is not possible to issue Reset key using HTTP API.
func (req issueKeyReq) validate() error {
	if req.Type == authn.UserKey {
		return nil
	}
	if req.issuer == "" || (req.Type != authn.APIKey) {
		return authn.ErrMalformedEntity
	}
	return nil
}

type keyReq struct {
	issuer string
	id     string
}

func (req keyReq) validate() error {
	if req.issuer == "" || req.id == "" {
		return authn.ErrMalformedEntity
	}
	return nil
}
