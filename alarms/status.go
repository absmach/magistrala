// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

type Status uint8

const (
	ActiveStatus Status = iota
	ClearedStatus

	// AllStatus is used for querying purposes to list alarms irrespective
	// of their status. It is never stored in the database as the actual
	// Alarm status and should always be the largest value in this enumeration.
	AllStatus
)

const (
	Active  = "active"
	Cleared = "cleared"
	Unknown = "unknown"
	All     = "all"
)

// String converts alarm status to string literal.
func (s Status) String() string {
	switch s {
	case ActiveStatus:
		return Active
	case ClearedStatus:
		return Cleared
	default:
		return Unknown
	}
}

// ToStatus converts string value to a valid Alarm status.
func ToStatus(status string) (Status, error) {
	switch strings.ToLower(status) {
	case Active:
		return ActiveStatus, nil
	case Cleared:
		return ClearedStatus, nil
	case All:
		return AllStatus, nil
	default:
		return Status(0), svcerr.ErrInvalidStatus
	}
}

// Custom Marshaller for Alarm.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for Alarm.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val

	return err
}
