// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package channels

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

// Status represents Channel status.
type Status uint8

// Possible Channel status values.
const (
	// EnabledStatus represents enabled Channel.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Channel.
	DisabledStatus
	// DeletedStatus represents deleted Channel.
	DeletedStatus

	// AllStatus is used for querying purposes to list channels irrespective
	// of their status - both active and inactive. It is never stored in the
	// database as the actual Channel status and should always be the largest
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

// String converts Channel status to string literal.
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

// ToStatus converts string value to a valid Channel status.
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

func (channel Channel) MarshalJSON() ([]byte, error) {
	type Alias Channel
	return json.Marshal(&struct {
		Alias
		Status string `json:"status,omitempty"`
	}{
		Alias:  (Alias)(channel),
		Status: channel.Status.String(),
	})
}

// Custom Unmarshaler for Status.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
