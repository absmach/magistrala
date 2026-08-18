// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

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
			cond := fmtCondition("chan-1", tc.pageMeta)
			assert.Equal(t, tc.contains, strings.Contains(cond, deviceIDsClause), "condition was %q", cond)
			assert.Equal(t, 1, strings.Count(cond, "channel = :channel"), "condition was %q", cond)
		})
	}
}

func TestFmtConditionDeviceIDsComposes(t *testing.T) {
	cond := fmtCondition("chan-1", readers.PageMetadata{
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
	cond := fmtCondition("chan-1", readers.PageMetadata{
		DeviceIDs: []string{"meter-1"},
		Publisher: "pub-1",
	})

	assert.True(t, strings.Contains(cond, deviceIDsClause), "condition was %q", cond)
	assert.True(t, strings.Contains(cond, `publisher = :publisher`), "condition was %q", cond)
	assert.False(t, strings.Contains(cond, `publisher = ANY(:publishers)`), "condition was %q", cond)
}

const scopeClause = `publisher = ANY(:scope_publishers) OR device_id = ANY(:scope_device_ids)`

// DeviceScope is the authorization boundary, so unlike the convenience filters
// an empty one must survive into the query and exclude every row. Being a
// pointer is what makes that work: `omitempty` drops nil, not a non-nil empty.
func TestFmtConditionDeviceScope(t *testing.T) {
	cases := []struct {
		desc     string
		pageMeta readers.PageMetadata
		contains bool
	}{
		{
			desc:     "no scope leaves the query unbounded",
			pageMeta: readers.PageMetadata{},
			contains: false,
		},
		{
			desc:     "a nil scope leaves the query unbounded",
			pageMeta: readers.PageMetadata{DeviceScope: nil},
			contains: false,
		},
		{
			desc:     "an empty scope still bounds the query, so it matches nothing",
			pageMeta: readers.PageMetadata{DeviceScope: &readers.DeviceScope{}},
			contains: true,
		},
		{
			desc: "a populated scope bounds the query",
			pageMeta: readers.PageMetadata{DeviceScope: &readers.DeviceScope{
				PublisherIDs: []string{"uuid-1"},
				DeviceIDs:    []string{"Meter.1-01:X"},
			}},
			contains: true,
		},
		{
			desc: "a scope with only one projection populated still bounds the query",
			pageMeta: readers.PageMetadata{DeviceScope: &readers.DeviceScope{
				PublisherIDs: []string{"uuid-1"},
			}},
			contains: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cond := fmtCondition("chan-1", tc.pageMeta)
			assert.Equal(t, tc.contains, strings.Contains(cond, scopeClause), "condition was %q", cond)
		})
	}
}

// The two projections are OR'd: a device is named by publisher when it publishes
// for itself and by device_id when a gateway relays for it, and one row carries
// only one of the two. AND would make gateway-relayed data unreachable.
func TestFmtConditionDeviceScopeComposesConjunctively(t *testing.T) {
	cond := fmtCondition("chan-1", readers.PageMetadata{
		DeviceScope: &readers.DeviceScope{PublisherIDs: []string{"uuid-1"}, DeviceIDs: []string{"serial-1"}},
		DeviceIDs:   []string{"serial-1"},
		Subtopic:    "sub",
		From:        1,
	})

	assert.True(t, strings.Contains(cond, scopeClause), "condition was %q", cond)
	assert.True(t, strings.Contains(cond, deviceIDsClause), "condition was %q", cond)
	assert.True(t, strings.Contains(cond, "subtopic = :subtopic"), "condition was %q", cond)
	assert.True(t, strings.Contains(cond, "time >= :from"), "condition was %q", cond)
}
