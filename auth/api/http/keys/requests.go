// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/apiutil"
)

type issueKeyReq struct {
	token    string
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// It is not possible to issue Reset key using HTTP API.
func (req issueKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Type != auth.LoginKey &&
		req.Type != auth.RecoveryKey &&
		req.Type != auth.APIKey {
		return apiutil.ErrInvalidAPIKey
	}

	return nil
}

type keyReq struct {
	token string
	id    string
}

func (req keyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listKeysReq struct {
	token   string
	subject string
	keyType uint32
	offset  uint64
	limit   uint64
}

func (req listKeysReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}
