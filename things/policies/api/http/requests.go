// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
)

type createPolicyReq struct {
	token   string
	Subject string   `json:"subject,omitempty"`
	Object  string   `json:"object,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

func (req createPolicyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Object == "" || req.Subject == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type createPoliciesReq struct {
	token    string
	Subjects []string `json:"subjects,omitempty"`
	Objects  []string `json:"objects,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

func (req createPoliciesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Objects) == 0 || len(req.Subjects) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, channelID := range req.Objects {
		if channelID == "" {
			return apiutil.ErrMissingID
		}
	}
	for _, thingID := range req.Subjects {
		if thingID == "" {
			return apiutil.ErrMissingID
		}
	}
	return nil
}

type identifyReq struct {
	secret string
}

func (req identifyReq) validate() error {
	if req.secret == "" {
		return apiutil.ErrMissingSecret
	}

	return nil
}

type authorizeReq struct {
	Subject    string `json:"subject"`
	Object     string `json:"object"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type"`
}

func (req authorizeReq) validate() error {
	if req.Object == "" {
		return apiutil.ErrMissingID
	}
	if req.Subject == "" {
		return apiutil.ErrMissingSecret
	}

	return nil
}

type policyReq struct {
	token   string
	Subject string   `json:"subject,omitempty"`
	Object  string   `json:"object,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

func (req policyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type listPoliciesReq struct {
	token  string
	offset uint64
	limit  uint64
	client string
	group  string
	action string
	owner  string
}

func (req listPoliciesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}
