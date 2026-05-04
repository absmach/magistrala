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
	DisabledStatus Status = iota
	EnabledStatus
)

// String representation of bootstrap status values.
const (
	Disabled = "disabled"
	Enabled  = "enabled"
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
	default:
		return Unknown
	}
}

// ToStatus converts a string or legacy numeric string value to Status.
func ToStatus(status string) (Status, error) {
	switch strings.ToLower(status) {
	case Disabled, "0":
		return DisabledStatus, nil
	case Enabled, "1":
		return EnabledStatus, nil
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
