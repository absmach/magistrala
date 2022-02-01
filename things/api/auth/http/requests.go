// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/mainflux/mainflux/pkg/errors"

type identifyReq struct {
	Token string `json:"token"`
}

func (req identifyReq) validate() error {
	if req.Token == "" {
		return errors.ErrAuthentication
	}

	return nil
}

type canAccessByKeyReq struct {
	chanID string
	Token  string `json:"token"`
}

func (req canAccessByKeyReq) validate() error {
	if req.Token == "" || req.chanID == "" {
		return errors.ErrAuthentication
	}

	return nil
}

type canAccessByIDReq struct {
	chanID  string
	ThingID string `json:"thing_id"`
}

func (req canAccessByIDReq) validate() error {
	if req.ThingID == "" || req.chanID == "" {
		return errors.ErrAuthentication
	}

	return nil
}
