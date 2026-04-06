// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import "github.com/absmach/magistrala/pkg/errors"

var _ errors.Mapper = (*duplicateErrors)(nil)

var errCyclicParentGroup = errors.NewRequestError("cyclic parent, group is parent of requested group")

type duplicateErrors struct{}

// GetError maps constraint names to known errors.
func (d duplicateErrors) GetError(constraint string) (error, bool) {
	switch constraint {
	case "groups_pkey":
		return errors.NewRequestError("group id already exists"), true
	default:
		return nil, false
	}
}

func NewDuplicateErrors() errors.Mapper {
	return duplicateErrors{}
}
