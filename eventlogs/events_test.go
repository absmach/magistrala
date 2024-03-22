// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package eventlogs_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	"github.com/stretchr/testify/assert"
)

func TestEventsPage_MarshalJSON(t *testing.T) {
	occurredAt := time.Now()

	cases := []struct {
		desc string
		page eventlogs.EventsPage
		res  string
	}{
		{
			desc: "empty page",
			page: eventlogs.EventsPage{
				Events: []eventlogs.Event(nil),
			},
			res: `{"total":0,"offset":0,"limit":0,"events":[]}`,
		},
		{
			desc: "page with events",
			page: eventlogs.EventsPage{
				Total:  1,
				Offset: 0,
				Limit:  0,
				Events: []eventlogs.Event{
					{
						ID:         "123",
						Operation:  "123",
						OccurredAt: occurredAt,
						Payload:    map[string]interface{}{"123": "123"},
					},
				},
			},
			res: fmt.Sprintf(`{"total":1,"offset":0,"limit":0,"events":[{"id":"123","operation":"123","occurred_at":"%s","payload":{"123":"123"}}]}`, occurredAt.Format(time.RFC3339Nano)),
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
