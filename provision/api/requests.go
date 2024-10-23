// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/absmach/magistrala/pkg/apiutil"

type provisionReq struct {
	token       string
	domainID    string
	Name        string `json:"name"`
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
}

func (req provisionReq) validate() error {
	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.ExternalKey == "" {
		return apiutil.ErrBearerKey
	}

	if req.Name == "" {
		return apiutil.ErrMissingName
	}

	return nil
}

type mappingReq struct {
	token    string
	domainID string
}

func (req mappingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	return nil
}
