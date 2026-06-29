// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"math"
	"testing"
	"time"

	api "github.com/absmach/magistrala/api/http"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
)

func TestAtomRepositoryCreateAlarmSuppressesDuplicateActiveSeverity(t *testing.T) {
	ts := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	existing := testAlarm("alarm-1", ts)
	store := &alarmAtomStore{
		resources: []atom.Resource{alarmProjection(existing)},
	}
	repo := NewAtomRepository(store)

	_, err := repo.CreateAlarm(context.Background(), testAlarm("alarm-2", ts.Add(time.Minute)))
	if !errors.Contains(err, repoerr.ErrNotFound) {
		t.Fatalf("expected duplicate suppression, got %v", err)
	}
	if store.created.ID != "" {
		t.Fatalf("duplicate alarm should not be created: %#v", store.created)
	}
}

func TestAtomRepositoryCreateAlarmCreatesOnSeverityChange(t *testing.T) {
	ts := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	existing := testAlarm("alarm-1", ts)
	next := testAlarm("alarm-2", ts.Add(time.Minute))
	next.Severity = existing.Severity + 1
	store := &alarmAtomStore{
		resources: []atom.Resource{alarmProjection(existing)},
	}
	repo := NewAtomRepository(store)

	created, err := repo.CreateAlarm(context.Background(), next)
	if err != nil {
		t.Fatalf("create alarm: %v", err)
	}
	if created.ID != next.ID || created.Severity != next.Severity {
		t.Fatalf("unexpected created alarm: %#v", created)
	}
	if store.created.ID != next.ID || store.created.Kind != atom.KindAlarm || store.created.Name != next.ID {
		t.Fatalf("unexpected created resource: %#v", store.created)
	}
}

func TestAtomRepositoryUpdateAlarmMergesMutableFields(t *testing.T) {
	ts := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	current := testAlarm("alarm-1", ts)
	store := &alarmAtomStore{
		resources: []atom.Resource{alarmProjection(current)},
	}
	repo := NewAtomRepository(store)

	updatedAt := ts.Add(time.Hour)
	got, err := repo.UpdateAlarm(context.Background(), Alarm{
		ID:             current.ID,
		Status:         ClearedStatus,
		UpdatedAt:      updatedAt,
		UpdatedBy:      "user-1",
		ResolvedBy:     "user-1",
		ResolvedAt:     updatedAt,
		AcknowledgedBy: "user-2",
		Metadata:       Metadata{"note": "resolved"},
	})
	if err != nil {
		t.Fatalf("update alarm: %v", err)
	}
	if got.Status != ClearedStatus || got.RuleID != current.RuleID || got.ResolvedBy != "user-1" || got.AcknowledgedBy != "user-2" {
		t.Fatalf("unexpected updated alarm: %#v", got)
	}
	if got.Metadata["note"] != "resolved" {
		t.Fatalf("metadata not updated: %#v", got.Metadata)
	}
	if store.updated.ID != current.ID || store.updated.Attributes["status"] != Cleared {
		t.Fatalf("unexpected updated resource: %#v", store.updated)
	}
}

func TestAtomRepositoryListAlarmsFiltersAndSorts(t *testing.T) {
	ts := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	first := testAlarm("alarm-1", ts)
	second := testAlarm("alarm-2", ts.Add(time.Hour))
	second.Severity = 20
	otherChannel := testAlarm("alarm-3", ts.Add(2*time.Hour))
	otherChannel.ChannelID = "other-channel"
	store := &alarmAtomStore{
		resources: []atom.Resource{
			alarmProjection(first),
			alarmProjection(second),
			alarmProjection(otherChannel),
		},
	}
	repo := NewAtomRepository(store)

	page, err := repo.ListAllAlarms(context.Background(), PageMetadata{
		DomainID:  "domain-1",
		ChannelID: "channel-1",
		Status:    AllStatus,
		Severity:  math.MaxUint8,
		Offset:    0,
		Limit:     10,
		Order:     api.CreatedAtOrder,
		Dir:       api.DescDir,
	})
	if err != nil {
		t.Fatalf("list alarms: %v", err)
	}
	if page.Total != 2 || len(page.Alarms) != 2 {
		t.Fatalf("unexpected page: %#v", page)
	}
	if page.Alarms[0].ID != second.ID || page.Alarms[1].ID != first.ID {
		t.Fatalf("unexpected order: %#v", page.Alarms)
	}
}

func testAlarm(id string, createdAt time.Time) Alarm {
	return Alarm{
		ID:          id,
		RuleID:      "rule-1",
		DomainID:    "domain-1",
		ChannelID:   "channel-1",
		ClientID:    "client-1",
		Subtopic:    "temperature",
		Status:      ActiveStatus,
		Measurement: "temperature",
		Value:       "91.2",
		Unit:        "C",
		Threshold:   "80",
		Cause:       "high temperature",
		Severity:    90,
		CreatedAt:   createdAt,
	}
}

type alarmAtomStore struct {
	resources []atom.Resource
	created   atom.Resource
	updated   atom.Resource
	deleted   string
}

func (s *alarmAtomStore) CreateResource(_ context.Context, resource atom.Resource) (atom.Resource, error) {
	s.created = resource
	s.resources = append(s.resources, resource)
	return resource, nil
}

func (s *alarmAtomStore) UpdateResource(_ context.Context, id string, resource atom.Resource) (atom.Resource, error) {
	s.updated = resource
	for i, res := range s.resources {
		if res.ID == id {
			s.resources[i] = resource
			return resource, nil
		}
	}
	return atom.Resource{}, repoerr.ErrNotFound
}

func (s *alarmAtomStore) GetResource(_ context.Context, id string) (atom.Resource, error) {
	for _, res := range s.resources {
		if res.ID == id {
			return res, nil
		}
	}
	return atom.Resource{}, repoerr.ErrNotFound
}

func (s *alarmAtomStore) ListResources(_ context.Context, q atom.Query) (atom.ResourceList, error) {
	var items []atom.Resource
	for _, res := range s.resources {
		if q.Kind != "" && res.Kind != q.Kind {
			continue
		}
		if q.TenantID != "" && res.TenantID != q.TenantID {
			continue
		}
		items = append(items, res)
	}
	total := uint64(len(items))
	start := q.Offset
	if start > total {
		start = total
	}
	end := total
	if q.Limit > 0 && start+q.Limit < end {
		end = start + q.Limit
	}
	return atom.ResourceList{Items: items[start:end], Total: total}, nil
}

func (s *alarmAtomStore) DeleteResource(_ context.Context, id string) error {
	s.deleted = id
	for i, res := range s.resources {
		if res.ID == id {
			s.resources = append(s.resources[:i], s.resources[i+1:]...)
			return nil
		}
	}
	return repoerr.ErrNotFound
}
