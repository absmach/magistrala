// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"strings"
	"time"
)

// resolveTimezone returns a *time.Location from a user-provided IANA timezone name.
// Supported inputs:
// - IANA names (e.g., "Europe/Paris", "America/New_York").
// - Empty string defaults to UTC.
func resolveTimezone(s string) *time.Location {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(s); err == nil {
		return loc
	}
	return time.UTC
}
