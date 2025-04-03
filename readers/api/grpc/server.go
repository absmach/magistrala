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
		pageMeta: readers.PageMetadata{
			Offset:      req.GetPageMetadata().GetOffset(),
			Limit:       req.GetPageMetadata().GetLimit(),
			Comparator:  req.GetPageMetadata().GetComparator(),
			Aggregation: stringigyAggreation(req.GetPageMetadata().GetAggregation()),
			From:        req.GetPageMetadata().GetFrom(),
			To:          req.GetPageMetadata().GetTo(),
			Interval:    req.GetPageMetadata().GetInterval(),
			Subtopic:    req.GetPageMetadata().GetSubtopic(),
			Publisher:   req.GetPageMetadata().GetPublisher(),
			Protocol:    req.GetPageMetadata().GetProtocol(),
			Name:        req.GetPageMetadata().GetName(),
			Value:       req.GetPageMetadata().GetValue(),
			BoolValue:   req.GetPageMetadata().GetBoolValue(),
			StringValue: req.GetPageMetadata().GetStringValue(),
			DataValue:   req.GetPageMetadata().GetDataValue(),
			Format:      req.GetPageMetadata().GetFormat(),
		},
	}, nil
}

func encodeReadMessagesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(readMessagesRes)

	resp := &grpcReadersV1.ReadMessagesRes{
		Total:    res.Total,
		Messages: toResponseMessages(res.Messages),
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: res.PageMetadata.Offset,
			Limit:  res.PageMetadata.Limit,
		},
	}
	return resp, nil
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
		data, err := json.Marshal(m)
		if err != nil {
			continue
		}
		res = append(res, &grpcReadersV1.Message{
			Data: data,
		})
	}
	return res
}

func stringigyAggreation(agg grpcReadersV1.Aggregation) string {
	switch agg {
	case grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED:
		return ""
	case grpcReadersV1.Aggregation_MAX:
		return "MAX"
	case grpcReadersV1.Aggregation_MIN:
		return "MIN"
	case grpcReadersV1.Aggregation_AVG:
		return "AVG"
	case grpcReadersV1.Aggregation_SUM:
		return "SUM"
	case grpcReadersV1.Aggregation_COUNT:
		return "COUNT"
	default:
		return ""
	}
}
