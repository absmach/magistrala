// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
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

	testMessages := []readers.Message{
		map[string]interface{}{"key": "value"},
	}

	cases := []struct {
		desc            string
		token           string
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
				Offset:    testOffset,
				Limit:     testLimit,
			},
			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{
				Total:    uint64(len(testMessages)),
				Messages: []*grpcReadersV1.Message{{Data: []byte(`{"key":"value"}`)}},
			},
		},
		{
			desc:            "read invalid req with invalid token",
			token:           inValidToken,
			ReadMessagesReq: &grpcReadersV1.ReadMessagesReq{},
			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{
				Total:    0,
				Messages: []*grpcReadersV1.Message{},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc:  "read invalid req with invalid token and missing domainID",
			token: inValidToken,
			ReadMessagesReq: &grpcReadersV1.ReadMessagesReq{
				ChannelId: channelID,
				DomainId: "",
			},
			ReadMessagesRes: &grpcReadersV1.ReadMessagesRes{
				Total:    0,
				Messages: []*grpcReadersV1.Message{},
			},
			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		repoCall := svc.On("ReadAll", mock.Anything, mock.Anything).Return(readers.MessagesPage{}, tc.err)
		dpr, err := grpcClient.ReadMessages(context.Background(), tc.ReadMessagesReq)
		assert.Equal(t, tc.ReadMessagesRes.Messages, dpr.Messages, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ReadMessagesRes.Messages, dpr.Messages))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
