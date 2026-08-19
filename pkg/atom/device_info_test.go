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

// fakeDeviceInfoAtom answers the aliased batch query the way Atom does: one
// selection per alias, nulls for entities that could not be read, and an
// errors array alongside the data when any of them failed.
type fakeDeviceInfoAtom struct {
	t          *testing.T
	external   map[string]string
	isGateway  map[string]bool
	unreadable map[string]struct{}
	requests   int
}

func (f *fakeDeviceInfoAtom) handle(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(f.t, r)
	if !strings.Contains(payload.Query, "query EntityDeviceInfo") {
		f.t.Fatalf("unexpected query: %s", payload.Query)
	}
	f.requests++

	data := map[string]any{}
	errs := []map[string]any{}
	for i := 0; ; i++ {
		raw, ok := payload.Variables[fmt.Sprintf("id%d", i)]
		if !ok {
			break
		}
		id, _ := raw.(string)
		alias := fmt.Sprintf("e%d", i)
		if _, blocked := f.unreadable[id]; blocked {
			data[alias] = nil
			errs = append(errs, map[string]any{"message": "forbidden", "path": []any{alias}})
			continue
		}
		entity := map[string]any{"id": id}
		if external, ok := f.external[id]; ok {
			entity["external_id"] = external
		}
		if gw, ok := f.isGateway[id]; ok {
			entity["attributes"] = map[string]any{"is_gateway": gw}
		}
		data[alias] = entity
	}

	body := map[string]any{"data": data}
	if len(errs) > 0 {
		body["errors"] = errs
	}
	_ = writeJSON(w, body)
}

func newDeviceInfoClient(t *testing.T, fake *fakeDeviceInfoAtom) *Client {
	t.Helper()
	fake.t = t
	srv := httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(srv.Close)
	return NewClient(Config{URL: srv.URL, Timeout: time.Second})
}

func TestEntityDeviceInfoResolvesExternalIDAndGatewayFlag(t *testing.T) {
	fake := &fakeDeviceInfoAtom{
		external:  map[string]string{"uuid-1": "serial-1", "uuid-2": "serial-2"},
		isGateway: map[string]bool{"uuid-1": true, "uuid-2": false},
	}
	client := newDeviceInfoClient(t, fake)

	got, unreadable, err := client.EntityDeviceInfo(context.Background(), []string{"uuid-1", "uuid-2"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(unreadable) != 0 {
		t.Fatalf("expected no unreadable ids, got %v", unreadable)
	}
	if got["uuid-1"] != (DeviceInfo{ExternalID: "serial-1", IsGateway: true}) {
		t.Fatalf("uuid-1: got %+v", got["uuid-1"])
	}
	if got["uuid-2"] != (DeviceInfo{ExternalID: "serial-2", IsGateway: false}) {
		t.Fatalf("uuid-2: got %+v", got["uuid-2"])
	}
}

// An ordinary device never has is_gateway set at all — absence must decode as
// false, not be mistaken for an unresolved id.
func TestEntityDeviceInfoTreatsAbsentAttributeAsNotAGateway(t *testing.T) {
	fake := &fakeDeviceInfoAtom{external: map[string]string{"uuid-1": "serial-1"}}
	client := newDeviceInfoClient(t, fake)

	got, _, err := client.EntityDeviceInfo(context.Background(), []string{"uuid-1"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	info, ok := got["uuid-1"]
	if !ok {
		t.Fatalf("expected uuid-1 to resolve, got %v", got)
	}
	if info.IsGateway {
		t.Fatalf("a device with no is_gateway attribute must resolve as not a gateway, got %+v", info)
	}
}

// A device with no external id must still resolve — MG-01 keeps such devices
// in the grant, and R1's fix needs their gateway flag regardless.
func TestEntityDeviceInfoResolvesDeviceWithNoExternalID(t *testing.T) {
	fake := &fakeDeviceInfoAtom{isGateway: map[string]bool{"uuid-1": false}}
	client := newDeviceInfoClient(t, fake)

	got, _, err := client.EntityDeviceInfo(context.Background(), []string{"uuid-1"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	info, ok := got["uuid-1"]
	if !ok {
		t.Fatalf("expected uuid-1 to resolve despite no external id, got %v", got)
	}
	if info.ExternalID != "" {
		t.Fatalf("expected empty external id, got %q", info.ExternalID)
	}
}

// A regression test for Q4: a per-entity failure must be surfaced in the
// second return value, not just silently absent from the first -- callers
// that need to tell "could not read this time" apart from "confirmed no
// info" (readAuthorizer's resolveDeviceInfo, to avoid caching a transient
// failure as a stable negative) depend on this distinction actually leaving
// the client.
func TestEntityDeviceInfoToleratesPerEntityFailures(t *testing.T) {
	fake := &fakeDeviceInfoAtom{
		external:   map[string]string{"uuid-1": "serial-1"},
		unreadable: map[string]struct{}{"uuid-2": {}},
	}
	client := newDeviceInfoClient(t, fake)

	got, unreadable, err := client.EntityDeviceInfo(context.Background(), []string{"uuid-1", "uuid-2"})
	if err != nil {
		t.Fatalf("one unreadable entity must not fail the batch: %s", err)
	}
	if _, ok := got["uuid-2"]; ok {
		t.Fatalf("an unreadable entity must not appear in the resolved map: %v", got)
	}
	if _, ok := unreadable["uuid-2"]; !ok {
		t.Fatalf("expected uuid-2 to be reported as unreadable, got %v", unreadable)
	}
	if got["uuid-1"].ExternalID != "serial-1" {
		t.Fatalf("expected the readable id to survive, got %v", got)
	}
}
