// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

type issueKeyReq struct {
	token    string
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// It is not possible to issue Reset key using HTTP API.
func (req issueKeyReq) validate() error {
	if req.Type != auth.APIKey || req.token == "" {
		return errors.ErrAuthentication
	}
	return nil
}

type keyReq struct {
	token string
	id    string
}

func (req keyReq) validate() error {
	if req.token == "" || req.id == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}
