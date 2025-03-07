// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"testing"

	"github.com/absmach/supermq/domains"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   domains.Status
		expected string
	}{
		{
			desc:     "Enabled",
			status:   domains.EnabledStatus,
			expected: "enabled",
		},
		{
			desc:     "Disabled",
			status:   domains.DisabledStatus,
			expected: "disabled",
		},
		{
			desc:     "Freezed",
			status:   domains.FreezeStatus,
			expected: "freezed",
		},
		{
			desc:     "All",
			status:   domains.AllStatus,
			expected: "all",
		},
		{
			desc:     "Unknown",
			status:   domains.Status(100),
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
		expetcted domains.Status
		err       error
	}{
		{
			desc:      "Enabled",
			status:    "enabled",
			expetcted: domains.EnabledStatus,
			err:       nil,
		},
		{
			desc:      "Disabled",
			status:    "disabled",
			expetcted: domains.DisabledStatus,
			err:       nil,
		},
		{
			desc:      "Freezed",
			status:    "freezed",
			expetcted: domains.FreezeStatus,
			err:       nil,
		},
		{
			desc:      "All",
			status:    "all",
			expetcted: domains.AllStatus,
			err:       nil,
		},
		{
			desc:      "Unknown",
			status:    "unknown",
			expetcted: domains.Status(0),
			err:       svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := domains.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expetcted, got, "ToStatus() = %v, expected %v", got, tc.expetcted)
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		status   domains.Status
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`"enabled"`),
			status:   domains.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`"disabled"`),
			status:   domains.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			status:   domains.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			status:   domains.Status(100),
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
		expected domains.Status
		status   []byte
		err      error
	}{
		{
			desc:     "Enabled",
			expected: domains.EnabledStatus,
			status:   []byte(`"enabled"`),
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: domains.DisabledStatus,
			status:   []byte(`"disabled"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: domains.AllStatus,
			status:   []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: domains.Status(0),
			status:   []byte(`"unknown"`),
			err:      svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var s domains.Status
			err := s.UnmarshalJSON(tc.status)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, s, "UnmarshalJSON() = %v, expected %v", s, tc.expected)
		})
	}
}
