// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations

import (
	"encoding/json"
	"strings"

	"github.com/absmach/magistrala/internal/apiutil"
)

// State represents invitation state.
type State uint8

const (
	All      State = iota // All is used for querying purposes to list invitations irrespective of their state - both pending and accepted.
	Pending               // Pending is the state of an invitation that has not been accepted yet.
	Accepted              // Accepted is the state of an invitation that has been accepted.
)

// String representation of the possible state values.
const (
	all      = "all"
	pending  = "pending"
	accepted = "accepted"
	unknown  = "unknown"
)

// String converts invitation state to string literal.
func (s State) String() string {
	switch s {
	case All:
		return all
	case Pending:
		return pending
	case Accepted:
		return accepted
	default:
		return unknown
	}
}

// ToState converts string value to a valid invitation state.
func ToState(status string) (State, error) {
	switch status {
	case all:
		return All, nil
	case pending:
		return Pending, nil
	case accepted:
		return Accepted, nil
	}

	return State(0), apiutil.ErrInvitationState
}

func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for Client/Groups.
func (s *State) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToState(str)
	*s = val
	return err
}
