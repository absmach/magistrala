// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/absmach/magistrala/pkg/errors"

// Wrapper for Service errors.
var (
	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = errors.New("authentication error")

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = errors.New("failed to perform authorization over the entity")

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = errors.New("failed to create entity in the db")

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = errors.New("failed to remove entity")

	// ErrViewEntity indicates error in viewing entity or entities.
	ErrViewEntity = errors.New("view entity failed")

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = errors.New("update entity failed")

	// ErrUniqueID indicates an error in generating a unique ID.
	ErrUniqueID = errors.New("failed to generate unique identifier")

	// ErrInvalidStatus indicates an invalid status.
	ErrInvalidStatus = errors.New("invalid status")

	// ErrInvalidRole indicates that an invalid role.
	ErrInvalidRole = errors.New("invalid client role")

	// ErrInvalidPolicy indicates that an invalid policy.
	ErrInvalidPolicy = errors.New("invalid policy")
)
