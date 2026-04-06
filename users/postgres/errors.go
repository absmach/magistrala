// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import "github.com/absmach/magistrala/pkg/errors"

var _ errors.Mapper = (*duplicateErrors)(nil)

type duplicateErrors struct{}

// GetError maps constraint names to known errors.
func (d duplicateErrors) GetError(constraint string) (error, bool) {
	switch constraint {
	case "clients_email_key":
		return errors.NewRequestError("email id already registered"), true
	case "clients_username_key":
		return errors.NewRequestError("username not available"), true
	default:
		return nil, false
	}
}

func NewDuplicateErrors() errors.Mapper {
	return duplicateErrors{}
}
