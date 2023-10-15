package grpcutil

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	rpbv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
)

// NewAuthlessReflectionInterceptor creates a proxy GRPCServer which automatically converts
// ServerReflectionServer instances to ones that skip grpc auth middleware.
//
// change:
// reflection.Register(srv)
// to:
// reflection.Register(grpcutil.NewAuthlessReflectionInterceptor(srv))
func NewAuthlessReflectionInterceptor(srv reflection.GRPCServer) reflection.GRPCServer {
	return interceptingRegistrar{srv}
}

type interceptingRegistrar struct {
	delegate reflection.GRPCServer
}

func (ir interceptingRegistrar) GetServiceInfo() map[string]grpc.ServiceInfo {
	return ir.delegate.GetServiceInfo()
}

func (ir interceptingRegistrar) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	// NOTE: This method is now being invoked for both v1 (handled below) and v1alpha, which is formally deprecated.
	// Since the v1alpha handler is now internally typed, we only wrap the V1.
	reflectionSrvv1, ok := impl.(rpbv1.ServerReflectionServer)
	if ok {
		ir.delegate.RegisterService(desc, &authlessReflectionV1{ServerReflectionServer: reflectionSrvv1})
	}
}

type authlessReflectionV1 struct {
	IgnoreAuthMixin

	rpbv1.ServerReflectionServer
}
