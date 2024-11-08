// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/apiutil"
)

const maxLimitSize = 100

type sendInvitationReq struct {
	UserID   string `json:"user_id,omitempty"`
	DomainID string `json:"domain_id,omitempty"`
	Relation string `json:"relation,omitempty"`
	Resend   bool   `json:"resend,omitempty"`
}

func (req *sendInvitationReq) validate() error {
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
	invitations.Page
}

func (req *listInvitationsReq) validate() error {
	if req.Page.Limit > maxLimitSize || req.Page.Limit < 1 {
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
