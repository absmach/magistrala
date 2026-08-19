// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ErrGatewaysConflict means the device's gateways attribute no longer
// matches expectedCurrent — the caller's view was stale, so nothing was written.
var ErrGatewaysConflict = errors.New("atom: device gateways changed since last read")

// SetDeviceGateways replaces a device's entire gateway list, preserving its
// other attributes. expectedCurrent must match what DeviceGateways would
// return right now (order independent) or it fails with ErrGatewaysConflict
// — Atom has no compare-and-swap, so this only narrows the race window, not
// closes it.
//
// The comparison is against the same live, since-deleted-gateways-dropped
// view DeviceGateways returns, not the raw stored attribute: every caller
// reads expectedCurrent from DeviceGateways, so once a declared gateway is
// deleted the raw attribute and that live view disagree permanently, and
// comparing against the raw one would fail this call forever with no way to
// retry into success.
func (c *Client) SetDeviceGateways(ctx context.Context, deviceID string, gatewayIDs, expectedCurrent []string) error {
	device, err := c.GetEntity(ctx, deviceID)
	if err != nil {
		return err
	}

	current, err := c.liveGateways(ctx, attrStrings(device.Attributes, atomAttributeGateways))
	if err != nil {
		return err
	}
	if !sameStringSet(current, expectedCurrent) {
		return ErrGatewaysConflict
	}

	if err := c.markGateways(ctx, gatewayIDs); err != nil {
		return err
	}

	// Re-read rather than reuse the snapshot taken above (P2): a device can
	// name itself as one of its own gateways, in which case markGateways
	// just wrote deviceID's own is_gateway through a separate call. Writing
	// back the pre-mark snapshot here would silently discard that -- the
	// same instant, no unrelated write in between required to trigger it.
	// This still cannot close the race against a genuinely concurrent
	// edit — Atom has no compare-and-swap, which is exactly why this
	// function's own expectedCurrent check only narrows that window rather
	// than closing it — but it removes the self-inflicted case where this
	// call clobbers its own prior write.
	device, err = c.GetEntity(ctx, deviceID)
	if err != nil {
		return err
	}

	attrs := cloneAttributes(device.Attributes)
	attrs[atomAttributeGateways] = gatewayIDs
	_, err = c.UpdateEntity(ctx, deviceID, Entity{Attributes: attrs})
	return err
}

// markGateways sets attributes.is_gateway on every id in gatewayIDs that
// does not already have it. is_gateway is otherwise entirely manual —
// operators set it themselves when creating or updating a device (see
// cli/devices.go) — so without this, the two facts that identify a gateway
// can drift apart: a device can be named in another device's gateways list,
// making it a real relay, while its own is_gateway attribute stays unset
// because whoever created it never flagged it. readAuthorizer's R1 scope
// leg trusts a held device's publisher identity unconditionally whenever
// is_gateway is false, so that drift reopened finding 02's cross-tenant
// leak for any gateway an operator forgot to flag (Q1).
//
// This only ever sets the flag, never clears it: SetDeviceGateways is the
// only place in this repo that establishes the relationship, so it is the
// only place that can reliably keep it in sync, and always setting rather
// than reconciling both directions means a device dropped from every
// gateways list simply keeps a flag it no longer strictly needs — the safe
// direction to be wrong in, unlike the leak this closes.
//
// Reads gatewayIDs in one batched round trip via batchEntities rather than
// one GetEntity per id (P1) — the exact per-id shape round 2's finding 15
// retired from DeviceGateways below, reappearing here. An id neither
// resolved nor readable is treated the same as liveGateways treats it
// elsewhere: silently skipped if it simply does not (yet, or any longer)
// exist, but a hard error if it merely could not be read this time — "could
// not tell" must not be read as "nothing to mark", the same distinction
// entitiesExist exists to preserve.
func (c *Client) markGateways(ctx context.Context, gatewayIDs []string) error {
	raw, unreadable, err := c.batchEntities(ctx, gatewayIDs, "EntityDeviceInfo", "id external_id: externalId attributes")
	if err != nil {
		return err
	}
	if len(unreadable) > 0 {
		return fmt.Errorf("atom: could not determine gateway status of %d of %d gateways (e.g. %s): %w",
			len(unreadable), len(gatewayIDs), firstUnreadable(gatewayIDs, unreadable), errEntitiesUnreadable)
	}

	for _, id := range gatewayIDs {
		data, ok := raw[id]
		if !ok {
			// Not yet resolvable -- e.g. a forward-declared id, or one
			// concurrently deleted. SetDeviceGateways has always let a
			// device's own gateways attribute name an id that does not
			// (yet, or any longer) resolve, since liveGateways drops it on
			// the next read regardless; best-effort marking must not turn
			// that into a hard failure it never was before.
			continue
		}
		var gateway Entity
		if err := json.Unmarshal(data, &gateway); err != nil {
			return err
		}
		if isGateway, _ := gateway.Attributes[atomAttributeIsGateway].(bool); isGateway {
			continue
		}

		attrs := cloneAttributes(gateway.Attributes)
		attrs[atomAttributeIsGateway] = true
		if _, err := c.UpdateEntity(ctx, id, Entity{Attributes: attrs}); err != nil {
			return err
		}
	}
	return nil
}

