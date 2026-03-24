// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"testing"

	"github.com/absmach/supermq/auth"
	"github.com/stretchr/testify/assert"
)

func TestEntityTypeString(t *testing.T) {
	cases := []struct {
		desc     string
		et       auth.EntityType
		expected string
	}{
		{
			desc:     "Groups entity type",
			et:       auth.EntityType("groups"),
			expected: "groups",
		},
		{
			desc:     "Channels entity type",
			et:       auth.EntityType("channels"),
			expected: "channels",
		},
		{
			desc:     "Clients entity type",
			et:       auth.EntityType("clients"),
			expected: "clients",
		},
		{
			desc:     "Dashboard entity type",
			et:       auth.EntityType("dashboards"),
			expected: "dashboards",
		},
		{
			desc:     "Messages entity type",
			et:       auth.EntityType("messages"),
			expected: "messages",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.et.String()
			assert.Equal(t, tc.expected, got, "String() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestParseEntityType(t *testing.T) {
	cases := []struct {
		desc     string
		et       string
		expected auth.EntityType
		err      bool
	}{
		{
			desc:     "Parse groups",
			et:       "groups",
			expected: auth.EntityType("groups"),
			err:      false,
		},
		{
			desc:     "Parse channels",
			et:       "channels",
			expected: auth.EntityType("channels"),
			err:      false,
		},
		{
			desc:     "Parse clients",
			et:       "clients",
			expected: auth.EntityType("clients"),
			err:      false,
		},
		{
			desc:     "Parse dashboards",
			et:       "dashboards",
			expected: auth.EntityType("dashboards"),
			err:      false,
		},
		{
			desc:     "Parse unknown entity type",
			et:       "unknown",
			expected: auth.EntityType("unknown"),
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := auth.ParseEntityType(tc.et)
			if tc.err {
				assert.Error(t, err, "ParseEntityType() should return error")
			} else {
				assert.NoError(t, err, "ParseEntityType() should not return error")
				assert.Equal(t, tc.expected, got, "ParseEntityType() = %v, expected %v", got, tc.expected)
			}
		})
	}
}

func TestEntityTypeMarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		et       auth.EntityType
		expected []byte
		err      error
	}{
		{
			desc:     "Marshal groups",
			et:       auth.EntityType("groups"),
			expected: []byte(`"groups"`),
			err:      nil,
		},
		{
			desc:     "Marshal channels",
			et:       auth.EntityType("channels"),
			expected: []byte(`"channels"`),
			err:      nil,
		},
		{
			desc:     "Marshal clients",
			et:       auth.EntityType("clients"),
			expected: []byte(`"clients"`),
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.et.MarshalJSON()
			assert.Equal(t, tc.err, err, "MarshalJSON() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalJSON() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestEntityTypeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		data     []byte
		expected auth.EntityType
		err      bool
	}{
		{
			desc:     "Unmarshal groups",
			data:     []byte(`"groups"`),
			expected: auth.EntityType("groups"),
			err:      false,
		},
		{
			desc:     "Unmarshal channels",
			data:     []byte(`"channels"`),
			expected: auth.EntityType("channels"),
			err:      false,
		},
		{
			desc:     "Unmarshal unknown",
			data:     []byte(`"unknown"`),
			expected: auth.EntityType("unknown"),
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var et auth.EntityType
			err := et.UnmarshalJSON(tc.data)
			if tc.err {
				assert.Error(t, err, "UnmarshalJSON() should return error")
			} else {
				assert.NoError(t, err, "UnmarshalJSON() should not return error")
				assert.Equal(t, tc.expected, et, "UnmarshalJSON() = %v, expected %v", et, tc.expected)
			}
		})
	}
}

func TestEntityTypeMarshalText(t *testing.T) {
	cases := []struct {
		desc     string
		et       auth.EntityType
		expected []byte
		err      error
	}{
		{
			desc:     "Marshal groups as text",
			et:       auth.EntityType("groups"),
			expected: []byte("groups"),
			err:      nil,
		},
		{
			desc:     "Marshal channels as text",
			et:       auth.EntityType("channels"),
			expected: []byte("channels"),
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.et.MarshalText()
			assert.Equal(t, tc.err, err, "MarshalText() error = %v, expected %v", err, tc.err)
			assert.Equal(t, tc.expected, got, "MarshalText() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestEntityTypeUnmarshalText(t *testing.T) {
	cases := []struct {
		desc     string
		data     []byte
		expected auth.EntityType
		err      bool
	}{
		{
			desc:     "Unmarshal groups from text",
			data:     []byte("groups"),
			expected: auth.EntityType("groups"),
			err:      false,
		},
		{
			desc:     "Unmarshal channels from text",
			data:     []byte("channels"),
			expected: auth.EntityType("channels"),
			err:      false,
		},
		{
			desc:     "Unmarshal unknown from text",
			data:     []byte("unknown"),
			expected: auth.EntityType("unknown"),
			err:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var et auth.EntityType
			err := et.UnmarshalText(tc.data)
			if tc.err {
				assert.Error(t, err, "UnmarshalText() should return error")
			} else {
				assert.NoError(t, err, "UnmarshalText() should not return error")
				assert.Equal(t, tc.expected, et, "UnmarshalText() = %v, expected %v", et, tc.expected)
			}
		})
	}
}
