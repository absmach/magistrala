// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeAtomDevices is a minimal in-memory Atom stand-in. Like the real Atom,
// its updateEntity replaces the whole attributes column rather than merging keys.
type fakeAtomDevices struct {
	t           *testing.T
	entities    map[string]Entity
	updateCalls int
}

func newFakeAtomDevices(t *testing.T, seed ...Entity) (*fakeAtomDevices, *httptest.Server) {
	t.Helper()
	fa := &fakeAtomDevices{t: t, entities: map[string]Entity{}}
	for _, e := range seed {
		fa.entities[e.ID] = e
	}
	srv := httptest.NewServer(http.HandlerFunc(fa.handle))
	t.Cleanup(srv.Close)
	return fa, srv
}

func (fa *fakeAtomDevices) handle(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(fa.t, r)
	switch {
	case strings.Contains(payload.Query, "mutation UpdateEntity"):
		fa.updateCalls++
		id, _ := payload.Variables["id"].(string)
		existing, ok := fa.entities[id]
		if !ok {
			_ = writeJSON(w, map[string]any{
				"errors": []map[string]string{{"message": "entity not found"}},
			})
			return
		}
		input, _ := payload.Variables["input"].(map[string]any)
		if attrs, ok := input["attributes"].(map[string]any); ok {
			existing.Attributes = Attributes(attrs)
		}
		fa.entities[id] = existing
		_ = writeJSON(w, map[string]any{"data": map[string]any{"updateEntity": deviceEntityJSON(existing)}})

	case strings.Contains(payload.Query, "query Entities("):
		contains, hasFilter := payload.Variables["attributesContains"].(map[string]any)
		if !hasFilter {
			fa.t.Fatalf("entities query ran without an attributesContains filter: %+v", payload.Variables)
		}
		kind, _ := payload.Variables["kind"].(string)
		var items []map[string]any
		for _, e := range fa.entities {
			if kind != "" && e.Kind != kind {
				continue
			}
			if entityMatchesContains(e, contains) {
				items = append(items, deviceEntityJSON(e))
			}
		}
		_ = writeJSON(w, map[string]any{
			"data": map[string]any{"entities": map[string]any{"items": items, "total": len(items)}},
		})

	case strings.Contains(payload.Query, "query Entity("):
		id, _ := payload.Variables["id"].(string)
		e, ok := fa.entities[id]
		if !ok {
			_ = writeJSON(w, map[string]any{
				"errors": []map[string]string{{"message": "entity not found"}},
			})
			return
		}
		_ = writeJSON(w, map[string]any{"data": map[string]any{"entity": deviceEntityJSON(e)}})

	default:
		fa.t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
	}
}

func deviceEntityJSON(e Entity) map[string]any {
	attrs := map[string]any(e.Attributes)
	if attrs == nil {
		attrs = map[string]any{}
	}
	return map[string]any{
		"id":         e.ID,
		"kind":       e.Kind,
		"name":       e.Name,
		"tenant_id":  e.TenantID,
		"status":     e.Status,
		"attributes": attrs,
	}
}

