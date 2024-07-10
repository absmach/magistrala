// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
)

type page struct {
	offset     uint64
	limit      uint64
	order      string
	dir        string
	name       string
	metadata   map[string]interface{}
	tag        string
	permission string
	status     auth.Status
}

type createDomainReq struct {
	token    string
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Alias    string                 `json:"alias"`
}

func (req createDomainReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Name == "" {
		return apiutil.ErrMissingName
	}

	if req.Alias == "" {
		return apiutil.ErrMissingAlias
	}

	return nil
}

type retrieveDomainRequest struct {
	token    string
	domainID string
}

func (req retrieveDomainRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type retrieveDomainPermissionsRequest struct {
	token    string
	domainID string
}

func (req retrieveDomainPermissionsRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateDomainReq struct {
	token    string
	domainID string
	Name     *string                 `json:"name,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
	Tags     *[]string               `json:"tags,omitempty"`
	Alias    *string                 `json:"alias,omitempty"`
}

func (req updateDomainReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listDomainsReq struct {
	token  string
	userID string
	page
}

func (req listDomainsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type enableDomainReq struct {
	token    string
	domainID string
}

func (req enableDomainReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type disableDomainReq struct {
	token    string
	domainID string
}

func (req disableDomainReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type freezeDomainReq struct {
	token    string
	domainID string
}

func (req freezeDomainReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type assignUsersReq struct {
	token    string
	domainID string
	UserIDs  []string `json:"user_ids"`
	Relation string   `json:"relation"`
}

func (req assignUsersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.UserIDs) == 0 {
		return apiutil.ErrMissingID
	}

	if req.Relation == "" {
		return apiutil.ErrMissingRelation
	}

	return nil
}

type unassignUserReq struct {
	token    string
	domainID string
	UserID   string `json:"user_id"`
}

func (req unassignUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	if req.UserID == "" {
		return apiutil.ErrMalformedPolicy
	}

	return nil
}
