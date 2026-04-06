// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/journal"
	aevents "github.com/absmach/magistrala/journal/events"
	"github.com/absmach/magistrala/journal/mocks"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	operation = "users.create"
	payload   = map[string]any{
		"temperature": rand.Float64(),
		"humidity":    float64(rand.Intn(1000)),
		"locations": []any{
			strings.Repeat("a", 100),
			strings.Repeat("a", 100),
		},
		"status": "active",
	}
	idProvider = uuid.New()
)

type testEvent struct {
	data map[string]any
	err  error
}

func (e testEvent) Encode() (map[string]any, error) {
	return e.data, e.err
}

func NewTestEvent(data map[string]any, err error) testEvent {
	return testEvent{data: data, err: err}
}

func TestHandle(t *testing.T) {
	repo := new(mocks.Repository)
	svc := journal.NewService(idProvider, repo)

	cases := []struct {
		desc      string
		event     map[string]any
		encodeErr error
		repoErr   error
		err       error
	}{
		{
			desc: "success",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with encode error",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			encodeErr: errors.New("encode error"),
			err:       errors.New("encode error"),
		},
		{
			desc: "with missing operation",
			event: map[string]any{
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with empty operation",
			event: map[string]any{
				"operation":   "",
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid operation",
			event: map[string]any{
				"operation":   1,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with missing occurred_at",
			event: map[string]any{
				"operation": operation,
				"id":        testsutil.GenerateUUID(t),
				"tags":      []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":    float64(rand.Intn(1000)),
				"metadata":  payload,
			},
			err: nil,
		},
		{
			desc: "with empty occurred_at",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(0),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid occurred_at",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": "invalid",
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with missing metadata",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
			},
			err: nil,
		},
		{
			desc: "with empty metadata",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    map[string]any{},
			},
			err: nil,
		},
		{
			desc: "with invalid metadata",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    1,
			},
			err: nil,
		},
		{
			desc: "with missing attributes",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with empty attributes",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          "",
				"tags":        []any{},
				"number":      float64(0),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid attributes",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"nested": map[string]any{
					"key": float64(rand.Intn(1000)),
					"nested": map[string]any{
						"key": float64(rand.Intn(1000)),
						"nested": map[string]any{
							"key": float64(rand.Intn(1000)),
							"nested": map[string]any{
								"key": float64(rand.Intn(1000)),
								"nested": map[string]any{
									"key": float64(rand.Intn(1000)),
									"nested": map[string]any{
										"key": float64(rand.Intn(1000)),
									},
								},
							},
						},
					},
				},
				"metadata": payload,
			},
			err: nil,
		},
		{
			desc: "success",
			event: map[string]any{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []any{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			repoErr: repoerr.ErrCreateEntity,
			err:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data, err := json.Marshal(tc.event)
			assert.NoError(t, err)

			event := map[string]any{}
			err = json.Unmarshal(data, &event)
			assert.NoError(t, err)

			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoErr)
			err = aevents.Handle(svc)(context.Background(), NewTestEvent(event, tc.encodeErr))
			switch {
			case tc.err == nil:
				assert.NoError(t, err)
			default:
				assert.ErrorContains(t, err, tc.err.Error())
			}
			repoCall.Unset()
		})
	}
}
