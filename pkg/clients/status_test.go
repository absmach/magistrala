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
		{"Enabled", clients.EnabledStatus, "enabled"},
		{"Disabled", clients.DisabledStatus, "disabled"},
		{"All", clients.AllStatus, "all"},
		{"Unknown", clients.Status(100), "unknown"},
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
		{"Enabled", "enabled", clients.EnabledStatus, nil},
		{"Disabled", "disabled", clients.DisabledStatus, nil},
		{"All", "all", clients.AllStatus, nil},
		{"Unknown", "unknown", clients.Status(0), apiutil.ErrInvalidStatus},
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
		{"Enabled", []byte(`"enabled"`), clients.EnabledStatus, nil},
		{"Disabled", []byte(`"disabled"`), clients.DisabledStatus, nil},
		{"All", []byte(`"all"`), clients.AllStatus, nil},
		{"Unknown", []byte(`"unknown"`), clients.Status(100), nil},
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
		{"Enabled", clients.EnabledStatus, []byte(`"enabled"`), nil},
		{"Disabled", clients.DisabledStatus, []byte(`"disabled"`), nil},
		{"All", clients.AllStatus, []byte(`"all"`), nil},
		{"Unknown", clients.Status(0), []byte(`"unknown"`), apiutil.ErrInvalidStatus},
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
		{"Enabled", []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"enabled"}`), clients.Client{Status: clients.EnabledStatus}, nil},
		{"Disabled", []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"disabled"}`), clients.Client{Status: clients.DisabledStatus}, nil},
		{"All", []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"all"}`), clients.Client{Status: clients.AllStatus}, nil},
		{"Unknown", []byte(`{"id":"","credentials":{},"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"unknown"}`), clients.Client{Status: clients.Status(100)}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.user.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", got, tc.expected)
		})
	}
}
