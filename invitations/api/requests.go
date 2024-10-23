// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/apiutil"
)

const maxLimitSize = 100

type sendInvitationReq struct {
	token    string
	DomainID string `json:"domain_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Relation string `json:"relation,omitempty"`
	Resend   bool   `json:"resend,omitempty"`
}

func (req *sendInvitationReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.UserID == "" {
		return apiutil.ErrMissingID
	}
	if req.DomainID == "" {
		return apiutil.ErrMissingDomainID
	}
	if err := invitations.CheckRelation(req.Relation); err != nil {
		return err
	}

	return nil
}

type listInvitationsReq struct {
	token string
	invitations.Page
}

func (req *listInvitationsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Page.DomainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if req.Page.Limit > maxLimitSize || req.Page.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}

type acceptInvitationReq struct {
	token    string
	domainID string
}

func (req *acceptInvitationReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type invitationReq struct {
	token    string
	userID   string
	domainID string
}

func (req *invitationReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.userID == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}
