package things

import (
	"github.com/mainflux/mainflux"
	grpcClient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
)

const envThingsAuthGrpcPrefix = "MF_THINGS_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Things gRPC configuration from environment variable and creates new Things gRPC API
func Setup(envPrefix, jaegerURL string) (mainflux.ThingsServiceClient, grpcClient.ClientHandler, error) {
	config := grpcClient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envThingsAuthGrpcPrefix, AltPrefix: envPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}

	c, ch, err := grpcClient.Setup(config, "things", jaegerURL)
	if err != nil {
		return nil, nil, err
	}

	return thingsapi.NewClient(c.ClientConn, c.Tracer, config.Timeout), ch, nil
}
