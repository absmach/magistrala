// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/absmach/supermq/pkg/errors"

var (
	// ErrEmptySerialNo indicates that the serial number is empty.
	ErrEmptySerialNo = errors.New("empty serial number provided")

	// ErrEmptyTTL indicates that the TTL is empty.
	ErrEmptyTTL = errors.New("empty TTL provided")

	// ErrEmptyToken indicates that the token is empty.
	ErrEmptyToken = errors.New("empty token provided")

	// ErrEmptyList indicates that entity data is empty.
	ErrEmptyList = errors.New("empty list provided")

	// ErrMissingEntityID indicates missing entity ID.
	ErrMissingEntityID = errors.New("missing entity ID")

	// ErrMissingCommonName indicates missing common name.
	ErrMissingCommonName = errors.New("missing common name")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type.
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrValidation indicates that an error was returned by the API.
	ErrValidation = errors.New("something went wrong with the request")

	// ErrInvalidQueryParams indicates invalid query parameters.
	ErrInvalidQueryParams = errors.New("invalid query parameters")

	// ErrInvalidRequest indicates that the request is invalid.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMissingCSR indicates missing csr.
	ErrMissingCSR = errors.New("missing CSR")

	// ErrMissingPrivKey indicates missing csr.
	ErrMissingPrivKey = errors.New("missing private key")
)
