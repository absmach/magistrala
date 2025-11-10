// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re_test

import (
	"encoding/json"
	"testing"

	"github.com/absmach/magistrala/re"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToStatus(t *testing.T) {
	cases := []struct {
		desc   string
		status string
		res    re.Status
		err    error
	}{
		{
			desc:   "convert enabled status",
			status: re.Enabled,
			res:    re.EnabledStatus,
			err:    nil,
		},
		{
			desc:   "convert empty string to enabled status",
			status: "",
			res:    re.EnabledStatus,
			err:    nil,
		},
		{
			desc:   "convert disabled status",
			status: re.Disabled,
			res:    re.DisabledStatus,
			err:    nil,
		},
		{
			desc:   "convert deleted status",
			status: re.Deleted,
			res:    re.DeletedStatus,
			err:    nil,
		},
		{
			desc:   "convert all status",
			status: re.All,
			res:    re.AllStatus,
			err:    nil,
		},
		{
			desc:   "convert invalid status",
			status: "invalid",
			res:    re.Status(0),
			err:    svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			status, err := re.ToStatus(tc.status)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.res, status)
		})
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		desc   string
		status re.Status
		res    string
	}{
		{
			desc:   "enabled status to string",
			status: re.EnabledStatus,
			res:    re.Enabled,
		},
		{
			desc:   "disabled status to string",
			status: re.DisabledStatus,
			res:    re.Disabled,
		},
		{
			desc:   "deleted status to string",
			status: re.DeletedStatus,
			res:    re.Deleted,
		},
		{
			desc:   "all status to string",
			status: re.AllStatus,
			res:    re.All,
		},
		{
			desc:   "unknown status to string",
			status: re.Status(99),
			res:    re.Unknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.res, tc.status.String())
		})
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc   string
		status re.Status
		res    string
	}{
		{
			desc:   "marshal enabled status",
			status: re.EnabledStatus,
			res:    `"enabled"`,
		},
		{
			desc:   "marshal disabled status",
			status: re.DisabledStatus,
			res:    `"disabled"`,
		},
		{
			desc:   "marshal deleted status",
			status: re.DeletedStatus,
			res:    `"deleted"`,
		},
		{
			desc:   "marshal all status",
			status: re.AllStatus,
			res:    `"all"`,
		},
		{
			desc:   "marshal unknown status",
			status: re.Status(99),
			res:    `"unknown"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data, err := json.Marshal(tc.status)
			require.NoError(t, err)
			assert.Equal(t, tc.res, string(data))
		})
	}
}

func TestStatusUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc string
		data string
		res  re.Status
		err  error
	}{
		{
			desc: "unmarshal enabled status",
			data: `"enabled"`,
			res:  re.EnabledStatus,
			err:  nil,
		},
		{
			desc: "unmarshal disabled status",
			data: `"disabled"`,
			res:  re.DisabledStatus,
			err:  nil,
		},
		{
			desc: "unmarshal deleted status",
			data: `"deleted"`,
			res:  re.DeletedStatus,
			err:  nil,
		},
		{
			desc: "unmarshal all status",
			data: `"all"`,
			res:  re.AllStatus,
			err:  nil,
		},
		{
			desc: "unmarshal empty string to enabled status",
			data: `""`,
			res:  re.EnabledStatus,
			err:  nil,
		},
		{
			desc: "unmarshal invalid status",
			data: `"invalid"`,
			res:  re.Status(0),
			err:  svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var status re.Status
			err := json.Unmarshal([]byte(tc.data), &status)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.res, status)
		})
	}
}
