// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import smqauth "github.com/absmach/magistrala/auth"

type authenticateRes struct {
	id       string
	userID   string
	userRole smqauth.Role
	verified bool
}

type authorizeRes struct {
	id         string
	authorized bool
}
