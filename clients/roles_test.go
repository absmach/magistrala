// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"testing"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

func TestRoleString(t *testing.T) {
	cases := []struct {
		desc     string
		role     clients.Role
		expected string
	}{
		{
			desc:     "User",
			role:     clients.UserRole,
			expected: "user",
		},
		{
			desc:     "Admin",
			role:     clients.AdminRole,
			expected: "admin",
		},
		{
			desc:     "All",
			role:     clients.AllRole,
			expected: "all",
		},
		{
			desc:     "Unknown",
			role:     clients.Role(100),
			expected: "unknown",
		},
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
		desc     string
		role     string
		expected clients.Role
		err      error
	}{
		{
			desc:     "User",
			role:     "user",
			expected: clients.UserRole,
			err:      nil,
		},
		{
			desc:     "Admin",
			role:     "admin",
			expected: clients.AdminRole,
			err:      nil,
		},
		{
			desc:     "All",
			role:     "all",
			expected: clients.AllRole,
			err:      nil,
		},
		{
			desc:     "Unknown",
			role:     "unknown",
			expected: clients.Role(0),
			err:      apiutil.ErrInvalidRole,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := clients.ToRole(c.role)
			assert.Equal(t, c.err, err, "ToRole() error = %v, expected %v", err, c.err)
			assert.Equal(t, c.expected, got, "ToRole() = %v, expected %v", got, c.expected)
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
		{
			desc:     "User",
			expected: []byte(`"user"`),
			role:     clients.UserRole,
			err:      nil,
		},
		{
			desc:     "Admin",
			expected: []byte(`"admin"`),
			role:     clients.AdminRole,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			role:     clients.AllRole,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			role:     clients.Role(100),
			err:      nil,
		},
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
		{
			desc:     "User",
			expected: clients.UserRole,
			role:     []byte(`"user"`),
			err:      nil,
		},
		{
			desc:     "Admin",
			expected: clients.AdminRole,
			role:     []byte(`"admin"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: clients.AllRole,
			role:     []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: clients.Role(0),
			role:     []byte(`"unknown"`),
			err:      apiutil.ErrInvalidRole,
		},
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
