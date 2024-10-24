// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"testing"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/things"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   things.Status
		expected string
	}{
		{
			desc:     "Enabled",
			status:   things.EnabledStatus,
			expected: "enabled",
		},
		{
			desc:     "Disabled",
			status:   things.DisabledStatus,
			expected: "disabled",
		},
		{
			desc:     "Deleted",
			status:   things.DeletedStatus,
			expected: "deleted",
		},
		{
			desc:     "All",
			status:   things.AllStatus,
			expected: "all",
		},
		{
			desc:     "Unknown",
			status:   things.Status(100),
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
		expetcted things.Status
		err       error
	}{
		{
			desc:      "Enabled",
			status:    "enabled",
			expetcted: things.EnabledStatus,
			err:       nil,
		},
		{
			desc:      "Disabled",
			status:    "disabled",
			expetcted: things.DisabledStatus,
			err:       nil,
		},
		{
			desc:      "Deleted",
			status:    "deleted",
			expetcted: things.DeletedStatus,
			err:       nil,
		},
		{
			desc:      "All",
			status:    "all",
			expetcted: things.AllStatus,
			err:       nil,
		},
		{
			desc:      "Unknown",
			status:    "unknown",
			expetcted: things.Status(0),
			err:       svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := things.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expetcted, got, "ToStatus() = %v, expected %v", got, tc.expetcted)
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		status   things.Status
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`"enabled"`),
			status:   things.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`"disabled"`),
			status:   things.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: []byte(`"deleted"`),
			status:   things.DeletedStatus,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			status:   things.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			status:   things.Status(100),
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
		expected things.Status
		status   []byte
		err      error
	}{
		{
			desc:     "Enabled",
			expected: things.EnabledStatus,
			status:   []byte(`"enabled"`),
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: things.DisabledStatus,
			status:   []byte(`"disabled"`),
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: things.DeletedStatus,
			status:   []byte(`"deleted"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: things.AllStatus,
			status:   []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: things.Status(0),
			status:   []byte(`"unknown"`),
			err:      svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var s things.Status
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
		user     things.Client
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"enabled"}`),
			user:     things.Client{Status: things.EnabledStatus},
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"disabled"}`),
			user:     things.Client{Status: things.DisabledStatus},
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"deleted"}`),
			user:     things.Client{Status: things.DeletedStatus},
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"all"}`),
			user:     things.Client{Status: things.AllStatus},
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"unknown"}`),
			user:     things.Client{Status: things.Status(100)},
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
