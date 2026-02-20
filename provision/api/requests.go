// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	apiutil "github.com/absmach/supermq/api/http/util"
)

type provisionReq struct {
	token       string
	Name        string `json:"name"`
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
}

func (req provisionReq) validate() error {
	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}

	if req.ExternalKey == "" {
		return apiutil.ErrBearerKey
	}

	if req.Name == "" {
		return apiutil.ErrMissingName
	}

	return nil
}

type certReq struct {
	token    string
	ClientID string `json:"client_id"`
	TTL      string `json:"ttl,omitempty"`
}

func (req certReq) validate() error {
	if req.ClientID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
