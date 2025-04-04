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
				"publisher": "testPublisher",
				"subtopic":  "testSubtopic",
				"protocol":  "testProtocol",
				"time":      "2021-01-01T00:00:00Z",
				"channel":   1234,
				"name":      "testName",
				"unit":      "testUnit",
				"value":     30,
			},
		},
	}

	expectedData, err := json.Marshal(tmp.Messages[0])
	require.NoError(t, err)

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

			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{
				Total: 1,
				Messages: []*grpcReadersV1.Message{
					{Data: expectedData},
				},
			},
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
	}

	for _, tc := range cases {
		repoCall := svc.On("ReadAll", mock.Anything, mock.Anything).Return(tc.svcRes, tc.err)
		dpr, err := grpcClient.ReadMessages(context.Background(), tc.ReadMessagesReq)
		assert.Equal(t, tc.ReadMessagesRes.Messages, dpr.Messages, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ReadMessagesRes.Messages, dpr.Messages))

		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
