package grpc

import (
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	handler kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(svc things.Service) mainflux.ThingsServiceServer {
	handler := kitgrpc.NewServer(
		canAccessEndpoint(svc),
		decodeCanAccessRequest,
		encodeCanAccessResponse,
	)
	return &grpcServer{handler}
}

func (s *grpcServer) CanAccess(ctx context.Context, req *mainflux.AccessReq) (*mainflux.Identity, error) {
	_, res, err := s.handler.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.Identity), nil
}

func decodeCanAccessRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AccessReq)
	return accessReq{req.GetToken(), req.GetChanID()}, nil
}

func encodeCanAccessResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(accessRes)
	return &mainflux.Identity{Value: res.id}, encodeError(res.err)
}

func encodeError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case things.ErrMalformedEntity:
		return status.Error(codes.InvalidArgument, "received invalid can access request")
	case things.ErrUnauthorizedAccess:
		return status.Error(codes.PermissionDenied, "missing or invalid credentials provided")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
