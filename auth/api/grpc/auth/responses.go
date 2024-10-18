// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

type authenticateRes struct {
	id       string
	userID   string
	domainID string
}

type authorizeRes struct {
	id         string
	authorized bool
}
