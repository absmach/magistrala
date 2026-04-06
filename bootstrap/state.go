// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import "strconv"

const (
	// Inactive Client is created, but not able to exchange messages using Magistrala.
	Inactive State = iota
	// Active Client is created, configured, and whitelisted.
	Active
)

// State represents corresponding Magistrala Client state. The possible Config States
// as well as description of what that State represents are given in the table:
// | State    | What it means                                                                  |
// |----------+--------------------------------------------------------------------------------|
// | Inactive | Client is created, but isn't able to communicate over Magistrala                  |
// | Active   | Client is able to communicate using Magistrala                                    |.
type State int

// String returns string representation of State.
func (s State) String() string {
	return strconv.Itoa(int(s))
}
