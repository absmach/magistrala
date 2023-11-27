// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/absmach/magistrala/internal/apiutil"
)

// Status represents Client status.
type Status uint8

// Possible Client status values.
const (
	// EnabledStatus represents enabled Client.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Client.
	DisabledStatus

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
	All      = "all"
	Unknown  = "unknown"
)

// ErrStatusAlreadyAssigned indicated that the client or group has already been assigned the status.
var ErrStatusAlreadyAssigned = errors.New("status already assigned")

// String converts client/group status to string literal.
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

// ToStatus converts string value to a valid Client/Group status.
func ToStatus(status string) (Status, error) {
	switch status {
	case "", Enabled:
		return EnabledStatus, nil
	case Disabled:
		return DisabledStatus, nil
	case All:
		return AllStatus, nil
	}
	return Status(0), apiutil.ErrInvalidStatus
}

// Custom Marshaller for Client/Groups.
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

// Custom Unmarshaler for Client/Groups.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
