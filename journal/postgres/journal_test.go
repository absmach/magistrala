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

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/journal/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	operation = "user.create"
	payload   = map[string]interface{}{
		"temperature": rand.Float64(),
		"humidity":    float64(rand.Intn(1000)),
		"locations": []interface{}{
			strings.Repeat("a", 100),
			strings.Repeat("a", 100),
		},
		"status": "active",
		"nested": map[string]interface{}{
			"nested": map[string]interface{}{
				"nested": map[string]interface{}{
					"nested": map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
	}

	entityID          = testsutil.GenerateUUID(&testing.T{})
	clientOperation    = "client.create"
	clientAttributesV1 = map[string]interface{}{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "client",
		"tags":       []interface{}{"tag1", "tag2"},
		"domain":     testsutil.GenerateUUID(&testing.T{}),
		"metadata":   payload,
		"identity":   testsutil.GenerateUUID(&testing.T{}),
	}
	clientAttributesV2 = map[string]interface{}{
		"client_id": entityID,
		"metadata":  payload,
	}
	userAttributesV1 = map[string]interface{}{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "user",
		"tags":       []interface{}{"tag1", "tag2"},
		"domain":     testsutil.GenerateUUID(&testing.T{}),
		"metadata":   payload,
		"identity":   testsutil.GenerateUUID(&testing.T{}),
	}
	userAttributesV2 = map[string]interface{}{
		"user_id":  entityID,
		"metadata": payload,
	}
	validTimeStamp = time.Now().UTC().Truncate(time.Millisecond)
)

func TestJournalSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM journal")
		require.Nil(t, err, fmt.Sprintf("clean journal unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	occurredAt := time.Now()
	id := testsutil.GenerateUUID(t)

	cases := []struct {
		desc    string
		journal journal.Journal
		err     error
	}{
		{
			desc: "new journal successfully",
			journal: journal.Journal{
				ID:         id,
				Operation:  operation,
				OccurredAt: occurredAt,
				Attributes: payload,
				Metadata:   payload,
			},
			err: nil,
		},
		{
			desc: "with duplicate journal",
			journal: journal.Journal{
				ID:         id,
				Operation:  operation,
				OccurredAt: occurredAt,
				Attributes: payload,
				Metadata:   payload,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "with massive journal metadata and attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Attributes: map[string]interface{}{
					"attributes": map[string]interface{}{
						"attributes": map[string]interface{}{
							"attributes": map[string]interface{}{
								"attributes": map[string]interface{}{
									"attributes": map[string]interface{}{
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
				Metadata: map[string]interface{}{
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
			desc: "with nil journal operation",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				OccurredAt: time.Now(),
				Attributes: payload,
				Metadata:   payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty journal operation",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  "",
				OccurredAt: time.Now().Add(-time.Hour),
				Attributes: payload,
				Metadata:   payload,
			},
			err: nil,
		},
		{
			desc: "with nil journal occurred_at",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				Attributes: payload,
				Metadata:   payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty journal occurred_at",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Time{},
				Attributes: payload,
				Metadata:   payload,
			},
			err: nil,
		},
		{
			desc: "with nil journal attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation + ".with.nil.attributes",
				OccurredAt: time.Now(),
				Metadata:   payload,
			},
			err: nil,
		},
		{
			desc: "with invalid journal attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Attributes: map[string]interface{}{"invalid": make(chan struct{})},
				Metadata:   payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty journal attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation + ".with.empty.attributes",
				OccurredAt: time.Now(),
				Attributes: map[string]interface{}{},
				Metadata:   payload,
			},
			err: nil,
		},
		{
			desc: "with nil journal metadata",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation + ".with.nil.metadata",
				OccurredAt: time.Now(),
				Attributes: payload,
			},
			err: nil,
		},
		{
			desc: "with invalid journal metadata",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Metadata:   map[string]interface{}{"invalid": make(chan struct{})},
				Attributes: payload,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "with empty journal metadata",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation + ".with.empty.metadata",
				OccurredAt: time.Now(),
				Metadata:   map[string]interface{}{},
				Attributes: payload,
			},
			err: nil,
		},
		{
			desc:    "with empty journal",
			journal: journal.Journal{},
			err:     repoerr.ErrCreateEntity,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch err := repo.Save(context.Background(), tc.journal); {
			case err == nil:
				assert.Nil(t, err)
			default:
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
		})
	}
}

func TestJournalRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM journal")
		require.Nil(t, err, fmt.Sprintf("clean journal unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	num := 200

	var items []journal.Journal
	for i := 0; i < num; i++ {
		j := journal.Journal{
			ID:         testsutil.GenerateUUID(t),
			Operation:  fmt.Sprintf("%s-%d", operation, i),
			OccurredAt: time.Now().UTC().Truncate(time.Millisecond),
			Attributes: userAttributesV1,
			Metadata:   payload,
		}
		if i%2 == 0 {
			j.Operation = fmt.Sprintf("%s-%d", clientOperation, i)
			j.Attributes = clientAttributesV1
		}
		if i%3 == 0 {
			j.Attributes = userAttributesV2
		}
		if i%5 == 0 {
			j.Attributes = clientAttributesV2
		}
		err := repo.Save(context.Background(), j)
		require.Nil(t, err, fmt.Sprintf("create journal unexpected error: %s", err))
		j.ID = ""
		items = append(items, j)
	}

	reversedItems := make([]journal.Journal, len(items))
	copy(reversedItems, items)
	sort.Slice(reversedItems, func(i, j int) bool {
		return reversedItems[i].OccurredAt.After(reversedItems[j].OccurredAt)
	})

	cases := []struct {
		desc     string
		page     journal.Page
		response journal.JournalsPage
		err      error
	}{
		{
			desc: "successfully",
			page: journal.Page{
				Offset: 0,
				Limit:  1,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    1,
				Journals: items[:1],
			},
			err: nil,
		},
		{
			desc: "with offset and empty limit",
			page: journal.Page{
				Offset: 10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   10,
				Limit:    0,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with limit and empty offset",
			page: journal.Page{
				Limit: 50,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    50,
				Journals: items[:50],
			},
		},
		{
			desc: "with offset and limit",
			page: journal.Page{
				Offset: 10,
				Limit:  50,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   10,
				Limit:    50,
				Journals: items[10:60],
			},
		},
		{
			desc: "with offset out of range",
			page: journal.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   1000,
				Limit:    50,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: journal.Page{
				Offset: 170,
				Limit:  50,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   170,
				Limit:    50,
				Journals: items[170:200],
			},
		},
		{
			desc: "with limit out of range",
			page: journal.Page{
				Offset: 0,
				Limit:  1000,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    1000,
				Journals: items,
			},
		},
		{
			desc: "with empty page",
			page: journal.Page{},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    0,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with operation",
			page: journal.Page{
				Operation: items[0].Operation,
				Offset:    0,
				Limit:     10,
			},
			response: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{items[0]},
			},
		},
		{
			desc: "with invalid operation",
			page: journal.Page{
				Operation: strings.Repeat("a", 37),
				Offset:    0,
				Limit:     10,
			},
			response: journal.JournalsPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with attributes",
			page: journal.Page{
				WithAttributes: true,
				Offset:         0,
				Limit:          10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with metadata",
			page: journal.Page{
				WithMetadata: true,
				Offset:       0,
				Limit:        10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with attributes and Metadata",
			page: journal.Page{
				WithAttributes: true,
				WithMetadata:   true,
				Offset:         0,
				Limit:          10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with from",
			page: journal.Page{
				From:   items[0].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with invalid from",
			page: journal.Page{
				From:   time.Now().UTC().Truncate(time.Millisecond).Add(time.Hour),
				Offset: 0,
				Limit:  10,
			},
			response: journal.JournalsPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with to",
			page: journal.Page{
				To:     items[num-1].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with invalid to",
			page: journal.Page{
				To:     time.Now().UTC().Truncate(time.Millisecond).Add(-time.Hour),
				Offset: 0,
				Limit:  10,
			},
			response: journal.JournalsPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with from and to",
			page: journal.Page{
				From:   items[0].OccurredAt,
				To:     items[num-1].OccurredAt,
				Offset: 0,
				Limit:  10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with asc direction",
			page: journal.Page{
				Direction: "ASC",
				Offset:    0,
				Limit:     10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: items[:10],
			},
		},
		{
			desc: "with desc direction",
			page: journal.Page{
				Direction: "DESC",
				Offset:    0,
				Limit:     10,
			},
			response: journal.JournalsPage{
				Total:    uint64(num),
				Offset:   0,
				Limit:    10,
				Journals: reversedItems[:10],
			},
		},
		{
			desc: "with user entity type",
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   entityID,
				EntityType: journal.UserEntity,
			},
			response: journal.JournalsPage{
				Total:    uint64(len(extractEntities(items, journal.UserEntity, entityID))),
				Offset:   0,
				Limit:    10,
				Journals: extractEntities(items, journal.UserEntity, entityID)[:10],
			},
		},
		{
			desc: "with user entity type, attributes and metadata",
			page: journal.Page{
				Offset:         0,
				Limit:          10,
				EntityID:       entityID,
				EntityType:     journal.UserEntity,
				WithAttributes: true,
				WithMetadata:   true,
			},
			response: journal.JournalsPage{
				Total:    uint64(len(extractEntities(items, journal.UserEntity, entityID))),
				Offset:   0,
				Limit:    10,
				Journals: extractEntities(items, journal.UserEntity, entityID)[:10],
			},
		},
		{
			desc: "with client entity type",
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   entityID,
				EntityType: journal.ClientEntity,
			},
			response: journal.JournalsPage{
				Total:    uint64(len(extractEntities(items, journal.ClientEntity, entityID))),
				Offset:   0,
				Limit:    10,
				Journals: extractEntities(items, journal.ClientEntity, entityID)[:10],
			},
		},
		{
			desc: "with invalid entity id",
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   testsutil.GenerateUUID(&testing.T{}),
				EntityType: journal.ChannelEntity,
			},
			response: journal.JournalsPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal(nil),
			},
		},
		{
			desc: "with all filters",
			page: journal.Page{
				Offset:         0,
				Limit:          10,
				Operation:      items[0].Operation,
				From:           items[0].OccurredAt,
				To:             items[num-1].OccurredAt,
				WithAttributes: true,
				WithMetadata:   true,
				Direction:      "asc",
			},
			response: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{items[0]},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.RetrieveAll(context.Background(), tc.page)
			assert.Equal(t, tc.response.Total, page.Total)
			assert.Equal(t, tc.response.Offset, page.Offset)
			assert.Equal(t, tc.response.Limit, page.Limit)
			for i := range tc.response.Journals {
				tc.response.Journals[i].Attributes = map[string]interface{}{}
				page.Journals[i].Attributes = map[string]interface{}{}
				tc.response.Journals[i].Metadata = map[string]interface{}{}
				page.Journals[i].Metadata = map[string]interface{}{}
				tc.response.Journals[i].OccurredAt = validTimeStamp
				page.Journals[i].OccurredAt = validTimeStamp
			}
			assert.ElementsMatch(t, tc.response.Journals, page.Journals)

			assert.Equal(t, tc.err, err)
		})
	}
}

func extractEntities(journals []journal.Journal, entityType journal.EntityType, entityID string) []journal.Journal {
	var entities []journal.Journal
	for _, j := range journals {
		switch entityType {
		case journal.UserEntity:
			if strings.HasPrefix(j.Operation, "user.") && j.Attributes["id"] == entityID || j.Attributes["user_id"] == entityID {
				entities = append(entities, j)
			}
		case journal.GroupEntity:
			if strings.HasPrefix(j.Operation, "group.") && j.Attributes["id"] == entityID || j.Attributes["group_id"] == entityID {
				entities = append(entities, j)
			}
		case journal.ClientEntity:
			if strings.HasPrefix(j.Operation, "client.") && j.Attributes["id"] == entityID || j.Attributes["client_id"] == entityID {
				entities = append(entities, j)
			}
		case journal.ChannelEntity:
			if strings.HasPrefix(j.Operation, "channel.") && j.Attributes["id"] == entityID || j.Attributes["group_id"] == entityID {
				entities = append(entities, j)
			}
		}
	}

	return entities
}
