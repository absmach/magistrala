// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/apiutil"
)

const maxLimitSize = 100

type addReq struct {
	token       string
	domainID    string
	ThingID     string   `json:"thing_id"`
	ExternalID  string   `json:"external_id"`
	ExternalKey string   `json:"external_key"`
	Channels    []string `json:"channels"`
	Name        string   `json:"name"`
	Content     string   `json:"content"`
	ClientCert  string   `json:"client_cert"`
	ClientKey   string   `json:"client_key"`
	CACert      string   `json:"ca_cert"`
}

func (req addReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}

	if req.ExternalKey == "" {
		return apiutil.ErrBearerKey
	}

	if len(req.Channels) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, channel := range req.Channels {
		if channel == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}

type entityReq struct {
	id       string
	domainID string
}

func (req entityReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateReq struct {
	id       string
	domainID string
	Name     string `json:"name"`
	Content  string `json:"content"`
}

func (req updateReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateCertReq struct {
	thingID    string
	domainID   string
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	CACert     string `json:"ca_cert"`
}

func (req updateCertReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	if req.thingID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateConnReq struct {
	token    string
	id       string
	domainID string
	Channels []string `json:"channels"`
}

func (req updateConnReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listReq struct {
	domainID string
	filter   bootstrap.Filter
	offset   uint64
	limit    uint64
}

func (req listReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type bootstrapReq struct {
	key      string
	id       string
	domainID string
}

func (req bootstrapReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeStateReq struct {
	token    string
	id       string
	domainID string
	State    bootstrap.State `json:"state"`
}

func (req changeStateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.State != bootstrap.Inactive &&
		req.State != bootstrap.Active {
		return apiutil.ErrBootstrapState
	}

	return nil
}
