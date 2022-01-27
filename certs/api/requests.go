// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/pkg/errors"
)

const maxLimitSize = 100

type addCertsReq struct {
	token   string
	ThingID string `json:"thing_id"`
	KeyBits int    `json:"key_bits"`
	KeyType string `json:"key_type"`
	TTL     string `json:"ttl"`
}

func (req addCertsReq) validate() error {
	if req.ThingID == "" && req.token == "" {
		return errUnauthorized
	}

	if req.TTL == "" || req.KeyType == "" || req.KeyBits == 0 {
		return errors.ErrMalformedEntity
	}
	return nil
}

type listReq struct {
	thingID string
	token   string
	offset  uint64
	limit   uint64
}

func (req *listReq) validate() error {
	if req.token == "" {
		return errors.ErrUnauthorizedAccess
	}
	if req.limit == 0 || req.limit > maxLimitSize {
		return errors.ErrMalformedEntity
	}
	return nil
}

type viewReq struct {
	serialID string
	token    string
}

func (req *viewReq) validate() error {
	if req.token == "" {
		return errUnauthorized
	}
	if req.serialID == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type revokeReq struct {
	token  string
	certID string
}

func (req *revokeReq) validate() error {
	if req.token == "" || req.certID == "" {
		return errors.ErrUnauthorizedAccess
	}

	return nil
}
