// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/apiutil"
)

const maxLimitSize = 100

type addReq struct {
	token          string
	ClientID       string   `json:"client_id"`
	ExternalID     string   `json:"external_id"`
	ExternalSecret string   `json:"external_secret"`
	Channels       []string `json:"channels"`
	Name           string   `json:"name"`
	Content        string   `json:"content"`
	ClientCert     string   `json:"client_cert"`
	ClientKey      string   `json:"client_key"`
	CACert         string   `json:"ca_cert"`
}

func (req addReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}

	if req.ExternalSecret == "" {
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
	id string
}

func (req entityReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateReq struct {
	id      string
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (req updateReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateCertReq struct {
	clientID    string
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	CACert     string `json:"ca_cert"`
}

func (req updateCertReq) validate() error {
	if req.clientID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateConnReq struct {
	token    string
	id       string
	Channels []string `json:"channels"`
}

func (req updateConnReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listReq struct {
	filter bootstrap.Filter
	offset uint64
	limit  uint64
}

func (req listReq) validate() error {
	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type bootstrapReq struct {
	key string
	id  string
}

func (req bootstrapReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeStateReq struct {
	token string
	id    string
	State bootstrap.State `json:"state"`
}

func (req changeStateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
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
