// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package eventlogs_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/eventlogs/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/stretchr/testify/assert"
)

var (
	operation = "users.create"
	payload   = map[string]interface{}{
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

type testEvent struct {
	data map[string]interface{}
}

func (e testEvent) Encode() (map[string]interface{}, error) {
	return e.data, nil
}

func TestHandle(t *testing.T) {
	repo := new(mocks.Repository)
	cases := []struct {
		desc  string
		event testEvent
		err   error
	}{
		{
			desc: "success",
			event: testEvent{
				data: map[string]interface{}{
					"id":          testsutil.GenerateUUID(t),
					"operation":   operation,
					"occurred_at": time.Now().UnixNano(),
					"payload":     payload,
				},
			},
			err: nil,
		},
		{
			desc: "with missing id",
			event: testEvent{
				data: map[string]interface{}{
					"id":          "",
					"operation":   operation,
					"occurred_at": time.Now().UnixNano(),
					"payload":     payload,
				},
			},
			err: nil,
		},
		{
			desc: "with missing operation",
			event: testEvent{
				data: map[string]interface{}{
					"id":          testsutil.GenerateUUID(t),
					"operation":   "",
					"occurred_at": time.Now().UnixNano(),
					"payload":     payload,
				},
			},
			err: nil,
		},
		{
			desc: "with missing occurred_at",
			event: testEvent{
				data: map[string]interface{}{
					"id":          testsutil.GenerateUUID(t),
					"operation":   operation,
					"occurred_at": 0,
					"payload":     payload,
				},
			},
			err: nil,
		},
		{
			desc: "with missing payload",
			event: testEvent{
				data: map[string]interface{}{
					"id":          testsutil.GenerateUUID(t),
					"operation":   operation,
					"occurred_at": time.Now().UnixNano(),
					"payload":     map[string]interface{}{},
				},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data, err := json.Marshal(tc.event.data)
			assert.NoError(t, err)

			var event map[string]interface{}
			err = json.Unmarshal(data, &event)
			assert.NoError(t, err)

			dbEvent := eventlogs.Event{
				ID:         event["id"].(string),
				Operation:  event["operation"].(string),
				OccurredAt: time.Unix(0, int64(event["occurred_at"].(float64))),
				Payload:    event["payload"].(map[string]interface{}),
			}
			repoCall := repo.On("Save", context.Background(), dbEvent)
			err = eventlogs.Handle(context.Background(), repo)(context.Background(), tc.event)
			assert.Equal(t, tc.err, err)
			repoCall.Unset()
		})
	}
}
