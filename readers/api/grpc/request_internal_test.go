// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"testing"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeReadMessagesRequestReadsDeviceIDs(t *testing.T) {
	got, err := decodeReadMessagesRequest(context.Background(), &grpcReadersV1.ReadMessagesReq{
		ChannelId: "channel",
		DomainId:  "domain",
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
		DomainId:    "domain",
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
	assert.Equal(t, "domain", req.domain)
	assert.Equal(t, "gateway-1", req.filterVal)
	assert.Equal(t, uint64(5), req.pageMeta.Offset)
	assert.Equal(t, uint64(10), req.pageMeta.Limit)
	assert.Equal(t, float64(100), req.pageMeta.From)
	assert.Equal(t, float64(200), req.pageMeta.To)
}

func TestDecodeListDeviceGatewaysRequest(t *testing.T) {
	got, err := decodeListDeviceGatewaysRequest(context.Background(), &grpcReadersV1.ListDeviceGatewaysReq{
		ChannelId: "channel",
		DomainId:  "domain",
		DeviceId:  "Meter.A-01:X",
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
