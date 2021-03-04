// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"time"

	"github.com/mainflux/mainflux/auth"
)

type issueKeyReq struct {
	token    string
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// It is not possible to issue Reset key using HTTP API.
func (req issueKeyReq) validate() error {
	if req.Type == auth.UserKey {
		return nil
	}
	if req.token == "" || (req.Type != auth.APIKey) {
		return auth.ErrMalformedEntity
	}
	return nil
}

type keyReq struct {
	token string
	id    string
}

func (req keyReq) validate() error {
	if req.token == "" || req.id == "" {
		return auth.ErrMalformedEntity
	}
	return nil
}
