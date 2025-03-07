// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

// Status represents User status.
type Status uint8

// Possible User status values.
const (
	// EnabledStatus represents enabled User.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled User.
	DisabledStatus
	// DeletedStatus represents a user that will be deleted.
	DeletedStatus

	// AllStatus is used for querying purposes to list users irrespective
	// of their status - both enabled and disabled. It is never stored in the
	// database as the actual User status and should always be the largest
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

// String converts user/group status to string literal.
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

// ToStatus converts string value to a valid User/Group status.
func ToStatus(status string) (Status, error) {
	switch status {
	case "", Enabled:
		return EnabledStatus, nil
	case Disabled:
		return DisabledStatus, nil
	case Deleted:
		return DeletedStatus, nil
	case All:
		return AllStatus, nil
	}
	return Status(0), svcerr.ErrInvalidStatus
}

// Custom Marshaller for Uesr/Groups.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for User/Groups.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
