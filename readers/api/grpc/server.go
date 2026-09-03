// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcReadersV1.ReadersServiceServer = (*readersGrpcServer)(nil)

type readersGrpcServer struct {
	grpcReadersV1.UnimplementedReadersServiceServer
	readMessages       kitgrpc.Handler
	listGatewayDevices kitgrpc.Handler
	listDeviceGateways kitgrpc.Handler
}

func NewReadersServer(svc readers.MessageRepository) grpcReadersV1.ReadersServiceServer {
	return &readersGrpcServer{
		readMessages: kitgrpc.NewServer(
			readMessagesEndpoint(svc),
			decodeReadMessagesRequest,
			encodeReadMessagesResponse,
		),
		listGatewayDevices: kitgrpc.NewServer(
			listGatewayDevicesEndpoint(svc),
			decodeListGatewayDevicesRequest,
			encodeDeviceStatsResponse,
		),
		listDeviceGateways: kitgrpc.NewServer(
			listDeviceGatewaysEndpoint(svc),
			decodeListDeviceGatewaysRequest,
			encodeDeviceStatsResponse,
		),
	}
}

func decodeReadMessagesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcReadersV1.ReadMessagesReq)
	return readMessagesReq{
		chanID:    req.GetChannelId(),
		workspace: req.GetWorkspaceId(),
		pageMeta: readers.PageMetadata{
			Offset:      req.GetPageMetadata().GetOffset(),
			Limit:       req.GetPageMetadata().GetLimit(),
			Comparator:  req.GetPageMetadata().GetComparator(),
			Aggregation: stringifyAggregation(req.GetPageMetadata().GetAggregation()),
			From:        req.GetPageMetadata().GetFrom(),
			To:          req.GetPageMetadata().GetTo(),
			Interval:    req.GetPageMetadata().GetInterval(),
			Subtopic:    req.GetPageMetadata().GetSubtopic(),
			Publisher:   req.GetPageMetadata().GetPublisher(),
			Publishers:  req.GetPageMetadata().GetPublishers(),
			DeviceIDs:   req.GetPageMetadata().GetDeviceIds(),
			Protocol:    req.GetPageMetadata().GetProtocol(),
			Name:        req.GetPageMetadata().GetName(),
			Value:       req.GetPageMetadata().GetValue(),
			BoolValue:   req.GetPageMetadata().GetBoolValue(),
			StringValue: req.GetPageMetadata().GetStringValue(),
			DataValue:   req.GetPageMetadata().GetDataValue(),
			Format:      req.GetPageMetadata().GetFormat(),
			Order:       req.GetPageMetadata().GetOrder(),
			Dir:         req.GetPageMetadata().GetDir(),
		},
	}, nil
}

func encodeReadMessagesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(readMessagesRes)

	resp := &grpcReadersV1.ReadMessagesRes{
		Total:    res.Total,
		Messages: toResponseMessages(res.Messages),
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: res.PageMetadata.Offset,
			Limit:  res.PageMetadata.Limit,
			Order:  res.PageMetadata.Order,
			Dir:    res.PageMetadata.Dir,
		},
	}
	return resp, nil
}

func (s *readersGrpcServer) ReadMessages(ctx context.Context, req *grpcReadersV1.ReadMessagesReq) (*grpcReadersV1.ReadMessagesRes, error) {
	_, res, err := s.readMessages.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcReadersV1.ReadMessagesRes), nil
}

func decodeListGatewayDevicesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcReadersV1.ListGatewayDevicesReq)
	from, to := readers.DefaultTimeWindow(req.GetPageMetadata().GetFrom(), req.GetPageMetadata().GetTo())
	return deviceViewReq{
		chanID:            req.GetChannelId(),
		workspace:         req.GetWorkspaceId(),
		filterVal:         req.GetPublisherId(),
		filterIsPublisher: true,
		pageMeta: readers.PageMetadata{
			Offset: req.GetPageMetadata().GetOffset(),
			Limit:  req.GetPageMetadata().GetLimit(),
			From:   from,
			To:     to,
		},
	}, nil
}

func decodeListDeviceGatewaysRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcReadersV1.ListDeviceGatewaysReq)
	from, to := readers.DefaultTimeWindow(req.GetPageMetadata().GetFrom(), req.GetPageMetadata().GetTo())
	return deviceViewReq{
		chanID:    req.GetChannelId(),
		workspace: req.GetWorkspaceId(),
		filterVal: req.GetDeviceId(),
		pageMeta: readers.PageMetadata{
			Offset: req.GetPageMetadata().GetOffset(),
			Limit:  req.GetPageMetadata().GetLimit(),
			From:   from,
			To:     to,
		},
	}, nil
}

func encodeDeviceStatsResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(deviceStatsRes)

	return &grpcReadersV1.DeviceStatsRes{
		Total: res.Total,
		Stats: toResponseDeviceStats(res.Stats),
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: res.PageMetadata.Offset,
			Limit:  res.PageMetadata.Limit,
		},
	}, nil
}

func toResponseDeviceStats(stats []readers.DeviceStat) []*grpcReadersV1.DeviceStat {
	res := make([]*grpcReadersV1.DeviceStat, 0, len(stats))
	for _, s := range stats {
		res = append(res, &grpcReadersV1.DeviceStat{
			Id:           s.ID,
			LastSeen:     s.LastSeen,
			MessageCount: s.MessageCount,
		})
	}
	return res
}

func (s *readersGrpcServer) ListGatewayDevices(ctx context.Context, req *grpcReadersV1.ListGatewayDevicesReq) (*grpcReadersV1.DeviceStatsRes, error) {
	_, res, err := s.listGatewayDevices.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcReadersV1.DeviceStatsRes), nil
}

func (s *readersGrpcServer) ListDeviceGateways(ctx context.Context, req *grpcReadersV1.ListDeviceGatewaysReq) (*grpcReadersV1.DeviceStatsRes, error) {
	_, res, err := s.listDeviceGateways.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcReadersV1.DeviceStatsRes), nil
}

func toResponseMessages(messages []readers.Message) []*grpcReadersV1.Message {
	var res []*grpcReadersV1.Message
	for _, m := range messages {
		switch typed := m.(type) {
		case senml.Message:
			res = append(res, &grpcReadersV1.Message{
				Payload: &grpcReadersV1.Message_Senml{
					Senml: &grpcReadersV1.SenMLMessage{
						Base: &grpcReadersV1.BaseMessage{
							Channel:   typed.Channel,
							Subtopic:  typed.Subtopic,
							Publisher: typed.Publisher,
							Protocol:  typed.Protocol,
							DeviceId:  typed.DeviceId,
						},
						Name:        typed.Name,
						Unit:        typed.Unit,
						Time:        typed.Time,
						UpdateTime:  typed.UpdateTime,
						Value:       typed.Value,
						StringValue: typed.StringValue,
						DataValue:   typed.DataValue,
						BoolValue:   typed.BoolValue,
						Sum:         typed.Sum,
					},
				},
			})
		case map[string]any:
			payload := typed["payload"]
			data, err := json.Marshal(payload)
			if err != nil {
				continue
			}
			res = append(res, &grpcReadersV1.Message{
				Payload: &grpcReadersV1.Message_Json{
					Json: &grpcReadersV1.JsonMessage{
						Base: &grpcReadersV1.BaseMessage{
							Channel:   safeString(typed["channel"]),
							Subtopic:  safeString(typed["subtopic"]),
							Publisher: safeString(typed["publisher"]),
							Protocol:  safeString(typed["protocol"]),
							DeviceId:  safeString(typed["device_id"]),
						},
						Created: safeInt64(typed["created"]),
						Payload: data,
					},
				},
			})
		}
	}
	return res
}

func stringifyAggregation(agg grpcReadersV1.Aggregation) string {
	switch agg {
	case grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED:
		return ""
	case grpcReadersV1.Aggregation_AGGREGATION_MAX:
		return aggregationMax
	case grpcReadersV1.Aggregation_AGGREGATION_MIN:
		return aggregationMin
	case grpcReadersV1.Aggregation_AGGREGATION_AVG:
		return aggregationAvg
	case grpcReadersV1.Aggregation_AGGREGATION_SUM:
		return aggregationSum
	case grpcReadersV1.Aggregation_AGGREGATION_COUNT:
		return aggregationCount
	default:
		return ""
	}
}

func safeString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func safeInt64(v any) int64 {
	switch v := v.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}
