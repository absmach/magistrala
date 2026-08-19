// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
	"fmt"
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
	requests    int
	// unreadable ids come back null from EntityExistence with a path-attributed
	// error entry, as a permission failure or transient per-entity error would,
	// rather than a clean null — which is reserved for ids that genuinely do
	// not exist.
	unreadable map[string]struct{}
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
	fa.requests++
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

	case strings.Contains(payload.Query, "query EntityExistence("):
		data := map[string]any{}
		var errs []map[string]any
		for i := 0; ; i++ {
			raw, ok := payload.Variables[fmt.Sprintf("id%d", i)]
			if !ok {
				break
			}
			id, _ := raw.(string)
			alias := fmt.Sprintf("e%d", i)
			_, blocked := fa.unreadable[id]
			_, exists := fa.entities[id]
			switch {
			case blocked:
				data[alias] = nil
				errs = append(errs, map[string]any{"message": "forbidden", "path": []any{alias}})
			case exists:
				data[alias] = map[string]any{"id": id}
			default:
				data[alias] = nil
			}
		}
		body := map[string]any{"data": data}
		if len(errs) > 0 {
			body["errors"] = errs
		}
		_ = writeJSON(w, body)

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
	fa, srv := newFakeAtomDevices(t,
		Entity{
			ID:   "device-1",
			Kind: atomKindDevice,
			Attributes: Attributes{
				"source":   "magistrala",
				"gateways": []string{"gw-1", "gw-2"},
			},
		},
		Entity{ID: "gw-1", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-2", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)

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

// A regression test for Q1: is_gateway is otherwise entirely manual, so a
// device newly linked as a gateway can have no is_gateway attribute at all
// -- exactly the case that let readAuthorizer's R1 scope leg trust it as an
// unconditional self-publisher, reopening finding 02's cross-tenant leak.
// SetDeviceGateways must flag it itself rather than depend on the operator
// having remembered to.
func TestSetDeviceGatewaysFlagsNewlyLinkedGateway(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{ID: "device-1", Kind: atomKindDevice},
		Entity{ID: "gw-new", Kind: atomKindDevice, Attributes: Attributes{"source": "magistrala"}},
	)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	if err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-new"}, nil); err != nil {
		t.Fatalf("set device gateways failed: %v", err)
	}

	gw := fa.entities["gw-new"]
	isGateway, _ := gw.Attributes[atomAttributeIsGateway].(bool)
	if !isGateway {
		t.Fatalf("expected gw-new to be flagged is_gateway after being linked, got: %+v", gw.Attributes)
	}
	if gw.Attributes["source"] != "magistrala" {
		t.Fatalf("flagging is_gateway must preserve the gateway's other attributes, got: %+v", gw.Attributes)
	}
}

// A device already flagged is_gateway must not be rewritten -- only the
// device's own gateways attribute write should count.
func TestSetDeviceGatewaysSkipsAlreadyFlaggedGateway(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{ID: "device-1", Kind: atomKindDevice},
		Entity{ID: "gw-already", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	if err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-already"}, nil); err != nil {
		t.Fatalf("set device gateways failed: %v", err)
	}
	if fa.updateCalls != 1 {
		t.Fatalf("expected exactly one write (device-1's own gateways list), got %d", fa.updateCalls)
	}
}

// A gateway id that does not yet resolve must not block the write --
// SetDeviceGateways has always let a device's own gateways attribute name
// an id that does not (yet, or any longer) exist; best-effort flagging must
// not turn that into a new hard failure.
func TestSetDeviceGatewaysToleratesAnUnresolvableNewGateway(t *testing.T) {
	_, srv := newFakeAtomDevices(t, Entity{ID: "device-1", Kind: atomKindDevice})

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	if err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-not-yet-created"}, nil); err != nil {
		t.Fatalf("an unresolvable new gateway must not fail the write, got: %v", err)
	}
}

