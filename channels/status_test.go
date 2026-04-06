// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package channels_test

import (
	"testing"

	"github.com/absmach/magistrala/channels"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc     string
		status   channels.Status
		expected string
	}{
		{
			desc:     "Enabled",
			status:   channels.EnabledStatus,
			expected: "enabled",
		},
		{
			desc:     "Disabled",
			status:   channels.DisabledStatus,
			expected: "disabled",
		},
		{
			desc:     "Deleted",
			status:   channels.DeletedStatus,
			expected: "deleted",
		},
		{
			desc:     "All",
			status:   channels.AllStatus,
			expected: "all",
		},
		{
			desc:     "Unknown",
			status:   channels.Status(100),
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
		expetcted channels.Status
		err       error
	}{
		{
			desc:      "Enabled",
			status:    "enabled",
			expetcted: channels.EnabledStatus,
			err:       nil,
		},
		{
			desc:      "Disabled",
			status:    "disabled",
			expetcted: channels.DisabledStatus,
			err:       nil,
		},
		{
			desc:      "Deleted",
			status:    "deleted",
			expetcted: channels.DeletedStatus,
			err:       nil,
		},
		{
			desc:      "All",
			status:    "all",
			expetcted: channels.AllStatus,
			err:       nil,
		},
		{
			desc:      "Unknown",
			status:    "unknown",
			expetcted: channels.Status(0),
			err:       svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := channels.ToStatus(tc.status)
			assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expetcted, got, "ToStatus() = %v, expected %v", got, tc.expetcted)
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		status   channels.Status
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`"enabled"`),
			status:   channels.EnabledStatus,
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`"disabled"`),
			status:   channels.DisabledStatus,
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: []byte(`"deleted"`),
			status:   channels.DeletedStatus,
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`"all"`),
			status:   channels.AllStatus,
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`"unknown"`),
			status:   channels.Status(100),
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
		expected channels.Status
		status   []byte
		err      error
	}{
		{
			desc:     "Enabled",
			expected: channels.EnabledStatus,
			status:   []byte(`"enabled"`),
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: channels.DisabledStatus,
			status:   []byte(`"disabled"`),
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: channels.DeletedStatus,
			status:   []byte(`"deleted"`),
			err:      nil,
		},
		{
			desc:     "All",
			expected: channels.AllStatus,
			status:   []byte(`"all"`),
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: channels.Status(0),
			status:   []byte(`"unknown"`),
			err:      svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var s channels.Status
			err := s.UnmarshalJSON(tc.status)
			assert.Equal(t, tc.err, err, "UnmarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, s, "UnmarshalJSON() = %v, expected %v", s, tc.expected)
		})
	}
}

func TestChannelMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		expected []byte
		user     channels.Channel
		err      error
	}{
		{
			desc:     "Enabled",
			expected: []byte(`{"id":"","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"enabled"}`),
			user:     channels.Channel{Status: channels.EnabledStatus},
			err:      nil,
		},
		{
			desc:     "Disabled",
			expected: []byte(`{"id":"","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"disabled"}`),
			user:     channels.Channel{Status: channels.DisabledStatus},
			err:      nil,
		},
		{
			desc:     "Deleted",
			expected: []byte(`{"id":"","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"deleted"}`),
			user:     channels.Channel{Status: channels.DeletedStatus},
			err:      nil,
		},
		{
			desc:     "All",
			expected: []byte(`{"id":"","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"all"}`),
			user:     channels.Channel{Status: channels.AllStatus},
			err:      nil,
		},
		{
			desc:     "Unknown",
			expected: []byte(`{"id":"","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"unknown"}`),
			user:     channels.Channel{Status: channels.Status(100)},
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.user.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", string(got), string(tc.expected))
		})
	}
}
