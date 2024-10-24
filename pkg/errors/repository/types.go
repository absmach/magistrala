// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package repository

import "github.com/absmach/magistrala/pkg/errors"

// Wrapper for Repository errors.
var (
	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = errors.New("failed to create entity in the db")

	// ErrViewEntity indicates error in viewing entity or entities.
	ErrViewEntity = errors.New("view entity failed")

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = errors.New("update entity failed")

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = errors.New("failed to remove entity")

	// ErrFailedOpDB indicates a failure in a database operation.
	ErrFailedOpDB = errors.New("operation on db element failed")

	// ErrFailedToRetrieveAllGroups failed to retrieve groups.
	ErrFailedToRetrieveAllGroups = errors.New("failed to retrieve all groups")

	ErrRoleMigration = errors.New("role migration initialization failed")

	// ErrMissingNames indicates missing first and last names.
	ErrMissingNames = errors.New("missing first or last name")
)
