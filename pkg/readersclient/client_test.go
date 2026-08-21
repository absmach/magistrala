// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package readersclient

import (
	"context"
	"errors"
	"testing"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	mgerrors "github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type readersClientStub struct {
	req         *grpcReadersV1.ReadMessagesReq
	res         *grpcReadersV1.ReadMessagesRes
	err         error
	hasDeadline bool

	gatewayDevicesReq *grpcReadersV1.ListGatewayDevicesReq
	deviceGatewaysReq *grpcReadersV1.ListDeviceGatewaysReq
	statsRes          *grpcReadersV1.DeviceStatsRes
	statsErr          error
}

func (stub *readersClientStub) ReadMessages(ctx context.Context, req *grpcReadersV1.ReadMessagesReq, _ ...grpc.CallOption) (*grpcReadersV1.ReadMessagesRes, error) {
	stub.req = req
	_, stub.hasDeadline = ctx.Deadline()
	return stub.res, stub.err
}

func (stub *readersClientStub) ListGatewayDevices(ctx context.Context, req *grpcReadersV1.ListGatewayDevicesReq, _ ...grpc.CallOption) (*grpcReadersV1.DeviceStatsRes, error) {
	stub.gatewayDevicesReq = req
	_, stub.hasDeadline = ctx.Deadline()
	return stub.statsRes, stub.statsErr
}

func (stub *readersClientStub) ListDeviceGateways(ctx context.Context, req *grpcReadersV1.ListDeviceGatewaysReq, _ ...grpc.CallOption) (*grpcReadersV1.DeviceStatsRes, error) {
	stub.deviceGatewaysReq = req
	_, stub.hasDeadline = ctx.Deadline()
	return stub.statsRes, stub.statsErr
}

func TestReadMessages(t *testing.T) {
	req := &grpcReadersV1.ReadMessagesReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		PageMetadata: &grpcReadersV1.PageMetadata{
			Publishers: []string{"publisher-1", "publisher-2"},
			Order:      "time",
			Dir:        "desc",
		},
	}
	expected := &grpcReadersV1.ReadMessagesRes{
		Total: 2,
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 3,
			Limit:  5,
			Order:  "time",
			Dir:    "desc",
		},
	}
	stub := &readersClientStub{res: expected}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	actual, err := client.ReadMessages(t.Context(), req)

	require.NoError(t, err)
	assert.Same(t, req, stub.req)
	assert.Same(t, expected, actual)
	assert.True(t, stub.hasDeadline)
}

func TestReadMessagesError(t *testing.T) {
	stub := &readersClientStub{err: status.Error(codes.NotFound, "missing messages")}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	res, err := client.ReadMessages(t.Context(), &grpcReadersV1.ReadMessagesReq{})

	require.Error(t, err)
	assert.NotNil(t, res)
	assert.True(t, mgerrors.Contains(err, svcerr.ErrNotFound))
}

func TestListGatewayDevices(t *testing.T) {
	req := &grpcReadersV1.ListGatewayDevicesReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		PublisherId: "publisher-1",
	}
	expected := &grpcReadersV1.DeviceStatsRes{
		Total: 1,
		Stats: []*grpcReadersV1.DeviceStat{{Id: "device-1", LastSeen: 100, MessageCount: 5}},
	}
	stub := &readersClientStub{statsRes: expected}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	actual, err := client.ListGatewayDevices(t.Context(), req)

	require.NoError(t, err)
	assert.Same(t, req, stub.gatewayDevicesReq)
	assert.Same(t, expected, actual)
	assert.True(t, stub.hasDeadline)
}

func TestListGatewayDevicesError(t *testing.T) {
	stub := &readersClientStub{statsErr: status.Error(codes.NotFound, "missing devices")}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	res, err := client.ListGatewayDevices(t.Context(), &grpcReadersV1.ListGatewayDevicesReq{})

	require.Error(t, err)
	assert.NotNil(t, res)
	assert.True(t, mgerrors.Contains(err, svcerr.ErrNotFound))
}

func TestListDeviceGateways(t *testing.T) {
	req := &grpcReadersV1.ListDeviceGatewaysReq{
		ChannelId:   "channel",
		WorkspaceId: "workspace",
		DeviceId:    "Meter.A-01:X",
	}
	expected := &grpcReadersV1.DeviceStatsRes{
		Total: 1,
		Stats: []*grpcReadersV1.DeviceStat{{Id: "publisher-1", LastSeen: 200, MessageCount: 3}},
	}
	stub := &readersClientStub{statsRes: expected}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	actual, err := client.ListDeviceGateways(t.Context(), req)

	require.NoError(t, err)
	assert.Same(t, req, stub.deviceGatewaysReq)
	assert.Same(t, expected, actual)
	assert.True(t, stub.hasDeadline)
}

func TestListDeviceGatewaysError(t *testing.T) {
	stub := &readersClientStub{statsErr: status.Error(codes.NotFound, "missing gateways")}
	client := &client{
		readers: stub,
		timeout: time.Second,
	}

	res, err := client.ListDeviceGateways(t.Context(), &grpcReadersV1.ListDeviceGatewaysReq{})

	require.Error(t, err)
	assert.NotNil(t, res)
	assert.True(t, mgerrors.Contains(err, svcerr.ErrNotFound))
}

func TestDecodeError(t *testing.T) {
	native := errors.New("native error")
	tests := []struct {
		name     string
		input    error
		expected error
	}{
		{
			name:     "unauthenticated",
			input:    status.Error(codes.Unauthenticated, "missing credentials"),
			expected: svcerr.ErrAuthentication,
		},
		{
			name:     "permission denied",
			input:    status.Error(codes.PermissionDenied, "forbidden"),
			expected: svcerr.ErrAuthorization,
		},
		{
			name:     "invalid argument",
			input:    status.Error(codes.InvalidArgument, "bad request"),
			expected: mgerrors.ErrMalformedEntity,
		},
		{
			name:     "failed precondition",
			input:    status.Error(codes.FailedPrecondition, "bad state"),
			expected: mgerrors.ErrMalformedEntity,
		},
		{
			name:     "not found",
			input:    status.Error(codes.NotFound, "missing"),
			expected: svcerr.ErrNotFound,
		},
		{
			name:     "already exists",
			input:    status.Error(codes.AlreadyExists, "duplicate"),
			expected: svcerr.ErrConflict,
		},
		{
			name:     "unknown status",
			input:    status.Error(codes.Unavailable, "offline"),
			expected: errors.New("unexpected gRPC status"),
		},
		{
			name:     "native error",
			input:    native,
			expected: native,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := decodeError(tc.input)
			if tc.name == "unknown status" {
				assert.ErrorContains(t, actual, tc.expected.Error())
				return
			}
			assert.True(t, mgerrors.Contains(actual, tc.expected), "expected %q to contain %q", actual, tc.expected)
		})
	}
}
