// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import "github.com/absmach/magistrala/auth"

type authenticateRes struct {
	id        string
	userID    string
	userRole  auth.Role
	verified  bool
	tokenType auth.KeyType
}

type authorizeRes struct {
	id         string
	authorized bool
}
