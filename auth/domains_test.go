// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   auth.Status
		expected string
	}{
		{
			desc:     "Enabled",
			status:   auth.EnabledStatus,
			expected: "enabled",
		},
		{
			desc:     "Disabled",
			status:   auth.DisabledStatus,
			expected: "disabled",
		},
		{
			desc:     "Freezed",
			status:   auth.FreezeStatus,
			expected: "freezed",
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
		desc      string
		status    string
		expetcted auth.Status
		err       error
	}{
		{
			desc:      "Enabled",
			status:    "enabled",
			expetcted: auth.EnabledStatus,
			err:       nil,
		},
		{
			desc:      "Disabled",
			status:    "disabled",
			expetcted: auth.DisabledStatus,
			err:       nil,
		},
		{
			desc:      "Freezed",
			status:    "freezed",
			expetcted: auth.FreezeStatus,
			err:       nil,
		},
		{
			desc:      "All",
			status:    "all",
			expetcted: auth.AllStatus,
			err:       nil,
		},
		{
			desc:      "Unknown",
			status:    "unknown",
			expetcted: auth.Status(0),
			err:       apiutil.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := auth.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expetcted, got, "ToStatus() = %v, expected %v", got, tc.expetcted)
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
			desc:     "Enabled",
			expected: []byte(`"enabled"`),
			status:   auth.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`"disabled"`),
			status:   auth.DisabledStatus,
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
			desc:     "Enabled",
			expected: auth.EnabledStatus,
			status:   []byte(`"enabled"`),
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: auth.DisabledStatus,
			status:   []byte(`"disabled"`),
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
			err:      apiutil.ErrInvalidStatus,
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
