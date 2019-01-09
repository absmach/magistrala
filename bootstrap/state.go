//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package bootstrap

import "strconv"

// State represents corresponding Mainflux Thing state. The possible Config States
// as well as description of what that State represents are given in the table:
// | State    | What it means 		                                                           |
// |----------+--------------------------------------------------------------------------------|
// | Inactive | Thing is created, but isn't able to communicate over Mainflux 				   |
// | Active   | Thing is able to communicate using Mainflux                                    |
type State int

// String returns string representation of State.
func (s State) String() string {
	return strconv.Itoa(int(s))
}
