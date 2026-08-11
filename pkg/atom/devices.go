// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
	"sort"
)

// ErrGatewaysConflict is returned by SetDeviceGateways when the device's
// current gateways attribute no longer matches expectedCurrent, so the
// caller's view of it is stale and the write was not performed.
var ErrGatewaysConflict = errors.New("atom: device gateways changed since last read")

// SetDeviceGateways replaces a device's entire gateway list.
//
// Atom's update_entity overwrites the whole attributes column rather than
// merging individual keys, so this reads the device's current attributes
// first and writes them back with only the gateways key changed - every
// other attribute on the device is preserved unmodified.
//
// expectedCurrent must match the gateway list last read for this device
// (order-independent), or the call fails with ErrGatewaysConflict instead of
// writing. Atom offers no compare-and-swap primitive, so there remains a
// window between the read below and the write; this check narrows it from
// "however long the caller took to decide" to one GraphQL round trip. It is
// not a substitute for real distributed locking, and does not attempt to be.
func (c *Client) SetDeviceGateways(ctx context.Context, deviceID string, gatewayIDs, expectedCurrent []string) error {
	device, err := c.GetEntity(ctx, deviceID)
	if err != nil {
		return err
	}

	current := attrStrings(device.Attributes, atomAttributeGateways)
	if !sameStringSet(current, expectedCurrent) {
		return ErrGatewaysConflict
	}

	attrs := cloneAttributes(device.Attributes)
	attrs[atomAttributeGateways] = gatewayIDs
	_, err = c.UpdateEntity(ctx, deviceID, Entity{Attributes: attrs})
	return err
}

// DeviceGateways returns the gateway IDs a device declares, resolving away
// any stale IDs pointing at gateways that no longer exist (or were deleted).
// Deleting a gateway leaves it referenced in the devices that named it -
// nothing sweeps those references - so this is where they get dropped.
func (c *Client) DeviceGateways(ctx context.Context, deviceID string) ([]string, error) {
	device, err := c.GetEntity(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	declared := attrStrings(device.Attributes, atomAttributeGateways)
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

// GatewayDevices lists devices whose gateways attribute contains gatewayID,
// via attributesContains JSONB array containment. Any AttributesContains
// already set on q is preserved alongside the gateways filter.
func (c *Client) GatewayDevices(ctx context.Context, gatewayID string, q Query) (EntityList, error) {
	filter := make(map[string]any, len(q.AttributesContains)+1)
	for k, v := range q.AttributesContains {
		filter[k] = v
	}
	filter[atomAttributeGateways] = []string{gatewayID}
	q.Kind = atomKindDevice
	q.AttributesContains = filter

	return c.ListEntities(ctx, q)
}

// attrStrings reads a []string-valued attribute. Values come back from the
// GraphQL client as []any (each element a string), since Attributes decodes
// from JSON, but a []string is accepted too so callers can build one without
// a round trip.
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

// cloneAttributes returns a shallow, mutable copy of attrs, never nil, so
// callers can add or overwrite a key without affecting the source map or
// panicking on a nil map.
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
