// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/errors"
	grpcapi "github.com/absmach/magistrala/readers/api/grpc"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/absmach/supermq/readers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port         = 7071
	channelID    = "testChannelID"
	domain       = "testDomain"
	validID      = "validID"
	validToken   = "valid"
	inValidToken = "invalid"
	testOffset   = 0
	testLimit    = 10
)

var authAddr = fmt.Sprintf("localhost:%d", port)

func startGRPCServer(svc readers.MessageRepository, port int) *grpc.Server {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	grpcReadersV1.RegisterReadersServiceServer(server, grpcapi.NewReadersServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating reader server %s"`, err))
	}()

	return server
}

func TestReadMessages(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewReadersClient(conn, time.Second)

	tmp := readers.MessagesPage{
		Total: 1,
		PageMetadata: readers.PageMetadata{
			Offset: 0,
			Limit:  10,
		},
		Messages: []readers.Message{
			map[string]interface{}{
				"channel":   "testChannel",
				"created":   int64(123456789),
				"subtopic":  "testSubtopic",
				"publisher": "testPublisher",
				"protocol":  "testProtocol",
				"payload": map[string]interface{}{
					"temp": 23.5,
				},
			},
		},
	}

	expectedPayload, err := json.Marshal(tmp.Messages[0].(map[string]interface{})["payload"])
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
								},
								Name:        "temperature",
								Unit:        "C",
								Time:        1672531200,
								UpdateTime:  1672531300,
								Value:       22.5,
								StringValue: "ok",
								DataValue:   "binary",
								BoolValue:   true,
								Sum:         123.4,
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

func float64Ptr(v float64) *float64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
