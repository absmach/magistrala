package grpc

import (
	"context"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/authz"
	"github.com/mainflux/mainflux/authz/api/pb"
	"github.com/mainflux/mainflux/pkg/errors"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ pb.AuthZServiceServer = (*server)(nil)

type server struct {
	authorize kitgrpc.Handler
}

// NewServer returns new AuthnServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc authz.Service) pb.AuthZServiceServer {
	return &server{
		authorize: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "authorize")(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
	}
}
func (s *server) Authorize(ctx context.Context, req *pb.AuthorizeReq) (*pb.AuthorizeRes, error) {
	_, resp, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.(*pb.AuthorizeRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.AuthorizeReq)
	return AuthZReq{
		Sub: req.GetSub(),
		Obj: req.GetObj(),
		Act: req.GetAct(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (r interface{}, err error) {
	res := grpcRes.(authorizeRes)
	if res.err != "" {
		err = errors.New(res.err)
	}
	return &pb.AuthorizeRes{Authorized: res.authorized, Err: res.err}, encodeError(err)
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
