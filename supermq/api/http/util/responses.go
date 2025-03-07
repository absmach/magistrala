// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package util

// ErrorRes represents the HTTP error response body.
type ErrorRes struct {
	Err string `json:"error"`
	Msg string `json:"message"`
}
