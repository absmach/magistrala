// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/apiutil"
)

const maxLimitSize = 100

type sendInvitationReq struct {
	domainID string
	UserID   string `json:"user_id,omitempty"`
	Relation string `json:"relation,omitempty"`
	Resend   bool   `json:"resend,omitempty"`
}

func (req *sendInvitationReq) validate() error {
	if req.UserID == "" {
		return apiutil.ErrMissingID
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
	domainID string
}

func (req *acceptInvitationReq) validate() error {
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

	return nil
}
