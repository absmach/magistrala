// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

func TestJournalsPage_MarshalJSON(t *testing.T) {
	occurredAt := time.Now()

	cases := []struct {
		desc string
		page journal.JournalsPage
		res  string
	}{
		{
			desc: "empty page",
			page: journal.JournalsPage{
				Journals: []journal.Journal(nil),
			},
			res: `{"total":0,"offset":0,"limit":0,"journals":[]}`,
		},
		{
			desc: "page with journals",
			page: journal.JournalsPage{
				Total:  1,
				Offset: 0,
				Limit:  0,
				Journals: []journal.Journal{
					{
						Operation:  "123",
						OccurredAt: occurredAt,
						Attributes: map[string]interface{}{"123": "123"},
						Metadata:   map[string]interface{}{"123": "123"},
					},
				},
			},
			res: fmt.Sprintf(`{"total":1,"offset":0,"limit":0,"journals":[{"operation":"123","occurred_at":"%s","attributes":{"123":"123"},"metadata":{"123":"123"}}]}`, occurredAt.Format(time.RFC3339Nano)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data, err := tc.page.MarshalJSON()
			assert.NoError(t, err, "Unexpected error: %v", err)
			assert.Equal(t, tc.res, string(data))
		})
	}
}

func TestEntityType(t *testing.T) {
	cases := []struct {
		desc        string
		e           journal.EntityType
		str         string
		authString  string
		queryString string
	}{
		{
			desc:       "UserEntity",
			e:          journal.UserEntity,
			str:        "user",
			authString: "user",
		},
		{
			desc:       "ClientEntity",
			e:          journal.ClientEntity,
			str:        "client",
			authString: "client",
		},
		{
			desc:       "GroupEntity",
			e:          journal.GroupEntity,
			str:        "group",
			authString: "group",
		},
		{
			desc:       "ChannelEntity",
			e:          journal.ChannelEntity,
			str:        "channel",
			authString: "group",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.str, tc.e.String())
			assert.Equal(t, tc.authString, tc.e.AuthString())
			assert.NotEmpty(t, tc.e.Query())
		})
	}
}

func TestToEntityType(t *testing.T) {
	cases := []struct {
		desc        string
		entityType  string
		expected    journal.EntityType
		expectedErr error
	}{
		{
			desc:       "UserEntity",
			entityType: "user",
			expected:   journal.UserEntity,
		},
		{
			desc:       "ClientEntity",
			entityType: "client",
			expected:   journal.ClientEntity,
		},
		{
			desc:       "GroupEntity",
			entityType: "group",
			expected:   journal.GroupEntity,
		},
		{
			desc:       "ChannelEntity",
			entityType: "channel",
			expected:   journal.ChannelEntity,
		},
		{
			desc:        "Invalid entity type",
			entityType:  "invalid",
			expectedErr: apiutil.ErrInvalidEntityType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			entityType, err := journal.ToEntityType(tc.entityType)
			assert.Equal(t, tc.expected, entityType)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
