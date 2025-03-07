// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"fmt"
	"testing"

	"github.com/absmach/supermq/domains"
	"github.com/stretchr/testify/assert"
)

func TestInvitation_MarshalJSON(t *testing.T) {
	cases := []struct {
		desc string
		page domains.InvitationPage
		res  string
	}{
		{
			desc: "empty page",
			page: domains.InvitationPage{
				Invitations: []domains.Invitation(nil),
			},
			res: `{"total":0,"offset":0,"limit":0,"invitations":[]}`,
		},
		{
			desc: "page with invitations",
			page: domains.InvitationPage{
				Total:  1,
				Offset: 0,
				Limit:  0,
				Invitations: []domains.Invitation{
					{
						InvitedBy:     "John",
						InviteeUserID: "123",
						DomainID:      "123",
					},
				},
			},
			res: `{"total":1,"offset":0,"limit":0,"invitations":[{"invited_by":"John","invitee_user_id":"123","domain_id":"123","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","confirmed_at":"0001-01-01T00:00:00Z","rejected_at":"0001-01-01T00:00:00Z"}]}`,
		},
	}

	for _, tc := range cases {
		data, err := tc.page.MarshalJSON()
		assert.NoError(t, err, "Unexpected error: %v", err)
		assert.Equal(t, tc.res, string(data), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.res, string(data)))
	}
}
