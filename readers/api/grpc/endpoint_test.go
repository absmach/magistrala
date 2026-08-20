// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	grpcapi "github.com/absmach/magistrala/readers/api/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	channelID    = "testChannelID"
	domain       = "testDomain"
	validID      = "validID"
	validToken   = "valid"
	inValidToken = "invalid"
	testOffset   = 0
	testLimit    = 10
)

func TestReadMessages(t *testing.T) {
	svc, addr := newTestServer(t)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewReadersClient(conn, time.Second)

	tmp := readers.MessagesPage{
		Total: 1,
		PageMetadata: readers.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
		Messages: []readers.Message{
			map[string]any{
				"channel":   "testChannel",
				"created":   int64(123456789),
				"subtopic":  "testSubtopic",
				"publisher": "testPublisher",
				"protocol":  "testProtocol",
				"device_id": "testDevice",
				"payload": map[string]any{
					"temp": 23.5,
				},
			},
		},
	}

	expectedPayload, err := json.Marshal(tmp.Messages[0].(map[string]any)["payload"])
	require.NoError(t, err)

	expectedRes := &grpcReadersV1.ReadMessagesRes{
		Total: 1,
		Messages: []*grpcReadersV1.Message{
			{
				Payload: &grpcReadersV1.Message_Json{
					Json: &grpcReadersV1.JsonMessage{
						Base: &grpcReadersV1.BaseMessage{
							Channel:   "testChannel",
							Subtopic:  "testSubtopic",
							Publisher: "testPublisher",
							Protocol:  "testProtocol",
							DeviceId:  "testDevice",
						},
						Created: 123456789,
						Payload: expectedPayload,
					},
				},
			},
		},
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
	}

	cases := []struct {
		desc            string
		token           string
		svcRes          readers.MessagesPage
		ReadMessagesReq *grpcReadersV1.ReadMessagesReq
		ReadMessagesRes *grpcReadersV1.ReadMessagesRes
		err             error
	}{
		{
			desc:  "read valid req",
			token: validToken,
			ReadMessagesReq: &grpcReadersV1.ReadMessagesReq{
				ChannelId: channelID,
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			svcRes: tmp,

			ReadMessagesRes: expectedRes,
			err:             nil,
		},
		{
			desc:  " read missing channel id",
			token: validToken,
			ReadMessagesReq: &grpcReadersV1.ReadMessagesReq{
				ChannelId: "",
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{},
			err:             apiutil.ErrMissingID,
		},
		{
			desc:  "read valid SenML message",
			token: validToken,
			ReadMessagesReq: &grpcReadersV1.ReadMessagesReq{
				ChannelId: channelID,
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			svcRes: readers.MessagesPage{
				Total: 1,
				PageMetadata: readers.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Messages: []readers.Message{
					senml.Message{
						Channel:     "senmlChannel",
						Subtopic:    "senmlSub",
						Publisher:   "senmlPublisher",
						Protocol:    "mqtt",
						DeviceId:    "senmlDevice",
						Name:        "temperature",
						Unit:        "C",
						Time:        1672531200,
						UpdateTime:  1672531300,
						Value:       float64Ptr(22.5),
						StringValue: stringPtr("ok"),
						DataValue:   stringPtr("binary"),
						BoolValue:   boolPtr(true),
						Sum:         float64Ptr(123.4),
					},
				},
			},
			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{
				Total: 1,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Messages: []*grpcReadersV1.Message{
					{
						Payload: &grpcReadersV1.Message_Senml{
							Senml: &grpcReadersV1.SenMLMessage{
								Base: &grpcReadersV1.BaseMessage{
									Channel:   "senmlChannel",
									Subtopic:  "senmlSub",
									Publisher: "senmlPublisher",
									Protocol:  "mqtt",
									DeviceId:  "senmlDevice",
								},
								Name:        "temperature",
								Unit:        "C",
								Time:        1672531200,
								UpdateTime:  1672531300,
								Value:       float64Ptr(22.5),
								StringValue: stringPtr("ok"),
								DataValue:   stringPtr("binary"),
								BoolValue:   boolPtr(true),
								Sum:         float64Ptr(123.4),
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		repoCall := svc.On("ReadAll", mock.Anything, mock.Anything).Return(tc.svcRes, tc.err)
		dpr, err := grpcClient.ReadMessages(context.Background(), tc.ReadMessagesReq)
		assert.Equal(t, tc.ReadMessagesRes.Messages, dpr.Messages, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ReadMessagesRes.Messages, dpr.Messages))

		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

// TestListGatewayDevices exercises the MG-15 gateway->devices RPC end to end:
// client, over the wire, to the server, to the (mocked) repository and back.
// Unlike the HTTP transport, there is no per-caller authorization here — see
// the comment on deviceViewReq — so this only has to show the aggregation
// itself round-trips correctly.
func TestListGatewayDevices(t *testing.T) {
	svc, addr := newTestServer(t)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	grpcClient := grpcapi.NewReadersClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcReadersV1.ListGatewayDevicesReq
		svcRes readers.DeviceStatsPage
		want   *grpcReadersV1.DeviceStatsRes
		err    error
	}{
		{
			desc: "valid request",
			req: &grpcReadersV1.ListGatewayDevicesReq{
				ChannelId:   channelID,
				DomainId:    domain,
				PublisherId: "1dcf1a0e-7a9d-4b1e-8d5f-9c2e6a3b4d01",
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			svcRes: readers.DeviceStatsPage{
				Total: 1,
				Stats: []readers.DeviceStat{{ID: "meter-a", LastSeen: 100, MessageCount: 5}},
			},
			want: &grpcReadersV1.DeviceStatsRes{
				Total: 1,
				Stats: []*grpcReadersV1.DeviceStat{{Id: "meter-a", LastSeen: 100, MessageCount: 5}},
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
		},
		{
			desc: "malformed publisher id",
			req: &grpcReadersV1.ListGatewayDevicesReq{
				ChannelId:   channelID,
				DomainId:    domain,
				PublisherId: "gateway-1",
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			want: &grpcReadersV1.DeviceStatsRes{},
			err:  apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "missing publisher id",
			req: &grpcReadersV1.ListGatewayDevicesReq{
				ChannelId: channelID,
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			want: &grpcReadersV1.DeviceStatsRes{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := svc.On("ListGatewayDevices", mock.Anything, mock.Anything, mock.Anything).Return(tc.svcRes, tc.err)
			defer repoCall.Unset()

			got, err := grpcClient.ListGatewayDevices(context.Background(), tc.req)
			assert.Equal(t, tc.want.Stats, got.Stats)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

// TestListDeviceGateways mirrors TestListGatewayDevices for the inverse
// direction.
func TestListDeviceGateways(t *testing.T) {
	svc, addr := newTestServer(t)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	grpcClient := grpcapi.NewReadersClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcReadersV1.ListDeviceGatewaysReq
		svcRes readers.DeviceStatsPage
		want   *grpcReadersV1.DeviceStatsRes
		err    error
	}{
		{
			desc: "valid request",
			req: &grpcReadersV1.ListDeviceGatewaysReq{
				ChannelId: channelID,
				DomainId:  domain,
				DeviceId:  "Meter.A-01:X",
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			svcRes: readers.DeviceStatsPage{
				Total: 2,
				Stats: []readers.DeviceStat{
					{ID: "gateway-1", LastSeen: 100, MessageCount: 3},
					{ID: "gateway-2", LastSeen: 200, MessageCount: 7},
				},
			},
			want: &grpcReadersV1.DeviceStatsRes{
				Total: 2,
				Stats: []*grpcReadersV1.DeviceStat{
					{Id: "gateway-1", LastSeen: 100, MessageCount: 3},
					{Id: "gateway-2", LastSeen: 200, MessageCount: 7},
				},
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
		},
		{
			desc: "missing device id",
			req: &grpcReadersV1.ListDeviceGatewaysReq{
				ChannelId: channelID,
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset: testOffset,
					Limit:  testLimit,
				},
			},
			want: &grpcReadersV1.DeviceStatsRes{},
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := svc.On("ListDeviceGateways", mock.Anything, mock.Anything, mock.Anything).Return(tc.svcRes, tc.err)
			defer repoCall.Unset()

			got, err := grpcClient.ListDeviceGateways(context.Background(), tc.req)
			assert.Equal(t, tc.want.Stats, got.Stats)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

// The list-valued filters have to survive the gRPC boundary intact; proto3
// `repeated` carries no field presence, so an absent list arrives as nil and
// there is no empty-versus-unset distinction to preserve on this path.
func TestReadMessagesCarriesListFilters(t *testing.T) {
	svc, addr := newTestServer(t)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewReadersClient(conn, time.Second)

	cases := []struct {
		desc              string
		deviceIDs         []string
		publishers        []string
		expectedDeviceIDs []string
	}{
		{
			desc:              "device ids and publishers travel together",
			deviceIDs:         []string{"Meter.A-01:X", "meter/b,02"},
			publishers:        []string{"pub-a"},
			expectedDeviceIDs: []string{"Meter.A-01:X", "meter/b,02"},
		},
		{
			desc:              "unset device ids arrive as nil",
			deviceIDs:         nil,
			publishers:        nil,
			expectedDeviceIDs: nil,
		},
		{
			desc:              "empty device ids are indistinguishable from unset over the wire",
			deviceIDs:         []string{},
			publishers:        []string{},
			expectedDeviceIDs: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var got readers.PageMetadata
			repoCall := svc.On("ReadAll", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					got = args.Get(1).(readers.PageMetadata)
				}).
				Return(readers.MessagesPage{}, nil)
			defer repoCall.Unset()

			_, err := grpcClient.ReadMessages(context.Background(), &grpcReadersV1.ReadMessagesReq{
				ChannelId: channelID,
				DomainId:  domain,
				PageMetadata: &grpcReadersV1.PageMetadata{
					Offset:     testOffset,
					Limit:      testLimit,
					DeviceIds:  tc.deviceIDs,
					Publishers: tc.publishers,
				},
			})
			require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
			assert.Equal(t, tc.expectedDeviceIDs, got.DeviceIDs)
		})
	}
}
