// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import "strconv"

const (
	// Inactive Thing is created, but not able to exchange messages using Magistrala.
	Inactive State = iota
	// Active Thing is created, configured, and whitelisted.
	Active
)

// State represents corresponding Magistrala Thing state. The possible Config States
// as well as description of what that State represents are given in the table:
// | State    | What it means                                                                  |
// |----------+--------------------------------------------------------------------------------|
// | Inactive | Thing is created, but isn't able to communicate over Magistrala                  |
// | Active   | Thing is able to communicate using Magistrala                                    |.
type State int

// String returns string representation of State.
func (s State) String() string {
	return strconv.Itoa(int(s))
}
