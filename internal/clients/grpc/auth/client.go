package auth

import (
	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	grpcClient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
)

const envAuthGrpcPrefix = "MF_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API
func Setup(envPrefix, jaegerURL string) (mainflux.AuthServiceClient, grpcClient.ClientHandler, error) {
	config := grpcClient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envAuthGrpcPrefix, AltPrefix: envPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}

	c, ch, err := grpcClient.Setup(config, "auth", jaegerURL)
	if err != nil {
		return nil, nil, err
	}

	return authapi.NewClient(c.Tracer, c.ClientConn, config.Timeout), ch, nil
}
