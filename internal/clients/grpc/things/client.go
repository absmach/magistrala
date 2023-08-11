// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	grpcclient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
	thingsapi "github.com/mainflux/mainflux/things/policies/api/grpc"
)

const envThingsAuthGrpcPrefix = "MF_THINGS_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Things gRPC configuration from environment variable and creates new Things gRPC API.
func Setup() (policies.AuthServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envThingsAuthGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}

	c, ch, err := grpcclient.Setup(config, "things")
	if err != nil {
		return nil, nil, err
	}

	return thingsapi.NewClient(c.ClientConn, config.Timeout), ch, nil
}
