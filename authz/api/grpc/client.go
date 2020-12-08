package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/authz/api/pb"
	"github.com/mainflux/mainflux/pkg/errors"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ pb.AuthZServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	authorize endpoint.Endpoint
	timeout   time.Duration
}

// NewClient returns new AuthZServiceClient instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) pb.AuthZServiceClient {
	const svcName = "authz.AuthZService"
	return &grpcClient{
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			pb.AuthorizeRes{},
		).Endpoint(),
		timeout: timeout,
	}

}

func (client grpcClient) Authorize(ctx context.Context, req *pb.AuthorizeReq, _ ...grpc.CallOption) (r *pb.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.authorize(ctx, AuthZReq{Act: req.Act, Obj: req.Obj, Sub: req.Sub})
	if err != nil {
		return &pb.AuthorizeRes{Authorized: false, Err: err.Error()}, err
	}

	ar := res.(authorizeRes)
	if ar.err != "" {
		err = errors.New(ar.err)
	}
	return &pb.AuthorizeRes{Authorized: ar.authorized, Err: ar.err}, err
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(AuthZReq)
	return &pb.AuthorizeReq{
		Sub: req.Sub,
		Obj: req.Obj,
		Act: req.Act,
	}, nil
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*pb.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized, err: res.Err}, nil
}
