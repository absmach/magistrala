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
