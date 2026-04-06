// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

// Status represents Group status.
type Status uint8

// Possible Group status values.
const (
	// EnabledStatus represents enabled Group.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Group.
	DisabledStatus
	// DeletedStatus represents deleted Group.
	DeletedStatus

	// AllStatus is used for querying purposes to list groups irrespective
	// of their status - both active and inactive. It is never stored in the
	// database as the actual Group status and should always be the largest
	// value in this enumeration.
	AllStatus
)

// String representation of the possible status values.
const (
	Disabled = "disabled"
	Enabled  = "enabled"
	Deleted  = "deleted"
	All      = "all"
	Unknown  = "unknown"
)

// String converts group status to string literal.
func (s Status) String() string {
	switch s {
	case DisabledStatus:
		return Disabled
	case EnabledStatus:
		return Enabled
	case DeletedStatus:
		return Deleted
	case AllStatus:
		return All
	default:
		return Unknown
	}
}

// ToStatus converts string value to a valid Group status.
func ToStatus(status string) (Status, error) {
	switch status {
	case Disabled:
		return DisabledStatus, nil
	case Enabled:
		return EnabledStatus, nil
	case Deleted:
		return DeletedStatus, nil
	case All:
		return AllStatus, nil
	}
	return Status(0), svcerr.ErrInvalidStatus
}

// Custom Marshaller for Status.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for Status.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