func TestSetDeviceGatewaysExpectedCurrentIsOrderIndependent(t *testing.T) {
	_, srv := newFakeAtomDevices(t,
		Entity{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-1", "gw-2"}},
		},
		Entity{ID: "gw-1", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-2", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)

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

// A regression test for R2: a gateway that could not be read (a permission
// failure, a transient per-entity error) must not be treated the same as one
// that was deleted. Before the fix, entitiesExist's tolerant decode dropped
// both alike, so DeviceGateways silently truncated the roster.
func TestDeviceGatewaysFailsWhenAGatewayIsUnreadable(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-live", "gw-unreadable"}},
		},
		Entity{ID: "gw-live", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-unreadable", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)
	fa.unreadable = map[string]struct{}{"gw-unreadable": {}}

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.DeviceGateways(context.Background(), "device-1")
	if err == nil {
		t.Fatalf("expected an unreadable gateway to surface as an error, got a clean result: %+v", got)
	}
	if !errors.Is(err, errEntitiesUnreadable) {
		t.Fatalf("expected errEntitiesUnreadable, got: %v", err)
	}
}

// A regression test for Q5: with more than one unreadable id, the error must
// name a specific, deterministic one -- the first in the caller's own
// declared order -- rather than whichever Go's map iteration happens to
// yield, which used to vary run to run and made the message impossible to
// correlate across logs or assert in a test.
func TestDeviceGatewaysUnreadableErrorNamesFirstDeclaredGateway(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-unreadable-a", "gw-unreadable-b"}},
		},
		Entity{ID: "gw-unreadable-a", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
		Entity{ID: "gw-unreadable-b", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)
	fa.unreadable = map[string]struct{}{"gw-unreadable-a": {}, "gw-unreadable-b": {}}

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	_, err := client.DeviceGateways(context.Background(), "device-1")
	if err == nil {
		t.Fatalf("expected an error")
	}
	if !strings.Contains(err.Error(), "gw-unreadable-a") {
		t.Fatalf("expected the error to name gw-unreadable-a, the first declared gateway, got: %v", err)
	}
}

// A read-modify-write caller (the CLI's gateway-set retry loop) must not be
// able to write back a truncated roster when DeviceGateways itself failed to
// resolve one of the declared gateways.
func TestSetDeviceGatewaysFailsWhenCurrentGatewayIsUnreadable(t *testing.T) {
	fa, srv := newFakeAtomDevices(t,
		Entity{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-unreadable"}},
		},
		Entity{ID: "gw-unreadable", Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}},
	)
	fa.unreadable = map[string]struct{}{"gw-unreadable": {}}

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-new"}, nil)
	if err == nil {
		t.Fatal("expected SetDeviceGateways to fail rather than write over an unresolved current gateway")
	}
	if fa.updateCalls != 0 {
		t.Fatalf("an unreadable current gateway must not write, got %d update calls", fa.updateCalls)
	}
}

// A regression test for the case the CLI's "gateways set changed since last
// read" retry loop could never actually retry into success: expectedCurrent,
// as read from DeviceGateways, drops a since-deleted declared gateway — so
// SetDeviceGateways comparing against the raw stored attribute instead of the
// same live view would report a conflict forever, on every retry.
func TestSetDeviceGatewaysSucceedsAfterDeclaredGatewayDeleted(t *testing.T) {
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
	current, err := client.DeviceGateways(context.Background(), "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}

	if err := client.SetDeviceGateways(context.Background(), "device-1", []string{"gw-live", "gw-new"}, current); err != nil {
		t.Fatalf("set device gateways failed using DeviceGateways' own view as expectedCurrent: %v", err)
	}
}

// DeviceGateways used to cost one GetEntity round trip per declared gateway
// purely to test existence. liveGateways now resolves them all in one
// batched EntityExistence request instead, so a device declaring many
// gateways costs a constant two round trips (the device itself, then the
// batch) rather than growing with the gateway count.
func TestDeviceGatewaysBatchesExistenceChecksInOneRoundTrip(t *testing.T) {
	seed := []Entity{
		{
			ID:         "device-1",
			Kind:       atomKindDevice,
			Attributes: Attributes{"gateways": []string{"gw-1", "gw-2", "gw-3", "gw-4", "gw-5"}},
		},
	}
	for _, gw := range []string{"gw-1", "gw-2", "gw-3", "gw-4"} {
		seed = append(seed, Entity{ID: gw, Kind: atomKindDevice, Attributes: Attributes{"is_gateway": true}})
	}
	// gw-5 intentionally not seeded: it no longer resolves.
	fa, srv := newFakeAtomDevices(t, seed...)

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.DeviceGateways(context.Background(), "device-1")
	if err != nil {
		t.Fatalf("device gateways failed: %v", err)
	}
	if !sameStringSet(got, []string{"gw-1", "gw-2", "gw-3", "gw-4"}) {
		t.Fatalf("expected the four live gateways, got: %+v", got)
	}
	if fa.requests != 2 {
		t.Fatalf("expected 2 round trips (device + one batched existence check), got %d", fa.requests)
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
