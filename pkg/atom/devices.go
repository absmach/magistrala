// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
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

	attrs := cloneAttributes(device.Attributes)
	attrs[atomAttributeGateways] = gatewayIDs
	_, err = c.UpdateEntity(ctx, deviceID, Entity{Attributes: attrs})
	return err
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
// that still exist.
func (c *Client) liveGateways(ctx context.Context, declared []string) ([]string, error) {
	live := make([]string, 0, len(declared))
	for _, gatewayID := range declared {
		if _, err := c.GetEntity(ctx, gatewayID); err != nil {
			if IsNotFound(err) {
				continue
			}
			return nil, err
		}
		live = append(live, gatewayID)
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
