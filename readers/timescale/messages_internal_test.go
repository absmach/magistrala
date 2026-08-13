// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"testing"

	"github.com/absmach/magistrala/readers"
	"github.com/stretchr/testify/assert"
)

func TestAggregateDeviceIDProjection(t *testing.T) {
	cases := []struct {
		desc string
		pm   readers.PageMetadata
		want string
	}{
		{
			desc: "no device filter omits aggregate attribution",
			pm:   readers.PageMetadata{},
			want: "'' AS device_id",
		},
		{
			desc: "single device filter preserves aggregate attribution",
			pm:   readers.PageMetadata{DeviceIDs: []string{"meter-a"}},
			want: "FIRST(device_id, time) AS device_id",
		},
		{
			desc: "multiple device filters omit mixed aggregate attribution",
			pm:   readers.PageMetadata{DeviceIDs: []string{"meter-a", "meter-b"}},
			want: "'' AS device_id",
		},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, aggregateDeviceIDProjection(tc.pm), tc.desc)
	}
}
