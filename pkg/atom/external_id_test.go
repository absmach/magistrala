// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeExternalIDAtom answers the aliased batch query the way Atom does: one
// selection per alias, nulls for entities that could not be read, and an errors
// array alongside the data when any of them failed.
type fakeExternalIDAtom struct {
	t        *testing.T
	external map[string]string
	// forbidden ids come back null with an error entry, as a deleted or
	// unreadable entity would.
	forbidden map[string]struct{}
	requests  int
	batches   [][]string
}

func (f *fakeExternalIDAtom) handle(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(f.t, r)
	if !strings.Contains(payload.Query, "query EntityExternalIDs") {
		f.t.Fatalf("unexpected query: %s", payload.Query)
	}
	f.requests++

	data := map[string]any{}
	errs := []map[string]string{}
	batch := []string{}
	for i := 0; ; i++ {
		raw, ok := payload.Variables[fmt.Sprintf("id%d", i)]
		if !ok {
			break
		}
		id, _ := raw.(string)
		batch = append(batch, id)
		alias := fmt.Sprintf("e%d", i)
		if _, blocked := f.forbidden[id]; blocked {
			data[alias] = nil
			errs = append(errs, map[string]string{"message": "Forbidden"})
			continue
		}
		entity := map[string]any{"id": id}
		if external, ok := f.external[id]; ok {
			entity["external_id"] = external
		}
		data[alias] = entity
	}
	f.batches = append(f.batches, batch)

	body := map[string]any{"data": data}
	if len(errs) > 0 {
		body["errors"] = errs
	}
	_ = writeJSON(w, body)
}

func newExternalIDClient(t *testing.T, fake *fakeExternalIDAtom) *Client {
	t.Helper()
	fake.t = t
	srv := httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(srv.Close)
	return NewClient(Config{URL: srv.URL, Timeout: time.Second})
}

func TestEntityExternalIDs(t *testing.T) {
	fake := &fakeExternalIDAtom{external: map[string]string{
		"uuid-1": "Meter.1-01:X",
		"uuid-2": "meter/2,02",
	}}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), []string{"uuid-1", "uuid-2"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resolved ids, got %d: %v", len(got), got)
	}
	// Serials round-trip byte-identically, including characters no separator
	// scheme would survive.
	if got["uuid-1"] != "Meter.1-01:X" || got["uuid-2"] != "meter/2,02" {
		t.Fatalf("external ids were not returned verbatim: %v", got)
	}
	if fake.requests != 1 {
		t.Fatalf("expected one round trip for a small set, got %d", fake.requests)
	}
}

// An entity with no external id is simply absent, so callers cannot mistake it
// for one whose serial is the empty string.
func TestEntityExternalIDsOmitsEntitiesWithoutOne(t *testing.T) {
	fake := &fakeExternalIDAtom{external: map[string]string{"uuid-1": "serial-1"}}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), []string{"uuid-1", "uuid-2"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, ok := got["uuid-2"]; ok {
		t.Fatalf("an entity with no external id must be absent, got %v", got)
	}
	if got["uuid-1"] != "serial-1" {
		t.Fatalf("expected serial-1, got %v", got)
	}
}

// One unreadable entity must not deny the caller every device they hold, so the
// batch keeps the ids it did resolve. An unresolved id contributes nothing to
// the filter it feeds, which is the safe direction.
func TestEntityExternalIDsToleratesPerEntityFailures(t *testing.T) {
	fake := &fakeExternalIDAtom{
		external:  map[string]string{"uuid-1": "serial-1", "uuid-3": "serial-3"},
		forbidden: map[string]struct{}{"uuid-2": {}},
	}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), []string{"uuid-1", "uuid-2", "uuid-3"})
	if err != nil {
		t.Fatalf("one unreadable entity must not fail the batch: %s", err)
	}
	if len(got) != 2 || got["uuid-1"] != "serial-1" || got["uuid-3"] != "serial-3" {
		t.Fatalf("expected the readable ids to survive, got %v", got)
	}
	if _, ok := got["uuid-2"]; ok {
		t.Fatalf("an unreadable entity must not appear: %v", got)
	}
}

func TestEntityExternalIDsBatchesLargeSets(t *testing.T) {
	external := map[string]string{}
	ids := make([]string, 0, 250)
	for i := 0; i < 250; i++ {
		id := fmt.Sprintf("uuid-%d", i)
		ids = append(ids, id)
		external[id] = fmt.Sprintf("serial-%d", i)
	}
	fake := &fakeExternalIDAtom{external: external}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), ids)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(got) != 250 {
		t.Fatalf("expected 250 resolved ids, got %d", len(got))
	}
	if fake.requests != 3 {
		t.Fatalf("expected 250 ids to travel as 3 batches of at most %d, got %d requests", externalIDBatchSize, fake.requests)
	}
	for _, batch := range fake.batches {
		if len(batch) > externalIDBatchSize {
			t.Fatalf("batch of %d exceeds the bound of %d", len(batch), externalIDBatchSize)
		}
	}
}

func TestEntityExternalIDsDeduplicatesAndSkipsEmpty(t *testing.T) {
	fake := &fakeExternalIDAtom{external: map[string]string{"uuid-1": "serial-1"}}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), []string{"uuid-1", "uuid-1", "", "uuid-1"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resolved id, got %v", got)
	}
	if len(fake.batches) != 1 || len(fake.batches[0]) != 1 {
		t.Fatalf("expected one id to be asked for once, got %v", fake.batches)
	}
}

func TestEntityExternalIDsEmptyInputMakesNoRequest(t *testing.T) {
	fake := &fakeExternalIDAtom{}
	client := newExternalIDClient(t, fake)

	got, err := client.EntityExternalIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty result, got %v", got)
	}
	if fake.requests != 0 {
		t.Fatalf("expected no round trip, got %d", fake.requests)
	}
}
