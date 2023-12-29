// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups_test

import (
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	cases := []struct {
		name     string
		status   groups.Status
		expected string
	}{
		{"Enabled", groups.EnabledStatus, "enabled"},
		{"Disabled", groups.DisabledStatus, "disabled"},
		{"All", groups.AllStatus, "all"},
		{"Unknown", groups.Status(100), "unknown"},
	}

	for _, tc := range cases {
		got := tc.status.String()
		assert.Equal(t, tc.expected, got, "Status.String() = %v, expected %v", got, tc.expected)
	}
}

func TestToStatus(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		gstatus groups.Status
		err     error
	}{
		{"Enabled", "enabled", groups.EnabledStatus, nil},
		{"Disabled", "disabled", groups.DisabledStatus, nil},
		{"All", "all", groups.AllStatus, nil},
		{"Unknown", "unknown", groups.Status(0), apiutil.ErrInvalidStatus},
	}

	for _, tc := range cases {
		got, err := groups.ToStatus(tc.status)
		assert.Equal(t, tc.err, err, "ToStatus() error = %v, expected %v", err, tc.err)
		assert.Equal(t, tc.gstatus, got, "ToStatus() = %v, expected %v", got, tc.gstatus)
	}
}
