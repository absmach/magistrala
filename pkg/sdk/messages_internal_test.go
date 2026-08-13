// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The list-valued message filters go out as repeated query parameters, one per
// value, so entries reach the reader byte-identically — device serials carry no
// format constraint (MG-09) and would not survive being joined on a separator.
func TestWithMessageQueryParamsRepeatsListFilters(t *testing.T) {
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "domain/channels/chan/messages", MessagePageMetadata{
		Publishers: []string{"pub-a", "pub-b"},
		DeviceIDs:  []string{"Meter.A-01:X", "meter/b,02"},
		Publisher:  "pub-c",
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	q := parsed.Query()
	assert.Equal(t, []string{"pub-a", "pub-b"}, q["publishers"])
	assert.Equal(t, []string{"Meter.A-01:X", "meter/b,02"}, q["device_ids"])
	assert.Equal(t, []string{"pub-c"}, q["publisher"])
}

func TestWithMessageQueryParamsOmitsUnsetListFilters(t *testing.T) {
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "domain/channels/chan/messages", MessagePageMetadata{})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	q := parsed.Query()
	assert.NotContains(t, q, "publishers")
	assert.NotContains(t, q, "device_ids")
}

// An empty slice is erased by `omitempty` exactly as a nil one is, so it cannot
// express "match nothing" — the same contract readers.PageMetadata documents.
func TestWithMessageQueryParamsDropsEmptyListFilters(t *testing.T) {
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "domain/channels/chan/messages", MessagePageMetadata{
		DeviceIDs: []string{},
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	assert.NotContains(t, parsed.Query(), "device_ids")
}
