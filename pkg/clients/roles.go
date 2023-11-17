// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"encoding/json"
	"strings"

	"github.com/absmach/magistrala/internal/apiutil"
)

// Role represents Client role.
type Role uint8

// Possible Client role values.
const (
	UserRole Role = iota
	AdminRole
)

// String representation of the possible role values.
const (
	Admin = "admin"
	User  = "user"
)

// String converts client role to string literal.
func (cs Role) String() string {
	switch cs {
	case AdminRole:
		return Admin
	case UserRole:
		return User
	default:
		return Unknown
	}
}

// ToRole converts string value to a valid Client role.
func ToRole(status string) (Role, error) {
	switch status {
	case "", User:
		return UserRole, nil
	case Admin:
		return AdminRole, nil
	}
	return Role(0), apiutil.ErrInvalidRole
}

func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Role) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToRole(str)
	*r = val
	return err
}
