// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package magistrala

type Constraints struct {
	Users        uint32 `toml:"users"`
	Domains      uint32 `toml:"domains"`
	Things       uint32 `toml:"things"`
	Groups       uint32 `toml:"groups"`
	Channels     uint32 `toml:"channels"`
	MsgRateLimit uint32 `toml:"msg_rate_limit"`
}

// ConstraintsProvider specifies an API for obtaining entity constraints.
type ConstraintsProvider interface {
	// Constraints returns constraints for the entities.
	Constraints() (Constraints, error)
}
