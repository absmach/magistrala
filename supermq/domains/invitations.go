// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"encoding/json"
	"time"
)

// Invitation is an invitation to join a domain.
type Invitation struct {
	InvitedBy     string    `json:"invited_by"`
	InviteeUserID string    `json:"invitee_user_id"`
	DomainID      string    `json:"domain_id"`
	DomainName    string    `json:"domain_name,omitempty"`
	RoleID        string    `json:"role_id,omitempty"`
	RoleName      string    `json:"role_name,omitempty"`
	Actions       []string  `json:"actions,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	ConfirmedAt   time.Time `json:"confirmed_at,omitempty"`
	RejectedAt    time.Time `json:"rejected_at,omitempty"`
}

// InvitationPage is a page of invitations.
type InvitationPage struct {
	Total       uint64       `json:"total"`
	Offset      uint64       `json:"offset"`
	Limit       uint64       `json:"limit"`
	Invitations []Invitation `json:"invitations"`
}

func (page InvitationPage) MarshalJSON() ([]byte, error) {
	type Alias InvitationPage
	a := struct {
		Alias
	}{
		Alias: Alias(page),
	}

	if a.Invitations == nil {
		a.Invitations = make([]Invitation, 0)
	}

	return json.Marshal(a)
}

type InvitationPageMeta struct {
	Offset            uint64 `json:"offset" db:"offset"`
	Limit             uint64 `json:"limit" db:"limit"`
	InvitedBy         string `json:"invited_by,omitempty" db:"invited_by,omitempty"`
	InviteeUserID     string `json:"invitee_user_id,omitempty" db:"invitee_user_id,omitempty"`
	DomainID          string `json:"domain_id,omitempty" db:"domain_id,omitempty"`
	RoleID            string `json:"role_id,omitempty" db:"role_id,omitempty"`
	InvitedByOrUserID string `db:"invited_by_or_user_id,omitempty"`
	State             State  `json:"state,omitempty"`
}
