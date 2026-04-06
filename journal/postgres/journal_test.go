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
	payload   = map[string]any{
		"temperature": rand.Float64(),
		"humidity":    float64(rand.Intn(1000)),
		"locations": []any{
			strings.Repeat("a", 100),
			strings.Repeat("a", 100),
		},
		"status": "active",
		"nested": map[string]any{
			"nested": map[string]any{
				"nested": map[string]any{
					"nested": map[string]any{
						"key": "value",
					},
				},
			},
		},
	}

	entityID           = testsutil.GenerateUUID(&testing.T{})
	domain             = testsutil.GenerateUUID(&testing.T{})
	clientOperation    = "client.create"
	channelOperation   = "channel.create"
	groupOperation     = "group.create"
	clientAttributesV1 = map[string]any{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "client",
		"tags":       []any{"tag1", "tag2"},
		"domain":     domain,
		"metadata":   payload,
		"identity":   testsutil.GenerateUUID(&testing.T{}),
	}
	clientAttributesV2 = map[string]any{
		"entity_id": entityID,
		"metadata":  payload,
	}
	userAttributesV1 = map[string]any{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "user",
		"tags":       []any{"tag1", "tag2"},
		"domain":     domain,
		"metadata":   payload,
		"identity":   testsutil.GenerateUUID(&testing.T{}),
	}
	userAttributesV2 = map[string]any{
		"user_id":  entityID,
		"metadata": payload,
	}
	channelAtttributes = map[string]any{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "channel",
	}
	groupAttributes = map[string]any{
		"id":         entityID,
		"status":     "enabled",
		"created_at": time.Now().Add(-time.Hour),
		"name":       "group",
	}
	validTimeStamp   = time.Now().UTC().Truncate(time.Millisecond)
	errJournalExists = errors.NewRequestError("journal entry already exists")
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
			err: errJournalExists,
		},
		{
			desc: "with massive journal metadata and attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation,
				OccurredAt: time.Now(),
				Attributes: map[string]any{
					"attributes": map[string]any{
						"attributes": map[string]any{
							"attributes": map[string]any{
								"attributes": map[string]any{
									"attributes": map[string]any{
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
				Metadata: map[string]any{
					"metadata": map[string]any{
						"metadata": map[string]any{
							"metadata": map[string]any{
								"metadata": map[string]any{
									"metadata": map[string]any{
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
				Attributes: map[string]any{"invalid": make(chan struct{})},
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
				Attributes: map[string]any{},
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
				Metadata:   map[string]any{"invalid": make(chan struct{})},
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
				Metadata:   map[string]any{},
				Attributes: payload,
			},
			err: nil,
		},
		{
			desc: "with domain in attributes",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  operation + ".with.domain.in.attributes",
				OccurredAt: time.Now(),
				Attributes: map[string]any{
					"domain": testsutil.GenerateUUID(t),
					"data":   "test",
				},
				Metadata: payload,
			},
			err: nil,
		},
		{
			desc: "with domain operation prefix",
			journal: journal.Journal{
				ID:         testsutil.GenerateUUID(t),
				Operation:  "domain.create",
				OccurredAt: time.Now(),
				Attributes: map[string]any{
					"id":     testsutil.GenerateUUID(t),
					"name":   "test-domain",
					"status": "enabled",
				},
				Metadata: payload,
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
			Domain:     domain,
			Operation:  fmt.Sprintf("%s-%d", operation, i),
			OccurredAt: time.Now().UTC().Truncate(time.Microsecond),
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
		if i%13 == 0 {
			j.Operation = fmt.Sprintf("%s-%d", channelOperation, i)
			j.Attributes = channelAtttributes
		}
		if i%17 == 0 {
			j.Operation = fmt.Sprintf("%s-%d", groupOperation, i)
			j.Attributes = groupAttributes
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
			desc: "with channel entity type",
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   entityID,
				EntityType: journal.ChannelEntity,
			},
			response: journal.JournalsPage{
				Total:    uint64(len(extractEntities(items, journal.ChannelEntity, entityID))),
				Offset:   0,
				Limit:    10,
				Journals: extractEntities(items, journal.ChannelEntity, entityID)[:10],
			},
		},
		{
			desc: "with group entity type",
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   entityID,
				EntityType: journal.GroupEntity,
			},
			response: journal.JournalsPage{
				Total:    uint64(len(extractEntities(items, journal.GroupEntity, entityID))),
				Offset:   0,
				Limit:    10,
				Journals: extractEntities(items, journal.GroupEntity, entityID)[:10],
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
				tc.response.Journals[i].Attributes = map[string]any{}
				page.Journals[i].Attributes = map[string]any{}
				tc.response.Journals[i].Metadata = map[string]any{}
				page.Journals[i].Metadata = map[string]any{}
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
			if strings.HasPrefix(j.Operation, "group.") && (j.Attributes["id"] == entityID || j.Attributes["entity_id"] == entityID) {
				entities = append(entities, j)
			}
		case journal.ClientEntity:
			if strings.HasPrefix(j.Operation, "client.") && (j.Attributes["id"] == entityID || j.Attributes["entity_id"] == entityID) {
				entities = append(entities, j)
			}
		case journal.ChannelEntity:
			if strings.HasPrefix(j.Operation, "channel.") && (j.Attributes["id"] == entityID || j.Attributes["entity_id"] == entityID) {
				entities = append(entities, j)
			}
		}
	}

	return entities
}

func TestSaveClientTelemetry(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err, fmt.Sprintf("clean clients_telemetry unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	firstSeen := time.Now().UTC().Truncate(time.Millisecond)
	lastSeen := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)

	cases := []struct {
		desc      string
		telemetry journal.ClientTelemetry
		err       error
	}{
		{
			desc: "save client telemetry successfully",
			telemetry: journal.ClientTelemetry{
				ClientID:         clientID,
				DomainID:         domainID,
				InboundMessages:  10,
				OutboundMessages: 5,
				FirstSeen:        firstSeen,
				LastSeen:         lastSeen,
			},
			err: nil,
		},
		{
			desc: "save duplicate client telemetry",
			telemetry: journal.ClientTelemetry{
				ClientID:         clientID,
				DomainID:         domainID,
				InboundMessages:  20,
				OutboundMessages: 10,
				FirstSeen:        firstSeen,
				LastSeen:         lastSeen,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "save client telemetry with zero messages",
			telemetry: journal.ClientTelemetry{
				ClientID:         testsutil.GenerateUUID(t),
				DomainID:         domainID,
				InboundMessages:  0,
				OutboundMessages: 0,
				FirstSeen:        firstSeen,
				LastSeen:         time.Time{},
			},
			err: nil,
		},
		{
			desc: "save client telemetry with high message counts",
			telemetry: journal.ClientTelemetry{
				ClientID:         testsutil.GenerateUUID(t),
				DomainID:         testsutil.GenerateUUID(t),
				InboundMessages:  1000000,
				OutboundMessages: 999999,
				FirstSeen:        firstSeen,
				LastSeen:         lastSeen,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.SaveClientTelemetry(context.Background(), tc.telemetry)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}

func TestDeleteClientTelemetry(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err, fmt.Sprintf("clean clients_telemetry unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)

	ct := journal.ClientTelemetry{
		ClientID:         clientID,
		DomainID:         domainID,
		InboundMessages:  10,
		OutboundMessages: 5,
		FirstSeen:        time.Now().UTC(),
		LastSeen:         time.Now().UTC(),
	}

	err := repo.SaveClientTelemetry(context.Background(), ct)
	require.Nil(t, err)

	cases := []struct {
		desc     string
		clientID string
		domainID string
		err      error
	}{
		{
			desc:     "delete existing client telemetry",
			clientID: clientID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "delete non-existing client telemetry",
			clientID: testsutil.GenerateUUID(t),
			domainID: domainID,
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteClientTelemetry(context.Background(), tc.clientID, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveClientTelemetry(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err, fmt.Sprintf("clean clients_telemetry unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	firstSeen := time.Now().UTC().Truncate(time.Millisecond)
	lastSeen := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)

	ct := journal.ClientTelemetry{
		ClientID:         clientID,
		DomainID:         domainID,
		InboundMessages:  10,
		OutboundMessages: 5,
		FirstSeen:        firstSeen,
		LastSeen:         lastSeen,
	}

	err := repo.SaveClientTelemetry(context.Background(), ct)
	require.Nil(t, err)

	cases := []struct {
		desc     string
		clientID string
		domainID string
		response journal.ClientTelemetry
		err      error
	}{
		{
			desc:     "retrieve existing client telemetry",
			clientID: clientID,
			domainID: domainID,
			response: ct,
			err:      nil,
		},
		{
			desc:     "retrieve non-existing client telemetry",
			clientID: testsutil.GenerateUUID(t),
			domainID: domainID,
			response: journal.ClientTelemetry{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := repo.RetrieveClientTelemetry(context.Background(), tc.clientID, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.response.ClientID, result.ClientID)
				assert.Equal(t, tc.response.DomainID, result.DomainID)
				assert.Equal(t, tc.response.InboundMessages, result.InboundMessages)
				assert.Equal(t, tc.response.OutboundMessages, result.OutboundMessages)
				assert.Equal(t, tc.response.FirstSeen.Unix(), result.FirstSeen.Unix())
				assert.Equal(t, tc.response.LastSeen.Unix(), result.LastSeen.Unix())
			}
		})
	}
}

func TestAddSubscription(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM subscriptions")
		require.Nil(t, err)
		_, err = db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err)
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)

	ct := journal.ClientTelemetry{
		ClientID:  clientID,
		DomainID:  domainID,
		FirstSeen: time.Now().UTC(),
	}

	err := repo.SaveClientTelemetry(context.Background(), ct)
	require.Nil(t, err)

	cases := []struct {
		desc         string
		subscription journal.ClientSubscription
		err          error
	}{
		{
			desc: "add subscription successfully",
			subscription: journal.ClientSubscription{
				ID:           testsutil.GenerateUUID(t),
				SubscriberID: testsutil.GenerateUUID(t),
				ChannelID:    testsutil.GenerateUUID(t),
				Subtopic:     "subtopic",
				ClientID:     clientID,
			},
			err: nil,
		},
		{
			desc: "add subscription with empty subtopic",
			subscription: journal.ClientSubscription{
				ID:           testsutil.GenerateUUID(t),
				SubscriberID: testsutil.GenerateUUID(t),
				ChannelID:    testsutil.GenerateUUID(t),
				Subtopic:     "",
				ClientID:     clientID,
			},
			err: nil,
		},
		{
			desc: "add duplicate subscription",
			subscription: journal.ClientSubscription{
				ID:           testsutil.GenerateUUID(t),
				SubscriberID: testsutil.GenerateUUID(t),
				ChannelID:    testsutil.GenerateUUID(t),
				Subtopic:     "another-subtopic",
				ClientID:     clientID,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.AddSubscription(context.Background(), tc.subscription)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}

func TestCountSubscriptions(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM subscriptions")
		require.Nil(t, err)
		_, err = db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err)
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)

	ct := journal.ClientTelemetry{
		ClientID:  clientID,
		DomainID:  domainID,
		FirstSeen: time.Now().UTC(),
	}

	err := repo.SaveClientTelemetry(context.Background(), ct)
	require.Nil(t, err)

	for i := 0; i < 3; i++ {
		sub := journal.ClientSubscription{
			ID:           testsutil.GenerateUUID(t),
			SubscriberID: testsutil.GenerateUUID(t),
			ChannelID:    testsutil.GenerateUUID(t),
			Subtopic:     fmt.Sprintf("subtopic%d", i),
			ClientID:     clientID,
		}
		err := repo.AddSubscription(context.Background(), sub)
		require.Nil(t, err)
	}

	cases := []struct {
		desc     string
		clientID string
		count    uint64
		err      error
	}{
		{
			desc:     "count subscriptions for existing client",
			clientID: clientID,
			count:    3,
			err:      nil,
		},
		{
			desc:     "count subscriptions for non-existing client",
			clientID: testsutil.GenerateUUID(t),
			count:    0,
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			count, err := repo.CountSubscriptions(context.Background(), tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
			assert.Equal(t, tc.count, count)
		})
	}
}

func TestRemoveSubscription(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM subscriptions")
		require.Nil(t, err)
		_, err = db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err)
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	subscriberID := testsutil.GenerateUUID(t)

	ct := journal.ClientTelemetry{
		ClientID:  clientID,
		DomainID:  domainID,
		FirstSeen: time.Now().UTC(),
	}

	err := repo.SaveClientTelemetry(context.Background(), ct)
	require.Nil(t, err)

	sub := journal.ClientSubscription{
		ID:           testsutil.GenerateUUID(t),
		SubscriberID: subscriberID,
		ChannelID:    testsutil.GenerateUUID(t),
		Subtopic:     "subtopic",
		ClientID:     clientID,
	}

	err = repo.AddSubscription(context.Background(), sub)
	require.Nil(t, err)

	cases := []struct {
		desc         string
		subscriberID string
		err          error
	}{
		{
			desc:         "remove existing subscription",
			subscriberID: subscriberID,
			err:          nil,
		},
		{
			desc:         "remove non-existing subscription",
			subscriberID: testsutil.GenerateUUID(t),
			err:          nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveSubscription(context.Background(), tc.subscriberID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}

func TestIncrementInboundMessages(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err)
	})
	repo := postgres.NewRepository(database)

	clientID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	firstSeen := time.Now().UTC().Truncate(time.Millisecond)

	cases := []struct {
		desc            string
		telemetry       journal.ClientTelemetry
		expectedInbound uint64
		err             error
		setupExisting   bool
		existingInbound uint64
	}{
		{
			desc: "increment inbound messages for new client",
			telemetry: journal.ClientTelemetry{
				ClientID:  clientID,
				DomainID:  domainID,
				FirstSeen: firstSeen,
				LastSeen:  firstSeen,
			},
			expectedInbound: 1,
			setupExisting:   false,
			err:             nil,
		},
		{
			desc: "increment inbound messages for existing client",
			telemetry: journal.ClientTelemetry{
				ClientID:  clientID,
				DomainID:  domainID,
				FirstSeen: firstSeen,
				LastSeen:  firstSeen.Add(time.Hour),
			},
			expectedInbound: 2,
			setupExisting:   true,
			existingInbound: 1,
			err:             nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.IncrementInboundMessages(context.Background(), tc.telemetry)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))

			if err == nil {
				result, err := repo.RetrieveClientTelemetry(context.Background(), tc.telemetry.ClientID, tc.telemetry.DomainID)
				require.Nil(t, err)
				assert.Equal(t, tc.expectedInbound, result.InboundMessages)
			}
		})
	}
}

func TestIncrementOutboundMessages(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM subscriptions")
		require.Nil(t, err)
		_, err = db.Exec("DELETE FROM clients_telemetry")
		require.Nil(t, err)
	})
	repo := postgres.NewRepository(database)

	clientID1 := testsutil.GenerateUUID(t)
	clientID2 := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	channelID := testsutil.GenerateUUID(t)
	subtopic := "test/subtopic"

	for i, cid := range []string{clientID1, clientID2} {
		ct := journal.ClientTelemetry{
			ClientID:  cid,
			DomainID:  domainID,
			FirstSeen: time.Now().UTC(),
		}
		err := repo.SaveClientTelemetry(context.Background(), ct)
		require.Nil(t, err)

		for j := 0; j < 2; j++ {
			sub := journal.ClientSubscription{
				ID:           testsutil.GenerateUUID(t),
				SubscriberID: fmt.Sprintf("subscriber-%d-%d", i, j),
				ChannelID:    channelID,
				Subtopic:     subtopic,
				ClientID:     cid,
			}
			err = repo.AddSubscription(context.Background(), sub)
			require.Nil(t, err)
		}
	}

	cases := []struct {
		desc              string
		channelID         string
		subtopic          string
		expectedIncrement uint64
		setupAdditional   bool
		err               error
	}{
		{
			desc:              "increment outbound messages for subscribed clients with multiple subscriptions",
			channelID:         channelID,
			subtopic:          subtopic,
			expectedIncrement: 2,
			err:               nil,
		},
		{
			desc:              "increment for non-existing channel",
			channelID:         testsutil.GenerateUUID(t),
			subtopic:          subtopic,
			expectedIncrement: 0,
			err:               nil,
		},
		{
			desc:              "increment with different subtopic",
			channelID:         channelID,
			subtopic:          "different/subtopic",
			expectedIncrement: 0,
			err:               nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.IncrementOutboundMessages(context.Background(), tc.channelID, tc.subtopic)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))

			if err == nil && tc.expectedIncrement > 0 {
				for _, cid := range []string{clientID1, clientID2} {
					result, err := repo.RetrieveClientTelemetry(context.Background(), cid, domainID)
					require.Nil(t, err)
					assert.Equal(t, tc.expectedIncrement, result.OutboundMessages)
				}
			}
		})
	}
}
