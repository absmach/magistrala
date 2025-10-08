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
func resolveTimezone(s string) (*time.Location, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(s)
	if err != nil {
		return time.UTC, err
	}
	return loc, nil
}
