// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The list-valued message filters go out as repeated query parameters, one per
// value, so entries reach the reader byte-identically — device serials carry no
// format constraint (MG-09) and would not survive being joined on a separator.
func TestWithMessageQueryParamsRepeatsListFilters(t *testing.T) {
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "workspace/channels/chan/messages", MessagePageMetadata{
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
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "workspace/channels/chan/messages", MessagePageMetadata{})
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
	raw, err := mgSDK{}.withMessageQueryParams("http://localhost", "workspace/channels/chan/messages", MessagePageMetadata{
		DeviceIDs: []string{},
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	assert.NotContains(t, parsed.Query(), "device_ids")
}

// The MG-15 device-view URL carries the anchoring identity (the gateway
// publisher id or device serial) plus the paging/time bounds as query
// parameters — never as path segments, since a serial may contain `/`.
func TestWithDeviceViewQueryParamsIncludesAnchorAndBounds(t *testing.T) {
	raw, err := mgSDK{}.withDeviceViewQueryParams("http://localhost", "workspace/channels/chan/devices", "device_id", "Meter.A-01:X", DeviceViewPageMetadata{
		Offset: 5,
		Limit:  10,
		From:   100.5,
		To:     200,
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	assert.Equal(t, "/workspace/channels/chan/devices", parsed.Path)
	q := parsed.Query()
	assert.Equal(t, []string{"Meter.A-01:X"}, q["device_id"])
	assert.Equal(t, []string{"5"}, q["offset"])
	assert.Equal(t, []string{"10"}, q["limit"])
	assert.Equal(t, []string{"100.5"}, q["from"])
	assert.Equal(t, []string{"200"}, q["to"])
}

// The anchor value is URL-encoded, so a format-unconstrained serial (MG-09)
// with `/`, spaces and `:` round-trips byte-identically.
func TestWithDeviceViewQueryParamsEncodesAnchorValue(t *testing.T) {
	raw, err := mgSDK{}.withDeviceViewQueryParams("http://localhost", "workspace/channels/chan/publishers", "device_id", "meter/01:X y", DeviceViewPageMetadata{})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	assert.Equal(t, "/workspace/channels/chan/publishers", parsed.Path)
	assert.Equal(t, []string{"meter/01:X y"}, parsed.Query()["device_id"])
	assert.Len(t, parsed.Query(), 1, "an empty DeviceViewPageMetadata must add no other parameters")
}

// TestListGatewayDevicesBuildsURLAndUnmarshalsResponse exercises the
// gateway->devices SDK method end to end: the request URL and authorization
// header it constructs, and the unmarshalling of the DeviceStatsPage-shaped
// response.
func TestListGatewayDevicesBuildsURLAndUnmarshalsResponse(t *testing.T) {
	var (
		gotPath  string
		gotQuery url.Values
		gotAuth  string
		gotMeth  string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotAuth = r.Header.Get("Authorization")
		gotMeth = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":1,"offset":0,"limit":10,"devices":[{"device_id":"meter-1","last_seen":100,"message_count":5}]}`))
	}))
	defer server.Close()

	mg := mgSDK{
		readersURL:     server.URL,
		msgContentType: CTJSON,
		client:         server.Client(),
	}

	page, sdkerr := mg.ListGatewayDevices(context.Background(), "chan-1", "gateway-1", DeviceViewPageMetadata{Limit: 10}, "workspace-1", "token")
	require.Nil(t, sdkerr)

	assert.Equal(t, http.MethodGet, gotMeth)
	assert.Equal(t, "/workspace-1/channels/chan-1/devices", gotPath)
	assert.Equal(t, "Bearer token", gotAuth)
	assert.Equal(t, []string{"gateway-1"}, gotQuery["publisher"])
	assert.Equal(t, []string{"10"}, gotQuery["limit"])

	require.Len(t, page.Devices, 1)
	assert.Equal(t, "meter-1", page.Devices[0].DeviceID)
	assert.Equal(t, float64(100), page.Devices[0].LastSeen)
	assert.Equal(t, uint64(5), page.Devices[0].MessageCount)
	assert.Equal(t, uint64(1), page.Total)
}

// TestListDeviceGatewaysBuildsURLAndUnmarshalsResponse mirrors
// TestListGatewayDevicesBuildsURLAndUnmarshalsResponse for the inverse
// direction, using a serial with `/` to prove it survives URL-encoding.
func TestListDeviceGatewaysBuildsURLAndUnmarshalsResponse(t *testing.T) {
	var (
		gotPath  string
		gotQuery url.Values
		gotMeth  string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotMeth = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":2,"offset":0,"limit":10,"publishers":[{"publisher":"gw-a","last_seen":100,"message_count":3},{"publisher":"gw-b","last_seen":200,"message_count":7}]}`))
	}))
	defer server.Close()

	mg := mgSDK{
		readersURL:     server.URL,
		msgContentType: CTJSON,
		client:         server.Client(),
	}

	page, sdkerr := mg.ListDeviceGateways(context.Background(), "chan-1", "Meter.A/01:X", DeviceViewPageMetadata{Limit: 10}, "workspace-1", "token")
	require.Nil(t, sdkerr)

	assert.Equal(t, http.MethodGet, gotMeth)
	assert.Equal(t, "/workspace-1/channels/chan-1/publishers", gotPath)
	assert.Equal(t, []string{"Meter.A/01:X"}, gotQuery["device_id"])

	require.Len(t, page.Publishers, 2)
	assert.Equal(t, "gw-a", page.Publishers[0].Publisher)
	assert.Equal(t, float64(100), page.Publishers[0].LastSeen)
	assert.Equal(t, uint64(3), page.Publishers[0].MessageCount)
	assert.Equal(t, "gw-b", page.Publishers[1].Publisher)
	assert.Equal(t, uint64(2), page.Total)
}
