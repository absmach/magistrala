// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/mainflux/mainflux/certs"

const maxLimitSize = 100

type addCertsReq struct {
	token   string
	ThingID string `json:"thing_id"`
	KeyBits int    `json:"key_bits"`
	KeyType string `json:"key_type"`
	Valid   string `json:"valid"`
}

func (req addCertsReq) validate() error {
	if req.ThingID == "" && req.token == "" {
		return errUnauthorized
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
		return certs.ErrUnauthorizedAccess
	}
	if req.limit == 0 || req.limit > maxLimitSize {
		return certs.ErrMalformedEntity
	}
	return nil
}

type revokeReq struct {
	token  string
	certID string
}

func (req *revokeReq) validate() error {
	if req.token == "" || req.certID == "" {
		return certs.ErrUnauthorizedAccess
	}

	return nil
}