// DeviceGateways returns the gateway IDs a device declares, dropping any
// that point at a since-deleted gateway (nothing else sweeps those).
func (c *Client) DeviceGateways(ctx context.Context, deviceID string) ([]string, error) {
	device, err := c.GetEntity(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return c.liveGateways(ctx, attrStrings(device.Attributes, atomAttributeGateways))
}

// liveGateways narrows a device's declared gateway ids down to the ones
// that still exist, in one batch of round trips via entitiesExist rather
// than one GetEntity call per declared gateway.
func (c *Client) liveGateways(ctx context.Context, declared []string) ([]string, error) {
	existing, err := c.entitiesExist(ctx, declared)
	if err != nil {
		return nil, err
	}

	live := make([]string, 0, len(declared))
	for _, gatewayID := range declared {
		if _, ok := existing[gatewayID]; ok {
			live = append(live, gatewayID)
		}
	}
	return live, nil
}

// GatewayDevices lists devices whose gateways attribute contains gatewayID.
// Any AttributesContains already set on q is preserved alongside it.
func (c *Client) GatewayDevices(ctx context.Context, gatewayID string, q Query) (EntityList, error) {
	filter := make(map[string]any, len(q.AttributesContains)+1)
	for k, v := range q.AttributesContains {
		filter[k] = v
	}
	filter[atomAttributeGateways] = mergeContainsString(filter[atomAttributeGateways], gatewayID)
	q.Kind = atomKindDevice
	q.AttributesContains = filter

	return c.ListEntities(ctx, q)
}

func mergeContainsString(existing any, value string) any {
	switch v := existing.(type) {
	case nil:
		return []string{value}
	case string:
		if v == value {
			return []string{value}
		}
		return []string{v, value}
	case []string:
		out := append([]string(nil), v...)
		for _, item := range out {
			if item == value {
				return out
			}
		}
		return append(out, value)
	case []any:
		out := append([]any(nil), v...)
		for _, item := range out {
			if item, ok := item.(string); ok && item == value {
				return out
			}
		}
		return append(out, value)
	default:
		return []any{v, value}
	}
}

// attrStrings reads a []string-valued attribute, accepting both []string and
// the []any that JSON decoding actually produces.
func attrStrings(attrs Attributes, key string) []string {
	if attrs == nil {
		return nil
	}
	switch v := attrs[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// cloneAttributes returns a shallow, mutable, never-nil copy of attrs.
func cloneAttributes(attrs Attributes) Attributes {
	out := make(Attributes, len(attrs)+1)
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

// sameStringSet reports whether a and b hold the same strings, ignoring
// order.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := append([]string(nil), a...)
	sortedB := append([]string(nil), b...)
	sort.Strings(sortedA)
	sort.Strings(sortedB)
	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}
