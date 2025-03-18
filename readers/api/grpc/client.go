// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	readers "github.com/absmach/supermq/readers"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const readersSvcName = "readers.v1.ReadersService"

var _ grpcReadersV1.ReadersServiceClient = (*readersGrpcClient)(nil)

type readersGrpcClient struct {
	readMessages endpoint.Endpoint
	timeout      time.Duration
}

// NewReadersClient returns new readers gRPC client instance.
func NewReadersClient(conn *grpc.ClientConn, timeout time.Duration) grpcReadersV1.ReadersServiceClient {
	return &readersGrpcClient{
		readMessages: kitgrpc.NewClient(
			conn,
			readersSvcName,
			"ReadMessages",
			encodeReadMessagesRequest,
			decodeReadMessagesResponse,
			grpcReadersV1.ReadMessagesRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client readersGrpcClient) ReadMessages(ctx context.Context, in *grpcReadersV1.ReadMessagesReq, opts ...grpc.CallOption) (*grpcReadersV1.ReadMessagesRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.readMessages(ctx, readMessagesReq{
		chanID: in.GetChannelId(),
		domain: in.GetDomainId(),
	})
	if err != nil {
		return &grpcReadersV1.ReadMessagesRes{}, grpcapi.DecodeError(err)
	}

	dpr := res.(readMessagesRes)
	return &grpcReadersV1.ReadMessagesRes{
		Total:    dpr.Total,
		Messages: toResponseMessages(dpr.Messages),
		Offset:   dpr.Offset,
	}, nil
}

func decodeReadMessagesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcReadersV1.ReadMessagesRes)
	return readMessagesRes{
		Total:    res.Total,
		Messages: fromResponseMessages(res.Messages),
	}, nil
}

func encodeReadMessagesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(readMessagesReq)
	return &grpcReadersV1.ReadMessagesReq{
		ChannelId: req.chanID,
		DomainId:  req.domain,
	}, nil
}

func fromResponseMessages(protoMessages []*grpcReadersV1.Message) []readers.Message {
	var messages []readers.Message
	for _, pm := range protoMessages {
		var m readers.Message
		if err := json.Unmarshal(pm.Data, &m); err != nil {
			return nil
		}
		messages = append(messages, m)
	}
	return messages
}
