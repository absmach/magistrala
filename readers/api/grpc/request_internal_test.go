// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"testing"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/readers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeReadMessagesRequestReadsDeviceIDs(t *testing.T) {
	got, err := decodeReadMessagesRequest(context.Background(), &grpcReadersV1.ReadMessagesReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Limit:     10,
			DeviceIds: []string{"meter-a", "meter-b"},
		},
	})
	require.NoError(t, err)

	req, ok := got.(readMessagesReq)
	require.True(t, ok)
	assert.Equal(t, []string{"meter-a", "meter-b"}, req.pageMeta.DeviceIDs)
}

func TestDecodeListGatewayDevicesRequest(t *testing.T) {
	got, err := decodeListGatewayDevicesRequest(context.Background(), &grpcReadersV1.ListGatewayDevicesReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		PublisherId: "gateway-1",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 5,
			Limit:  10,
			From:   100,
			To:     200,
		},
	})
	require.NoError(t, err)

	req, ok := got.(deviceViewReq)
	require.True(t, ok)
	assert.Equal(t, "channel", req.chanID)
	assert.Equal(t, "workspace", req.workspace)
	assert.Equal(t, "gateway-1", req.filterVal)
	assert.Equal(t, uint64(5), req.pageMeta.Offset)
	assert.Equal(t, uint64(10), req.pageMeta.Limit)
	assert.Equal(t, float64(100), req.pageMeta.From)
	assert.Equal(t, float64(200), req.pageMeta.To)
}

func TestDecodeListDeviceGatewaysRequest(t *testing.T) {
	got, err := decodeListDeviceGatewaysRequest(context.Background(), &grpcReadersV1.ListDeviceGatewaysReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		DeviceId:    "Meter.A-01:X",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
	})
	require.NoError(t, err)

	req, ok := got.(deviceViewReq)
	require.True(t, ok)
	assert.Equal(t, "Meter.A-01:X", req.filterVal)
}

// TestDecodeDeviceViewAppliesDefaultWindow asserts the gRPC device-view
// decoders bound a windowless query to the last 24h, mirroring the HTTP
// transport, so an unbounded GROUP BY is never the easy default. The default
// bounds are Unix nanoseconds, the unit the senml/json transformers store
// time in.
func TestDecodeDeviceViewAppliesDefaultWindow(t *testing.T) {
	window := float64(readers.DeviceViewDefaultWindow.Nanoseconds())
	before := time.Now()

	gotGateway, err := decodeListGatewayDevicesRequest(context.Background(), &grpcReadersV1.ListGatewayDevicesReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		PublisherId: "1dcf1a0e-7a9d-4b1e-8d5f-9c2e6a3b4d01",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
	})
	require.NoError(t, err)
	gatewayReq, ok := gotGateway.(deviceViewReq)
	require.True(t, ok)
	assert.GreaterOrEqual(t, gatewayReq.pageMeta.To, float64(before.UnixNano()))
	assert.InDelta(t, window, gatewayReq.pageMeta.To-gatewayReq.pageMeta.From, 2)

	gotDevice, err := decodeListDeviceGatewaysRequest(context.Background(), &grpcReadersV1.ListDeviceGatewaysReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		DeviceId:    "Meter.A-01:X",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
	})
	require.NoError(t, err)
	deviceReq, ok := gotDevice.(deviceViewReq)
	require.True(t, ok)
	assert.GreaterOrEqual(t, deviceReq.pageMeta.To, float64(before.UnixNano()))
	assert.InDelta(t, window, deviceReq.pageMeta.To-deviceReq.pageMeta.From, 2)
}

// TestDeviceViewReqValidateRejectsMalformedPublisherID asserts the gRPC
// device-view request rejects a publisher id that is not a UUID — the
// publishers column is a UUID column, so a malformed value must surface as a
// request error, not a database error — while leaving device serials (which
// have no format constraint, MG-09) untouched.
func TestDeviceViewReqValidateRejectsMalformedPublisherID(t *testing.T) {
	valid := deviceViewReq{
		chanID:            "channel",
		workspace:         "workspace",
		filterVal:         "1dcf1a0e-7a9d-4b1e-8d5f-9c2e6a3b4d01",
		filterIsPublisher: true,
		pageMeta:          readers.PageMetadata{Limit: 10},
	}
	require.NoError(t, valid.validate())

	malformed := valid
	malformed.filterVal = "not-a-uuid"
	require.Error(t, malformed.validate())

	serial := deviceViewReq{
		chanID:    "channel",
		workspace: "workspace",
		filterVal: "Meter.A-01:X",
		pageMeta:  readers.PageMetadata{Limit: 10},
	}
	require.NoError(t, serial.validate())
}

// A from/to bound outside the int64 range of the stored "time" column must
// be rejected as a request error: the column is a BIGINT of Unix
// nanoseconds, and binding an out-of-range float64 against it fails in the
// database driver rather than here.
func TestDeviceViewReqValidateRejectsOutOfRangeTimeBounds(t *testing.T) {
	valid := deviceViewReq{
		chanID:            "channel",
		workspace:         "workspace",
		filterVal:         "1dcf1a0e-7a9d-4b1e-8d5f-9c2e6a3b4d01",
		filterIsPublisher: true,
		pageMeta:          readers.PageMetadata{Limit: 10, From: 100, To: 200},
	}
	require.NoError(t, valid.validate())

	tooLarge := valid
	tooLarge.pageMeta.To = 9999999999999999999
	require.Error(t, tooLarge.validate())
}

// Same bound, on the plain ReadMessages request.
func TestReadMessagesReqValidateRejectsOutOfRangeTimeBounds(t *testing.T) {
	req := readMessagesReq{
		chanID:    "channel",
		workspace: "workspace",
		pageMeta:  readers.PageMetadata{Limit: 10, To: 9999999999999999999},
	}
	require.Error(t, req.validate())
}
