// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import "errors"

var (
	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = New("malformed entity specification")

	// ErrUnsupportedContentType indicates invalid content type.
	ErrUnsupportedContentType = errors.New("invalid content type")

	// ErrUnidentified indicates unidentified error.
	ErrUnidentified = errors.New("unidentified error")

	// ErrEmptyPath indicates empty file path.
	ErrEmptyPath = errors.New("empty file path")

	// ErrStatusAlreadyAssigned indicated that the client or group has already been assigned the status.
	ErrStatusAlreadyAssigned = errors.New("status already assigned")

	// ErrRollbackTx indicates failed to rollback transaction.
	ErrRollbackTx = errors.New("failed to rollback transaction")

	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = errors.New("failed to perform authentication over the entity")

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = errors.New("failed to perform authorization over the entity")
)
