// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"testing"

	"github.com/absmach/magistrala/auth"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   auth.Status
		expected string
	}{
		{
			desc:     "Active",
			status:   auth.ActiveStatus,
			expected: "active",
		},
		{
			desc:     "Revoked",
			status:   auth.RevokedStatus,
			expected: "revoked",
		},
		{
			desc:     "Expired",
			status:   auth.ExpiredStatus,
			expected: "expired",
		},
		{
			desc:     "All",
			status:   auth.AllStatus,
			expected: "all",
		},
		{
			desc:     "Unknown",
			status:   auth.Status(100),
			expected: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.status.String()
			assert.Equal(t, tc.expected, got, "String() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestToStatus(t *testing.T) {
	cases := []struct {
		desc     string
		status   string
		expected auth.Status
		err      error
	}{
		{
			desc:     "Active",
			status:   "active",
			expected: auth.ActiveStatus,
			err:      nil,
		},
		{
			desc:     "Empty string defaults to Active",
			status:   "",
			expected: auth.ActiveStatus,
			err:      nil,
		},
		{
			desc:     "Revoked",
			status:   "revoked",
			expected: auth.RevokedStatus,
			err:      nil,
		},
		{
			desc:     "Expired",
			status:   "expired",
			expected: auth.ExpiredStatus,
			err:      nil,
		},
		{
			desc:     "All",
			status:   "all",
			expected: auth.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			status:   "unknown",
			expected: auth.Status(0),
			err:      svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := auth.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "ToStatus() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		status   auth.Status
		err      error
	}{
		{
			desc:     "Active",
			expected: []byte(`"active"`),
			status:   auth.ActiveStatus,
			err:      nil,
		},
		{
			desc:     "Revoked",
			expected: []byte(`"revoked"`),
			status:   auth.RevokedStatus,
			err:      nil,
		},
		{
			desc:     "Expired",
			expected: []byte(`"expired"`),
			status:   auth.ExpiredStatus,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			status:   auth.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			status:   auth.Status(100),
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.status.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestStatusUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected auth.Status
		status   []byte
		err      error
	}{
		{
			desc:     "Active",
			expected: auth.ActiveStatus,
			status:   []byte(`"active"`),
			err:      nil,
		},
		{
			desc:     "Revoked",
			expected: auth.RevokedStatus,
			status:   []byte(`"revoked"`),
			err:      nil,
		},
		{
			desc:     "Expired",
			expected: auth.ExpiredStatus,
			status:   []byte(`"expired"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: auth.AllStatus,
			status:   []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: auth.Status(0),
			status:   []byte(`"unknown"`),
			err:      svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var s auth.Status
			err := s.UnmarshalJSON(tc.status)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, s, "UnmarshalJSON() = %v, expected %v", s, tc.expected)
		})
	}
}

func TestPATMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		pat      auth.PAT
		expected string
		err      error
	}{
		{
			desc: "Active PAT",
			pat: auth.PAT{
				ID:     "test-id",
				Name:   "test-pat",
				Status: auth.ActiveStatus,
			},
			expected: `"status":"active"`,
			err:      nil,
		},
		{
			desc: "Revoked PAT",
			pat: auth.PAT{
				ID:     "test-id",
				Name:   "test-pat",
				Status: auth.RevokedStatus,
			},
			expected: `"status":"revoked"`,
			err:      nil,
		},
		{
			desc: "Expired PAT",
			pat: auth.PAT{
				ID:     "test-id",
				Name:   "test-pat",
				Status: auth.ExpiredStatus,
			},
			expected: `"status":"expired"`,
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.pat.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Contains(t, string(got), tc.expected, "MarshalJSON() should contain %v", tc.expected)
		})
	}
}
