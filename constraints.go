// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package magistrala

type Operation int

const (
	Create Operation = iota
)

// Constraints specifies an API for obtaining entity constraints.
type Constraints interface {
	// Constraints returns constraints for the entities.
	CheckLimits(operation Operation, currentValue uint64) error
}
