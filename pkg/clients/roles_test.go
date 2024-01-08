// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/stretchr/testify/assert"
)

func TestRoleString(t *testing.T) {
	cases := []struct {
		desc     string
		role     clients.Role
		expected string
	}{
		{"User", clients.UserRole, "user"},
		{"Admin", clients.AdminRole, "admin"},
		{"All", clients.AllRole, "all"},
		{"Unknown", clients.Role(100), "unknown"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := c.role.String()
			assert.Equal(t, c.expected, got, "String() = %v, expected %v", got, c.expected)
		})
	}
}

func TestToRole(t *testing.T) {
	cases := []struct {
		desc      string
		role      string
		expetcted clients.Role
		err       error
	}{
		{"User", "user", clients.UserRole, nil},
		{"Admin", "admin", clients.AdminRole, nil},
		{"All", "all", clients.AllRole, nil},
		{"Unknown", "unknown", clients.Role(0), apiutil.ErrInvalidRole},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := clients.ToRole(c.role)
			assert.Equal(t, c.err, err, "ToRole() error = %v, expected %v", err, c.err)
			assert.Equal(t, c.expetcted, got, "ToRole() = %v, expected %v", got, c.expetcted)
		})
	}
}

func TestRoleMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		role     clients.Role
		err      error
	}{
		{"User", []byte(`"user"`), clients.UserRole, nil},
		{"Admin", []byte(`"admin"`), clients.AdminRole, nil},
		{"All", []byte(`"all"`), clients.AllRole, nil},
		{"Unknown", []byte(`"unknown"`), clients.Role(100), nil},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.role.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestRoleUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected clients.Role
		role     []byte
		err      error
	}{
		{"User", clients.UserRole, []byte(`"user"`), nil},
		{"Admin", clients.AdminRole, []byte(`"admin"`), nil},
		{"All", clients.AllRole, []byte(`"all"`), nil},
		{"Unknown", clients.Role(0), []byte(`"unknown"`), apiutil.ErrInvalidRole},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var r clients.Role
			err := r.UnmarshalJSON(tc.role)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, r, "UnmarshalJSON() = %v, expected %v", r, tc.expected)
		})
	}
}