// entityMatchesContains reimplements Atom's JSONB containment for the fake
// server: every key must be present, list values match by element containment.
func entityMatchesContains(e Entity, contains map[string]any) bool {
	for key, want := range contains {
		got, ok := e.Attributes[key]
		if !ok {
			return false
		}
		wantList := toAnySlice(want)
		if wantList == nil {
			if got != want {
				return false
			}
			continue
		}
		gotList := toAnySlice(got)
		for _, w := range wantList {
			found := false
			for _, g := range gotList {
				if g == w {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func toAnySlice(v any) []any {
	switch vv := v.(type) {
	case []any:
		return vv
	case []string:
		out := make([]any, len(vv))
		for i, s := range vv {
			out[i] = s
		}
		return out
	default:
		return nil
	}
}

func TestSetDeviceGatewaysWritesWhenExpectedCurrentMatches(t *testing.T) {
	fa, srv := newFakeAtomDevices(t, Entity{
		ID:   "device-1",
		Kind: atomKindDevice,
		Attributes: Attributes{
			"source":   "magistrala",
			"gateways": []string{"gw-1", "gw-2"},
		},
	})

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-3"}, []string{"gw-2", "gw-1"})
	if err != nil {
		t.Fatalf("set device gateways failed: %v", err)
	}
	if fa.updateCalls != 1 {
		t.Fatalf("expected exactly one write, got %d", fa.updateCalls)
	}

	updated := fa.entities["device-1"]
	got := attrStrings(updated.Attributes, atomAttributeGateways)
	if !sameStringSet(got, []string{"gw-3"}) {
		t.Fatalf("unexpected gateways after write: %+v", got)
	}
	if updated.Attributes["source"] != "magistrala" {
		t.Fatalf("write must preserve unrelated attributes, got: %+v", updated.Attributes)
	}
}

func TestSetDeviceGatewaysConflictWhenExpectedCurrentMismatches(t *testing.T) {
	fa, srv := newFakeAtomDevices(t, Entity{
		ID:         "device-1",
		Kind:       atomKindDevice,
		Attributes: Attributes{"gateways": []string{"gw-1"}},
	})

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-9"}, []string{"gw-2"})
	if !errors.Is(err, ErrGatewaysConflict) {
		t.Fatalf("expected ErrGatewaysConflict, got %v", err)
	}
	if fa.updateCalls != 0 {
		t.Fatalf("conflict must not write, got %d update calls", fa.updateCalls)
	}

	unchanged := attrStrings(fa.entities["device-1"].Attributes, atomAttributeGateways)
	if !sameStringSet(unchanged, []string{"gw-1"}) {
		t.Fatalf("device gateways must be unchanged after a conflict, got: %+v", unchanged)
	}
}

func TestSetDeviceGatewaysExpectedCurrentIsOrderIndependent(t *testing.T) {
	_, srv := newFakeAtomDevices(t, Entity{
		ID:         "device-1",
		Kind:       atomKindDevice,
		Attributes: Attributes{"gateways": []string{"gw-1", "gw-2"}},
	})

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	// expectedCurrent is given in the opposite order to how it is stored.
	if err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-3"}, []string{"gw-2", "gw-1"}); err != nil {
		t.Fatalf("expected reordered expectedCurrent to match, got: %v", err)
	}
}

func TestDeviceGatewaysDropsStaleGateway(t *testing.T) {
	_, srv := newFakeAtomDevices(t,
		Entity{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-live", "gw-deleted"}},
		},
		Entity{ID: "gw-live", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		// gw-deleted intentionally not seeded: it no longer resolves.
	)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.DeviceGateways(context.Background(), "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}
	if !sameStringSet(got, []string{"gw-live"}) {
		t.Fatalf("expected stale gateway to be dropped, got: %+v", got)
	}
}

func TestGatewayDevicesFiltersByAttributesContains(t *testing.T) {
	_, srv := newFakeAtomDevices(t,
		Entity{ID: "device-1", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-1"}}},
		Entity{ID: "device-2", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-2"}}},
		Entity{ID: "device-3", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-1", "gw-2"}}},
		Entity{ID: "human-1", Kind: atomKindHuman, Attributes: Attributes{"gateways": []string{"gw-1"}}},
	)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.GatewayDevices(context.Background(), "gw-1", Query{})
	if err != nil {
		t.Fatalf("gateway devices failed: %v", err)
	}
	if got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("unexpected result count: %+v", got)
	}
	ids := map[string]bool{}
	for _, item := range got.Items {
		ids[item.ID] = true
	}
	if !ids["device-1"] || !ids["device-3"] || ids["device-2"] {
		t.Fatalf("unexpected devices returned: %+v", ids)
	}
}

func TestGatewayDevicesPreservesExistingGatewaysContainsFilter(t *testing.T) {
	_, srv := newFakeAtomDevices(t,
		Entity{ID: "device-1", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-1"}}},
		Entity{ID: "device-2", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-2"}}},
		Entity{ID: "device-3", Kind: atomKindDevice, Attributes: Attributes{"gateways": []string{"gw-1", "gw-2"}}},
	)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.GatewayDevices(context.Background(), "gw-1", Query{
		AttributesContains: map[string]any{atomAttributeGateways: []string{"gw-2"}},
	})
	if err != nil {
		t.Fatalf("gateway devices failed: %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].ID != "device-3" {
		t.Fatalf("expected only the device containing both gateways, got: %+v", got)
	}
}

func TestDeviceGatewaysRoundTripsZeroOneAndManyGateways(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{ID: "device-1", Kind: atomKindDevice, Attributes: Attributes{"source": "magistrala"}},
		Entity{ID: "gw-a", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-b", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-c", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)
	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	ctx := context.Background()

	// 0 gateways.
	current, err := client.DeviceGateways(ctx, "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}
	if len(current) != 0 {
		t.Fatalf("expected no gateways initially, got: %+v", current)
	}

	// 1 gateway.
	if err := client.SetDeviceGateways(ctx, "device-1", []string{"gw-a"}, current); err != nil {
		t.Fatalf("set device gateways (1) failed: %v", err)
	}
	current, err = client.DeviceGateways(ctx, "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}
	if !sameStringSet(current, []string{"gw-a"}) {
		t.Fatalf("expected exactly gw-a, got: %+v", current)
	}

	// 3 gateways.
	if err := client.SetDeviceGateways(ctx, "device-1", []string{"gw-a", "gw-b", "gw-c"}, current); err != nil {
		t.Fatalf("set device gateways (3) failed: %v", err)
	}
	current, err = client.DeviceGateways(ctx, "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}
	if !sameStringSet(current, []string{"gw-a", "gw-b", "gw-c"}) {
		t.Fatalf("expected all three gateways, got: %+v", current)
	}
	if fa.updateCalls != 2 {
		t.Fatalf("expected exactly two writes across the round trip, got %d", fa.updateCalls)
	}
}
