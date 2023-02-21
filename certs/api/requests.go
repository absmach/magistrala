// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"time"

	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/internal/apiutil"
)

const maxLimitSize = 100

type addCertsReq struct {
	token   string
	Name    string `json:"name"`
	ThingID string `json:"thing_id"`
	TTL     string `json:"ttl"`
}

func (req addCertsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ThingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.TTL == "" {
		return apiutil.ErrMissingCertData
	}

	if _, err := time.ParseDuration(req.TTL); err != nil {
		return apiutil.ErrInvalidCertData
	}

	return nil
}

type listReq struct {
	certID     string
	thingID    string
	serial     string
	name       string
	status     string
	token      string
	offset     uint64
	limit      uint64
	certStatus certs.Status
}

func (req *listReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}
	cs, ok := certs.StringToStatus[req.status]
	if !ok {
		return apiutil.ErrInvalidCertData
	}
	req.certStatus = cs
	return nil
}

type viewRevokeRenewRemoveReq struct {
	certID string
	token  string
}

func (req *viewRevokeRenewRemoveReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.certID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type revokeRenewRemoveThingIDReq struct {
	thingID string
	token   string
	limit   int64
}

func (req *revokeRenewRemoveThingIDReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}
