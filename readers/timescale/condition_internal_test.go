// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"strings"
	"testing"

	"github.com/absmach/magistrala/readers"
	"github.com/stretchr/testify/assert"
)

const deviceIDsClause = `device_id = ANY(:device_ids)`

// TestFmtConditionDeviceIDs pins the WHERE-builder contract for DeviceIDs,
// including the empty-slice case that MG-08 depends on: `omitempty` erases an
// empty slice, so it emits no condition and reads as "no device filter". A
// caller authorized for zero devices must short-circuit, never assign []string{}.
func TestFmtConditionDeviceIDs(t *testing.T) {
	cases := []struct {
		desc     string
		pageMeta readers.PageMetadata
		contains bool
	}{
		{
			desc:     "unset device ids emit no condition",
			pageMeta: readers.PageMetadata{},
			contains: false,
		},
		{
			desc:     "nil device ids emit no condition",
			pageMeta: readers.PageMetadata{DeviceIDs: nil},
			contains: false,
		},
		{
			desc:     "empty device ids emit no condition and so do not match nothing",
			pageMeta: readers.PageMetadata{DeviceIDs: []string{}},
			contains: false,
		},
		{
			desc:     "single device id emits the condition",
			pageMeta: readers.PageMetadata{DeviceIDs: []string{"meter-1"}},
			contains: true,
		},
		{
			desc:     "multiple device ids emit one condition",
			pageMeta: readers.PageMetadata{DeviceIDs: []string{"meter-1", "meter-2", "meter-3"}},
			contains: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cond := fmtCondition(tc.pageMeta)
			assert.Equal(t, tc.contains, strings.Contains(cond, deviceIDsClause), "condition was %q", cond)
			assert.Equal(t, 1, strings.Count(cond, "channel = :channel"), "condition was %q", cond)
		})
	}
}

func TestFmtConditionDeviceIDsComposes(t *testing.T) {
	cond := fmtCondition(readers.PageMetadata{
		DeviceIDs:  []string{"meter-1"},
		Publishers: []string{"pub-1", "pub-2"},
		Subtopic:   "sub",
		Name:       "temperature",
		Protocol:   "mqtt",
		From:       1,
		To:         2,
	})

	for _, want := range []string{
		deviceIDsClause,
		`publisher = ANY(:publishers)`,
		`subtopic = :subtopic`,
		`name = :name`,
		`protocol = :protocol`,
		`time >= :from`,
		`time < :to`,
	} {
		assert.True(t, strings.Contains(cond, want), "expected %q in %q", want, cond)
	}
}

// The singular publisher is suppressed by the plural one, but device_ids is an
// independent axis and must survive alongside either.
func TestFmtConditionDeviceIDsWithSingularPublisher(t *testing.T) {
	cond := fmtCondition(readers.PageMetadata{
		DeviceIDs: []string{"meter-1"},
		Publisher: "pub-1",
	})

	assert.True(t, strings.Contains(cond, deviceIDsClause), "condition was %q", cond)
	assert.True(t, strings.Contains(cond, `publisher = :publisher`), "condition was %q", cond)
	assert.False(t, strings.Contains(cond, `publisher = ANY(:publishers)`), "condition was %q", cond)
}
