package grpc

import (
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ mainflux.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	canAccess endpoint.Endpoint
	identify  endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn) mainflux.ThingsServiceClient {
	svcName := "mainflux.ThingsService"

	return &grpcClient{
		canAccess: kitgrpc.NewClient(
			conn,
			svcName,
			"CanAccess",
			encodeCanAccessRequest,
			decodeIdentityResponse,
			mainflux.Identity{},
		).Endpoint(),
		identify: kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			mainflux.Identity{},
		).Endpoint(),
	}
}

func (client grpcClient) CanAccess(ctx context.Context, req *mainflux.AccessReq, _ ...grpc.CallOption) (*mainflux.Identity, error) {
	res, err := client.canAccess(ctx, accessReq{req.GetToken(), req.GetChanID()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.Identity{Value: ir.id}, ir.err
}

func (client grpcClient) Identify(ctx context.Context, req *mainflux.Token, _ ...grpc.CallOption) (*mainflux.Identity, error) {
	res, err := client.identify(ctx, identifyReq{req.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.Identity{Value: ir.id}, ir.err
}

func encodeCanAccessRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(accessReq)
	return &mainflux.AccessReq{Token: req.thingKey, ChanID: req.chanID}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &mainflux.Token{Value: req.key}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.Identity)
	return identityRes{res.GetValue(), nil}, nil
}
