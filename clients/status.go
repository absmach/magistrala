// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

// Status represents Client status.
type Status uint8

// Possible Client status values.
const (
	// EnabledStatus represents enabled Client.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Client.
	DisabledStatus
	// DeletedStatus represents a client that will be deleted.
	DeletedStatus

	// AllStatus is used for querying purposes to list clients irrespective
	// of their status - both enabled and disabled. It is never stored in the
	// database as the actual Client status and should always be the largest
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

// ToStatus converts string value to a valid Client status.
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

// Custom Marshaller for Client.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (client Client) MarshalJSON() ([]byte, error) {
	type Alias Client
	return json.Marshal(&struct {
		Alias
		Status string `json:"status,omitempty"`
	}{
		Alias:  (Alias)(client),
		Status: client.Status.String(),
	})
}

// Custom Unmarshaler for Client.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
