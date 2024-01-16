// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   clients.Status
		expected string
	}{
		{
			desc:     "Enabled",
			status:   clients.EnabledStatus,
			expected: "enabled",
		},
		{
			desc:     "Disabled",
			status:   clients.DisabledStatus,
			expected: "disabled",
		},
		{
			desc:     "All",
			status:   clients.AllStatus,
			expected: "all",
		},
		{
			desc:     "Unknown",
			status:   clients.Status(100),
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
		expetcted clients.Status
		err       error
	}{
		{
			desc:      "Enabled",
			status:    "enabled",
			expetcted: clients.EnabledStatus,
			err:       nil,
		},
		{
			desc:      "Disabled",
			status:    "disabled",
			expetcted: clients.DisabledStatus,
			err:       nil,
		},
		{
			desc:      "All",
			status:    "all",
			expetcted: clients.AllStatus,
			err:       nil,
		},
		{
			desc:      "Unknown",
			status:    "unknown",
			expetcted: clients.Status(0),
			err:       apiutil.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := clients.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expetcted, got, "ToStatus() = %v, expected %v", got, tc.expetcted)
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		status   clients.Status
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`"enabled"`),
			status:   clients.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`"disabled"`),
			status:   clients.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			status:   clients.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			status:   clients.Status(100),
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
		expected clients.Status
		status   []byte
		err      error
	}{
		{
			desc:     "Enabled",
			expected: clients.EnabledStatus,
			status:   []byte(`"enabled"`),
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: clients.DisabledStatus,
			status:   []byte(`"disabled"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: clients.AllStatus,
			status:   []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: clients.Status(0),
			status:   []byte(`"unknown"`),
			err:      apiutil.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var s clients.Status
			err := s.UnmarshalJSON(tc.status)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, s, "UnmarshalJSON() = %v, expected %v", s, tc.expected)
		})
	}
}

func TestUserMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		user     clients.Client
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"enabled"}`),
			user:     clients.Client{Status: clients.EnabledStatus},
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"disabled"}`),
			user:     clients.Client{Status: clients.DisabledStatus},
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"all"}`),
			user:     clients.Client{Status: clients.AllStatus},
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"unknown"}`),
			user:     clients.Client{Status: clients.Status(100)},
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.user.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", got, tc.expected)
		})
	}
}
