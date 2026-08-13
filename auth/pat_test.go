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
			desc:     "Devices entity type",
			et:       auth.DevicesType,
			expected: "devices",
		},
		{
			desc:     "Device types entity type",
			et:       auth.DeviceTypesType,
			expected: "device_types",
		},
		{
			desc:     "Bootstrap entity type",
			et:       auth.BootstrapType,
			expected: "bootstrap",
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
			desc:     "Parse devices",
			et:       "devices",
			expected: auth.DevicesType,
			err:      false,
		},
		{
			desc:     "Parse device types",
			et:       "device_types",
			expected: auth.DeviceTypesType,
			err:      false,
		},
		{
			desc:     "Parse bootstrap",
			et:       "bootstrap",
			expected: auth.BootstrapType,
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
			desc:     "Marshal devices",
			et:       auth.DevicesType,
			expected: []byte(`"devices"`),
			err:      nil,
		},
		{
			desc:     "Marshal device types",
			et:       auth.DeviceTypesType,
			expected: []byte(`"device_types"`),
			err:      nil,
		},
		{
			desc:     "Marshal bootstrap",
			et:       auth.BootstrapType,
			expected: []byte(`"bootstrap"`),
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
			desc:     "Unmarshal bootstrap",
			data:     []byte(`"bootstrap"`),
			expected: auth.BootstrapType,
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
		{
			desc:     "Marshal bootstrap as text",
			et:       auth.BootstrapType,
			expected: []byte("bootstrap"),
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
			desc:     "Unmarshal bootstrap from text",
			data:     []byte("bootstrap"),
			expected: auth.BootstrapType,
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

// TestEntityTypeRoundTrip pins every declared EntityType against all four of
// its representations at once. String(), ParseEntityType, JSON and text are
// written separately, so a new variant added to one and forgotten in another
// only shows up here.
func TestEntityTypeRoundTrip(t *testing.T) {
	cases := []struct {
		et   auth.EntityType
		name string
	}{
		{auth.GroupsType, "groups"},
		{auth.ChannelsType, "channels"},
		{auth.BootstrapType, "bootstrap"},
		{auth.DashboardType, "dashboards"},
		{auth.MessagesType, "messages"},
		{auth.DomainsType, "domains"},
		{auth.UsersType, "users"},
		{auth.RulesType, "rules"},
		{auth.ReportsType, "reports"},
		{auth.DevicesType, "devices"},
		{auth.DeviceTypesType, "device_types"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.name, tc.et.String())

			parsed, err := auth.ParseEntityType(tc.name)
			assert.NoError(t, err)
			assert.Equal(t, tc.et, parsed)

			marshalled, err := tc.et.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, []byte(`"`+tc.name+`"`), marshalled)

			var fromJSON auth.EntityType
			assert.NoError(t, fromJSON.UnmarshalJSON(marshalled))
			assert.Equal(t, tc.et, fromJSON)

			text, err := tc.et.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, []byte(tc.name), text)

			var fromText auth.EntityType
			assert.NoError(t, fromText.UnmarshalText(text))
			assert.Equal(t, tc.et, fromText)
		})
	}
}

// TestDeviceTypesOperationsAreValid mirrors devices: device types accept any
// operation, unlike dashboards and messages which accept a fixed pair.
func TestDeviceTypesOperationsAreValid(t *testing.T) {
	for _, op := range []string{"view", "update", "create_version", "list_versions"} {
		assert.True(t, auth.IsValidOperationForEntity(auth.DeviceTypesType, op), "operation %s must be valid for device_types", op)
	}
}
