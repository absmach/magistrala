// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"testing"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/things"
	"github.com/stretchr/testify/assert"
)

func TestRoleString(t *testing.T) {
	cases := []struct {
		desc     string
		role     things.Role
		expected string
	}{
		{
			desc:     "User",
			role:     things.UserRole,
			expected: "user",
		},
		{
			desc:     "Admin",
			role:     things.AdminRole,
			expected: "admin",
		},
		{
			desc:     "All",
			role:     things.AllRole,
			expected: "all",
		},
		{
			desc:     "Unknown",
			role:     things.Role(100),
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
		expected things.Role
		err      error
	}{
		{
			desc:     "User",
			role:     "user",
			expected: things.UserRole,
			err:      nil,
		},
		{
			desc:     "Admin",
			role:     "admin",
			expected: things.AdminRole,
			err:      nil,
		},
		{
			desc:     "All",
			role:     "all",
			expected: things.AllRole,
			err:      nil,
		},
		{
			desc:     "Unknown",
			role:     "unknown",
			expected: things.Role(0),
			err:      apiutil.ErrInvalidRole,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := things.ToRole(c.role)
			assert.Equal(t, c.err, err, "ToRole() error = %v, expected %v", err, c.err)
			assert.Equal(t, c.expected, got, "ToRole() = %v, expected %v", got, c.expected)
		})
	}
}

func TestRoleMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		role     things.Role
		err      error
	}{
		{
			desc:     "User",
			expected: []byte(`"user"`),
			role:     things.UserRole,
			err:      nil,
		},
		{
			desc:     "Admin",
			expected: []byte(`"admin"`),
			role:     things.AdminRole,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			role:     things.AllRole,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			role:     things.Role(100),
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
		expected things.Role
		role     []byte
		err      error
	}{
		{
			desc:     "User",
			expected: things.UserRole,
			role:     []byte(`"user"`),
			err:      nil,
		},
		{
			desc:     "Admin",
			expected: things.AdminRole,
			role:     []byte(`"admin"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: things.AllRole,
			role:     []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: things.Role(0),
			role:     []byte(`"unknown"`),
			err:      apiutil.ErrInvalidRole,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var r things.Role
			err := r.UnmarshalJSON(tc.role)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, r, "UnmarshalJSON() = %v, expected %v", r, tc.expected)
		})
	}
}
