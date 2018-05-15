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
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn) mainflux.ThingsServiceClient {
	endpoint := kitgrpc.NewClient(
		conn,
		"mainflux.ThingsService",
		"CanAccess",
		encodeCanAccessRequest,
		decodeCanAccessResponse,
		mainflux.Identity{},
	).Endpoint()

	return &grpcClient{endpoint}
}

func (client grpcClient) CanAccess(ctx context.Context, req *mainflux.AccessReq, _ ...grpc.CallOption) (*mainflux.Identity, error) {
	res, err := client.canAccess(ctx, accessReq{req.GetToken(), req.GetChanID()})
	if err != nil {
		return nil, err
	}

	ar := res.(accessRes)
	return &mainflux.Identity{Value: ar.id}, ar.err
}

func encodeCanAccessRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(accessReq)
	return &mainflux.AccessReq{Token: req.thingKey, ChanID: req.chanID}, nil
}

func decodeCanAccessResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.Identity)
	return accessRes{res.GetValue(), nil}, nil
}
