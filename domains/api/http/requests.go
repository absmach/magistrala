// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
)

const maxLimitSize = 100

type createDomainReq struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Route    string         `json:"route"`
}

func (req createDomainReq) validate() error {
	if req.ID != "" {
		return api.ValidateUUID(req.ID)
	}
	if req.Name == "" {
		return apiutil.ErrMissingName
	}
	if req.Route == "" {
		return apiutil.ErrMissingRoute
	}
	if err := validateRoute(req.Route); err != nil {
		return err
	}

	return nil
}

type retrieveDomainRequest struct {
	domainID string
	roles    bool
}

func (req retrieveDomainRequest) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateDomainReq struct {
	domainID string
	Name     *string           `json:"name,omitempty"`
	Metadata *domains.Metadata `json:"metadata,omitempty"`
	Tags     *[]string         `json:"tags,omitempty"`
}

func (req updateDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listDomainsReq struct {
	domains.Page
}

func (req listDomainsReq) validate() error {
	switch req.Order {
	case "", api.NameOrder, api.CreatedAtOrder, api.UpdatedAtOrder:
	default:
		return apiutil.ErrInvalidOrder
	}

	if req.Dir != "" && (req.Dir != api.DescDir && req.Dir != api.AscDir) {
		return apiutil.ErrInvalidDirection
	}

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

type deleteDomainReq struct {
	domainID string
}

func (req deleteDomainReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type sendInvitationReq struct {
	InviteeUserID string `json:"invitee_user_id,omitempty"`
	RoleID        string `json:"role_id,omitempty"`
	Resend        bool   `json:"resend,omitempty"`
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

type deleteInvitationReq struct {
	UserID   string `json:"user_id"`
	domainID string
}

func (req *deleteInvitationReq) validate() error {
	if req.UserID == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

func validateRoute(route string) error {
	if err := api.ValidateUUID(route); err == nil {
		return nil
	}
	if err := api.ValidateRoute(route); err != nil {
		return err
	}

	return nil
}
