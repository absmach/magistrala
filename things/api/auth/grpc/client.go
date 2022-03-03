// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mainflux/mainflux"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ mainflux.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout        time.Duration
	canAccessByKey endpoint.Endpoint
	canAccessByID  endpoint.Endpoint
	isChannelOwner endpoint.Endpoint
	identify       endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) mainflux.ThingsServiceClient {
	svcName := "mainflux.ThingsService"

	return &grpcClient{
		timeout: timeout,
		canAccessByKey: kitot.TraceClient(tracer, "can_access")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanAccessByKey",
			encodeCanAccessByKeyRequest,
			decodeIdentityResponse,
			mainflux.ThingID{},
		).Endpoint()),
		canAccessByID: kitot.TraceClient(tracer, "can_access_by_id")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanAccessByID",
			encodeCanAccessByIDRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		isChannelOwner: kitot.TraceClient(tracer, "is_channel_owner")(kitgrpc.NewClient(
			conn,
			svcName,
			"IsChannelOwner",
			encodeIsChannelOwner,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			mainflux.ThingID{},
		).Endpoint()),
	}
}

func (client grpcClient) CanAccessByKey(ctx context.Context, req *mainflux.AccessByKeyReq, _ ...grpc.CallOption) (*mainflux.ThingID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	ar := accessByKeyReq{
		thingKey: req.GetToken(),
		chanID:   req.GetChanID(),
	}
	res, err := client.canAccessByKey(ctx, ar)
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.ThingID{Value: ir.id}, nil
}

func (client grpcClient) CanAccessByID(ctx context.Context, req *mainflux.AccessByIDReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := accessByIDReq{thingID: req.GetThingID(), chanID: req.GetChanID()}
	res, err := client.canAccessByID(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) IsChannelOwner(ctx context.Context, req *mainflux.ChannelOwnerReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := channelOwnerReq{owner: req.GetOwner(), chanID: req.GetChanID()}
	res, err := client.isChannelOwner(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) Identify(ctx context.Context, req *mainflux.Token, _ ...grpc.CallOption) (*mainflux.ThingID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identifyReq{key: req.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.ThingID{Value: ir.id}, nil
}

func encodeCanAccessByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(accessByKeyReq)
	return &mainflux.AccessByKeyReq{Token: req.thingKey, ChanID: req.chanID}, nil
}

func encodeCanAccessByIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(accessByIDReq)
	return &mainflux.AccessByIDReq{ThingID: req.thingID, ChanID: req.chanID}, nil
}

func encodeIsChannelOwner(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(channelOwnerReq)
	return &mainflux.ChannelOwnerReq{Owner: req.owner, ChanID: req.chanID}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &mainflux.Token{Value: req.key}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.ThingID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}
