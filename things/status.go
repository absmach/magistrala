// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

// Status represents Thing status.
type Status uint8

// Possible Thing status values.
const (
	// EnabledStatus represents enabled Thing.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Thing.
	DisabledStatus
	// DeletedStatus represents a client that will be deleted.
	DeletedStatus

	// AllStatus is used for querying purposes to list clients irrespective
	// of their status - both enabled and disabled. It is never stored in the
	// database as the actual Thing status and should always be the largest
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

// String converts client/group status to string literal.
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

// ToStatus converts string value to a valid Thing/Group status.
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

// Custom Marshaller for Thing/Groups.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (thing Thing) MarshalJSON() ([]byte, error) {
	type Alias Thing
	return json.Marshal(&struct {
		Alias
		Status string `json:"status,omitempty"`
	}{
		Alias:  (Alias)(thing),
		Status: thing.Status.String(),
	})
}

// Custom Unmarshaler for Thing/Groups.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
