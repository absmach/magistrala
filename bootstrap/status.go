// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"encoding/json"
	"strconv"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

// Status represents bootstrap enrollment availability.
type Status uint8

// Possible bootstrap enrollment statuses.
const (
	EnabledStatus Status = iota
	DisabledStatus
	// AllStatus is used for querying purposes to list configs irrespective
	// of their status. It is never stored in the database.
	AllStatus
)

// String representation of bootstrap status values.
const (
	Disabled = "disabled"
	Enabled  = "enabled"
	All      = "all"
	Unknown  = "unknown"
)

// Backward-compatible aliases kept while callers move off the old names.
const (
	Inactive = DisabledStatus
	Active   = EnabledStatus
)

// String returns string representation of Status.
func (s Status) String() string {
	switch s {
	case DisabledStatus:
		return Disabled
	case EnabledStatus:
		return Enabled
	case AllStatus:
		return All
	default:
		return Unknown
	}
}

// ToStatus converts a string or legacy numeric string value to Status.
func ToStatus(status string) (Status, error) {
	switch strings.ToLower(status) {
	case "", Enabled, "0":
		return EnabledStatus, nil
	case Disabled, "1":
		return DisabledStatus, nil
	case All:
		return AllStatus, nil
	}
	return Status(0), svcerr.ErrInvalidStatus
}

// MarshalJSON renders bootstrap status as a string literal.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON accepts both string and legacy numeric bootstrap statuses.
func (s *Status) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if data[0] != '"' {
		var n int
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		parsed, err := ToStatus(strconv.Itoa(n))
		if err != nil {
			return err
		}
		*s = parsed
		return nil
	}

	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	parsed, err := ToStatus(status)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
