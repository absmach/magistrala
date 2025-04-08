// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
	readers "github.com/absmach/supermq/readers"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		pageMeta: readers.PageMetadata{
			Offset:      in.GetPageMetadata().GetOffset(),
			Limit:       in.GetPageMetadata().GetLimit(),
			Comparator:  in.GetPageMetadata().GetComparator(),
			Aggregation: in.GetPageMetadata().GetAggregation().String(),
			From:        in.GetPageMetadata().GetFrom(),
			To:          in.GetPageMetadata().GetTo(),
			Interval:    in.GetPageMetadata().GetInterval(),
			Subtopic:    in.GetPageMetadata().GetSubtopic(),
			Publisher:   in.GetPageMetadata().GetPublisher(),
			Protocol:    in.GetPageMetadata().GetProtocol(),
			Name:        in.GetPageMetadata().GetName(),
			Value:       in.GetPageMetadata().GetValue(),
			BoolValue:   in.GetPageMetadata().GetBoolValue(),
			StringValue: in.GetPageMetadata().GetStringValue(),
			DataValue:   in.GetPageMetadata().GetDataValue(),
			Format:      in.GetPageMetadata().GetFormat(),
		},
	})
	if err != nil {
		return &grpcReadersV1.ReadMessagesRes{}, decodeError(err)
	}

	dpr := res.(readMessagesRes)
	return &grpcReadersV1.ReadMessagesRes{
		Total:    dpr.Total,
		Messages: toResponseMessages(dpr.Messages),
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset: dpr.PageMetadata.Offset,
			Limit:  dpr.PageMetadata.Limit,
		},
	}, nil
}

func decodeReadMessagesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcReadersV1.ReadMessagesRes)
	return readMessagesRes{
		Total:    res.Total,
		Messages: fromResponseMessages(res.Messages),
		PageMetadata: readers.PageMetadata{
			Offset: res.GetPageMetadata().GetOffset(),
			Limit:  res.GetPageMetadata().GetLimit(),
		},
	}, nil
}

func encodeReadMessagesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(readMessagesReq)
	return &grpcReadersV1.ReadMessagesReq{
		ChannelId: req.chanID,
		DomainId:  req.domain,
		PageMetadata: &grpcReadersV1.PageMetadata{
			Offset:      req.pageMeta.Offset,
			Limit:       req.pageMeta.Limit,
			Comparator:  req.pageMeta.Comparator,
			Aggregation: parseAggregation(req.pageMeta.Aggregation),
			From:        req.pageMeta.From,
			To:          req.pageMeta.To,
			Interval:    req.pageMeta.Interval,
			Subtopic:    req.pageMeta.Subtopic,
			Publisher:   req.pageMeta.Publisher,
			Protocol:    req.pageMeta.Protocol,
			Name:        req.pageMeta.Name,
			Value:       req.pageMeta.Value,
			BoolValue:   req.pageMeta.BoolValue,
			StringValue: req.pageMeta.StringValue,
			DataValue:   req.pageMeta.DataValue,
			Format:      req.pageMeta.Format,
		},
	}, nil
}

func fromResponseMessages(protoMessages []*grpcReadersV1.Message) []readers.Message {
	var messages []readers.Message
	for _, m := range protoMessages {
		switch msg := m.Payload.(type) {
		case *grpcReadersV1.Message_Senml:
			s := msg.Senml
			base := s.GetBase()
			typed := senml.Message{
				Channel:     base.GetChannel(),
				Subtopic:    base.GetSubtopic(),
				Publisher:   base.GetPublisher(),
				Protocol:    base.GetProtocol(),
				Name:        s.GetName(),
				Unit:        s.GetUnit(),
				Time:        s.GetTime(),
				UpdateTime:  s.GetUpdateTime(),
				Value:       optionalFloat64(s.GetValue()),
				StringValue: optionalString(s.GetStringValue()),
				DataValue:   optionalString(s.GetDataValue()),
				BoolValue:   optionalBool(s.GetBoolValue()),
				Sum:         optionalFloat64(s.GetSum()),
			}
			messages = append(messages, typed)
		case *grpcReadersV1.Message_Json:
			j := msg.Json
			base := j.GetBase()
			var p map[string]interface{}
			if err := json.Unmarshal(j.GetPayload(), &p); err != nil {
				continue
			}
			messages = append(messages, map[string]interface{}{
				"channel":   base.GetChannel(),
				"created":   j.GetCreated(),
				"subtopic":  base.GetSubtopic(),
				"publisher": base.GetPublisher(),
				"protocol":  base.GetProtocol(),
				"payload":   p,
			})
		}
	}
	return messages
}

func parseAggregation(agg string) grpcReadersV1.Aggregation {
	switch strings.ToUpper(agg) {
	case "MAX":
		return grpcReadersV1.Aggregation_MAX
	case "MIN":
		return grpcReadersV1.Aggregation_MIN
	case "SUM":
		return grpcReadersV1.Aggregation_SUM
	case "COUNT":
		return grpcReadersV1.Aggregation_COUNT
	case "AVG":
		return grpcReadersV1.Aggregation_AVG
	default:
		return grpcReadersV1.Aggregation_AGGREGATION_UNSPECIFIED
	}
}

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
		case codes.PermissionDenied:
			return errors.Wrap(svcerr.ErrAuthorization, errors.New(st.Message()))
		case codes.InvalidArgument:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.FailedPrecondition:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.NotFound:
			return errors.Wrap(svcerr.ErrNotFound, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(svcerr.ErrConflict, errors.New(st.Message()))
		case codes.OK:
			if msg := st.Message(); msg != "" {
				return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
			}
			return nil
		default:
			return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
		}
	}
	return err
}

func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func optionalFloat64(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

func optionalBool(v bool) *bool {
	if !v {
		return nil
	}
	return &v
}
