// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/invitations"
	"github.com/stretchr/testify/assert"
)

var (
	errMissingRelation = errors.New("missing relation")
	errInvalidRelation = errors.New("invalid relation")
)

func TestInvitation_MarshalJSON(t *testing.T) {
	cases := []struct {
		desc string
		page invitations.InvitationPage
		res  string
	}{
		{
			desc: "empty page",
			page: invitations.InvitationPage{
				Invitations: []invitations.Invitation(nil),
			},
			res: `{"total":0,"offset":0,"limit":0,"invitations":[]}`,
		},
		{
			desc: "page with invitations",
			page: invitations.InvitationPage{
				Total:  1,
				Offset: 0,
				Limit:  0,
				Invitations: []invitations.Invitation{
					{
						InvitedBy: "John",
						UserID:    "123",
						DomainID:  "123",
					},
				},
			},
			res: `{"total":1,"offset":0,"limit":0,"invitations":[{"invited_by":"John","user_id":"123","domain_id":"123","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","confirmed_at":"0001-01-01T00:00:00Z"}]}`,
		},
	}

	for _, tc := range cases {
		data, err := tc.page.MarshalJSON()
		assert.NoError(t, err, "Unexpected error: %v", err)
		assert.Equal(t, tc.res, string(data), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.res, string(data)))
	}
}

func TestCheckRelation(t *testing.T) {
	cases := []struct {
		relation string
		err      error
	}{
		{"", errMissingRelation},
		{"admin", errInvalidRelation},
		{"editor", nil},
		{"viewer", nil},
		{"member", nil},
		{"domain", nil},
		{"parent_group", nil},
		{"role_group", nil},
		{"group", nil},
		{"platform", nil},
	}

	for _, tc := range cases {
		err := invitations.CheckRelation(tc.relation)
		assert.Equal(t, tc.err, err, "CheckRelation(%q) expected %v, got %v", tc.relation, tc.err, err)
	}
}
