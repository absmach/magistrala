// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	"github.com/absmach/supermq/readers"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcReadersV1.ReadersServiceServer = (*readersGrpcServer)(nil)

type readersGrpcServer struct {
	grpcReadersV1.UnimplementedReadersServiceServer
	readMessages kitgrpc.Handler
}

func NewReadersServer(svc readers.MessageRepository) grpcReadersV1.ReadersServiceServer {
	return &readersGrpcServer{
		readMessages: kitgrpc.NewServer(
			(readMessagesEndpoint(svc)),
			decodeReadMessagesRequest,
			encodeReadMessagesResponse,
		),
	}
}

func decodeReadMessagesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcReadersV1.ReadMessagesReq)
	return readMessagesReq{
		chanID: req.GetChannelId(),
		domain: req.GetDomainId(),
	}, nil
}

func encodeReadMessagesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(readMessagesRes)

	return &grpcReadersV1.ReadMessagesRes{
		Total: res.Total,
		Messages: toResponseMessages(res.Messages),
		Offset: res.Offset,
		Limit:  res.Limit,
	}, nil
}

func (s *readersGrpcServer) ReadMessages(ctx context.Context, req *grpcReadersV1.ReadMessagesReq) (*grpcReadersV1.ReadMessagesRes, error) {
	_, res, err := s.readMessages.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcReadersV1.ReadMessagesRes), nil
}

func toResponseMessages(messages []readers.Message) []*grpcReadersV1.Message {
	var res []*grpcReadersV1.Message
	for _, m := range messages {
		data, _ := json.Marshal(m)
		res = append(res, &grpcReadersV1.Message{
			Data: data,
		})
	}
	return res
}
