// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package readers

import "time"

// DeviceViewDefaultWindow bounds an MG-15 observed-device query, and is also
// the width DefaultTimeWindow falls back to when a caller supplies neither
// bound. `GROUP BY device_id` (or publisher) with no time bound at all is a
// full, unbounded partition scan on a busy channel — this keeps that form
// from being the easy one to call by accident, on both the HTTP and gRPC
// reader APIs.
//
// The window is expressed in the stored-time unit this package's built-in
// senml/json transformers use, which is Unix nanoseconds: they run every
// absolute time through transformers.ToUnixNano before the writers persist
// it, and PageMetadata.From/To and DeviceStat.LastSeen use that same
// representation. On a deployment storing any other unit a caller relying on
// the default would get an empty roster for the last 24h; supply from/to
// explicitly in your own unit.
const DeviceViewDefaultWindow = 24 * time.Hour

// DefaultTimeWindow fills in whichever of the two bounds the caller left at
// zero so the resulting query is always bounded to DeviceViewDefaultWindow:
//   - neither bound supplied: last 24h, [now-24h, now]
//   - only to supplied:        [to-24h, to]
//   - only from supplied:      [from, from+24h]
//
// The bounds are Unix nanoseconds, matching DeviceViewDefaultWindow's unit.
// A caller who supplies both bounds gets exactly what they asked for,
// unmodified. Shared by the HTTP and gRPC reader APIs so the window
// semantics can only be changed in one place.
func DefaultTimeWindow(from, to float64) (float64, float64) {
	window := float64(DeviceViewDefaultWindow.Nanoseconds())
	switch {
	case from == 0 && to == 0:
		now := float64(time.Now().UnixNano())
		return now - window, now
	case from == 0:
		return to - window, to
	case to == 0:
		return from, from + window
	default:
		return from, to
	}
}
