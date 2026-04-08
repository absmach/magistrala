// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"testing"

	"github.com/absmach/magistrala/auth"
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
			et:       auth.GroupsType,
			expected: "groups",
		},
		{
			desc:     "Channels entity type",
			et:       auth.ChannelsType,
			expected: "channels",
		},
		{
			desc:     "Clients entity type",
			et:       auth.ClientsType,
			expected: "clients",
		},
		{
			desc:     "Dashboard entity type",
			et:       auth.DashboardType,
			expected: "dashboards",
		},
		{
			desc:     "Messages entity type",
			et:       auth.MessagesType,
			expected: "messages",
		},
		{
			desc:     "Rules entity type",
			et:       auth.RulesType,
			expected: "rules",
		},
		{
			desc:     "Reports entity type",
			et:       auth.ReportsType,
			expected: "reports",
		},
		{
			desc:     "Unknown entity type",
			et:       auth.EntityType(100),
			expected: "unknown domain entity type 100",
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
			expected: auth.GroupsType,
			err:      false,
		},
		{
			desc:     "Parse channels",
			et:       "channels",
			expected: auth.ChannelsType,
			err:      false,
		},
		{
			desc:     "Parse clients",
			et:       "clients",
			expected: auth.ClientsType,
			err:      false,
		},
		{
			desc:     "Parse dashboards",
			et:       "dashboards",
			expected: auth.DashboardType,
			err:      false,
		},
		{
			desc:     "Parse rules",
			et:       "rules",
			expected: auth.RulesType,
			err:      false,
		},
		{
			desc:     "Parse reports",
			et:       "reports",
			expected: auth.ReportsType,
			err:      false,
		},
		{
			desc:     "Parse unknown entity type",
			et:       "unknown",
			expected: auth.EntityType(0),
			err:      true,
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
			et:       auth.GroupsType,
			expected: []byte(`"groups"`),
			err:      nil,
		},
		{
			desc:     "Marshal channels",
			et:       auth.ChannelsType,
			expected: []byte(`"channels"`),
			err:      nil,
		},
		{
			desc:     "Marshal clients",
			et:       auth.ClientsType,
			expected: []byte(`"clients"`),
			err:      nil,
		},
		{
			desc:     "Marshal rules",
			et:       auth.RulesType,
			expected: []byte(`"rules"`),
			err:      nil,
		},
		{
			desc:     "Marshal reports",
			et:       auth.ReportsType,
			expected: []byte(`"reports"`),
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
			expected: auth.GroupsType,
			err:      false,
		},
		{
			desc:     "Unmarshal channels",
			data:     []byte(`"channels"`),
			expected: auth.ChannelsType,
			err:      false,
		},
		{
			desc:     "Unmarshal rules",
			data:     []byte(`"rules"`),
			expected: auth.RulesType,
			err:      false,
		},
		{
			desc:     "Unmarshal reports",
			data:     []byte(`"reports"`),
			expected: auth.ReportsType,
			err:      false,
		},
		{
			desc:     "Unmarshal unknown",
			data:     []byte(`"unknown"`),
			expected: auth.EntityType(0),
			err:      true,
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
			et:       auth.GroupsType,
			expected: []byte("groups"),
			err:      nil,
		},
		{
			desc:     "Marshal channels as text",
			et:       auth.ChannelsType,
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
			expected: auth.GroupsType,
			err:      false,
		},
		{
			desc:     "Unmarshal channels from text",
			data:     []byte("channels"),
			expected: auth.ChannelsType,
			err:      false,
		},
		{
			desc:     "Unmarshal unknown from text",
			data:     []byte("unknown"),
			expected: auth.EntityType(0),
			err:      true,
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
