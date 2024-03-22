// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/eventlogs/postgres"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	invalidUUID = strings.Repeat("a", 37)
	operation   = "user.create"
	payload     = map[string]interface{}{
		"temperature": rand.Float64(),
		"humidity":    rand.Float64(),
		"sensor_id":   rand.Intn(1000),
		"locations": []string{
			strings.Repeat("a", 1024),
			strings.Repeat("a", 1024),
			strings.Repeat("a", 1024),
		},
		"status":    rand.Intn(1000),
		"timestamp": time.Now().UnixNano(),
	}
)

func TestEventsSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM events")
		require.Nil(t, err, fmt.Sprintf("clean events unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	id := testsutil.GenerateUUID(t)
	occurredAt := time.Now()

	cases := []struct {
		desc  string
		event eventlogs.Event
		err   error
	}{
		{
			desc: "new event successfully",
			event: eventlogs.Event{
				ID:         id,
				Operation:  operation,
				OccurredAt: occurredAt,
				Payload:    payload,
			},
			err: nil,
		},
		{
			desc: "with duplicate event",
			event: eventlogs.Event{
				ID:         id,
				Operation:  operation,
				OccurredAt: occurredAt,
				Payload:    payload,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "with massive event payload",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Payload: map[string]interface{}{
					"metadata": map[string]interface{}{
						"metadata": map[string]interface{}{
							"metadata": map[string]interface{}{
								"metadata": map[string]interface{}{
									"metadata": map[string]interface{}{
										"data": payload,
									},
									"data": payload,
								},
								"data": payload,
							},
							"data": payload,
						},
						"data": payload,
					},
					"data": payload,
				},
			},
			err: nil,
		},
		{
			desc: "with invalid event id",
			event: eventlogs.Event{
				ID:         invalidUUID,
				Operation:  operation,
				OccurredAt: time.Now(),
				Payload:    payload,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "with nil event id",
			event: eventlogs.Event{
				Operation:  operation,
				OccurredAt: time.Now(),
				Payload:    payload,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "with empty event id",
			event: eventlogs.Event{
				ID:         "",
				Operation:  operation,
				OccurredAt: time.Now(),
				Payload:    payload,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "with nil event operation",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				OccurredAt: time.Now(),
				Payload:    payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty event operation",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				Operation:  "",
				OccurredAt: time.Now(),
				Payload:    payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with nil event occurred_at",
			event: eventlogs.Event{
				ID:        testsutil.GenerateUUID(t),
				Operation: operation,
				Payload:   payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty event occurred_at",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Time{},
				Payload:    payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with nil event payload",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "with empty event payload",
			event: eventlogs.Event{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Payload:    map[string]interface{}{},
			},
			err: nil,
		},
		{
			desc:  "with empty event",
			event: eventlogs.Event{},
			err:   repoerr.ErrMalformedEntity,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch err := repo.Save(context.Background(), tc.event); {
			case err == nil:
				assert.Nil(t, err)
			default:
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
		})
	}
}

func TestEventsRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM events")
		require.Nil(t, err, fmt.Sprintf("clean events unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	num := 200

	var items []eventlogs.Event
	for i := 0; i < num; i++ {
		event := eventlogs.Event{
			ID:         testsutil.GenerateUUID(t),
			Operation:  fmt.Sprintf("%s-%d", operation, i),
			OccurredAt: time.Now().UTC().Truncate(time.Millisecond),
			Payload:    payload,
		}
		err := repo.Save(context.Background(), event)
		require.Nil(t, err, fmt.Sprintf("create event unexpected error: %s", err))
		event.Payload = nil
		items = append(items, event)
	}

	reversedItems := make([]eventlogs.Event, len(items))
	copy(reversedItems, items)
	sort.Slice(reversedItems, func(i, j int) bool {
		return reversedItems[i].OccurredAt.After(reversedItems[j].OccurredAt)
	})

	cases := []struct {
		desc     string
		page     eventlogs.Page
		response eventlogs.EventsPage
		err      error
	}{
		{
			desc: "successfully",
			page: eventlogs.Page{
				Offset: 0,
				Limit:  1,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  1,
				Events: items[:1],
			},
			err: nil,
		},
		{
			desc: "with offset and empty limit",
			page: eventlogs.Page{
				Offset: 10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 10,
				Limit:  0,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with limit and empty offset",
			page: eventlogs.Page{
				Limit: 50,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  50,
				Events: items[:50],
			},
		},
		{
			desc: "with offset and limit",
			page: eventlogs.Page{
				Offset: 10,
				Limit:  50,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 10,
				Limit:  50,
				Events: items[10:60],
			},
		},
		{
			desc: "with offset out of range",
			page: eventlogs.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 1000,
				Limit:  50,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: eventlogs.Page{
				Offset: 170,
				Limit:  50,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 170,
				Limit:  50,
				Events: items[170:200],
			},
		},
		{
			desc: "with limit out of range",
			page: eventlogs.Page{
				Offset: 0,
				Limit:  1000,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  1000,
				Events: items,
			},
		},
		{
			desc: "with empty page",
			page: eventlogs.Page{},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  0,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with id",
			page: eventlogs.Page{
				ID:     items[0].ID,
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event{items[0]},
			},
		},
		{
			desc: "with operation",
			page: eventlogs.Page{
				Operation: items[0].Operation,
				Offset:    0,
				Limit:     10,
			},
			response: eventlogs.EventsPage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event{items[0]},
			},
		},
		{
			desc: "with payload",
			page: eventlogs.Page{
				WithPayload: true,
				Offset:      0,
				Limit:       10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: items[:10],
			},
		},
		{
			desc: "with from",
			page: eventlogs.Page{
				From:   items[0].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: items[:10],
			},
		},
		{
			desc: "with to",
			page: eventlogs.Page{
				To:     items[num-1].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: items[:10],
			},
		},
		{
			desc: "with from and to",
			page: eventlogs.Page{
				From:   items[0].OccurredAt,
				To:     items[num-1].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: items[:10],
			},
		},
		{
			desc: "with asc direction",
			page: eventlogs.Page{
				Direction: "asc",
				Offset:    0,
				Limit:     10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: items[:10],
			},
		},
		{
			desc: "with desc direction",
			page: eventlogs.Page{
				Direction: "desc",
				Offset:    0,
				Limit:     10,
			},
			response: eventlogs.EventsPage{
				Total:  uint64(num),
				Offset: 0,
				Limit:  10,
				Events: reversedItems[:10],
			},
		},
		{
			desc: "with all filters",
			page: eventlogs.Page{
				ID:          items[0].ID,
				Operation:   items[0].Operation,
				From:        items[0].OccurredAt,
				To:          items[num-1].OccurredAt,
				WithPayload: true,
				Direction:   "asc",
				Offset:      0,
				Limit:       10,
			},
			response: eventlogs.EventsPage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event{items[0]},
			},
		},
		{
			desc: "with invalid id",
			page: eventlogs.Page{
				ID:     testsutil.GenerateUUID(t),
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with invalid operation",
			page: eventlogs.Page{
				Operation: strings.Repeat("a", 37),
				Offset:    0,
				Limit:     10,
			},
			response: eventlogs.EventsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with invalid from",
			page: eventlogs.Page{
				From:   time.Now().UTC().Truncate(time.Millisecond),
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event(nil),
			},
		},
		{
			desc: "with invalid to",
			page: eventlogs.Page{
				To:     time.Now().UTC().Truncate(time.Millisecond).Add(-time.Hour),
				Offset: 0,
				Limit:  10,
			},
			response: eventlogs.EventsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
				Events: []eventlogs.Event(nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.RetrieveAll(context.Background(), tc.page)
			assert.Equal(t, tc.response.Total, page.Total)
			assert.Equal(t, tc.response.Offset, page.Offset)
			assert.Equal(t, tc.response.Limit, page.Limit)
			if !tc.page.WithPayload {
				assert.ElementsMatch(t, page.Events, tc.response.Events)
			}
			assert.Equal(t, tc.err, err)
		})
	}
}
