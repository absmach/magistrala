// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
)

const maxLimitSize = 100

type page struct {
	offset   uint64
	limit    uint64
	order    string
	dir      string
	name     string
	metadata map[string]interface{}
	tag      string
	roleID   string
	roleName string
	actions  []string
	status   domains.Status
}

type createDomainReq struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Alias    string                 `json:"alias"`
}

func (req createDomainReq) validate() error {
	if req.ID != "" {
		return api.ValidateUUID(req.ID)
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
	domainID string
}

func (req retrieveDomainRequest) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateDomainReq struct {
	domainID string
	Name     *string                 `json:"name,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
	Tags     *[]string               `json:"tags,omitempty"`
	Alias    *string                 `json:"alias,omitempty"`
}

func (req updateDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listDomainsReq struct {
	page
}

func (req listDomainsReq) validate() error {
	return nil
}

type enableDomainReq struct {
	domainID string
}

func (req enableDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type disableDomainReq struct {
	domainID string
}

func (req disableDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type freezeDomainReq struct {
	domainID string
}

func (req freezeDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type sendInvitationReq struct {
	InviteeUserID string `json:"invitee_user_id,omitempty"`
	RoleID        string `json:"role_id,omitempty"`
}

func (req *sendInvitationReq) validate() error {
	if req.InviteeUserID == "" || req.RoleID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listInvitationsReq struct {
	domains.InvitationPageMeta
}

func (req *listInvitationsReq) validate() error {
	if req.InvitationPageMeta.Limit > maxLimitSize || req.InvitationPageMeta.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}

type acceptInvitationReq struct {
	DomainID string `json:"domain_id,omitempty"`
}

func (req *acceptInvitationReq) validate() error {
	if req.DomainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type invitationReq struct {
	userID   string
	domainID string
}

func (req *invitationReq) validate() error {
	if req.userID == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}
