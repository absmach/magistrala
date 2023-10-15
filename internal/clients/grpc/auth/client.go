// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/mainflux/mainflux"
	authgrpc "github.com/mainflux/mainflux/auth/api/grpc"
	grpcclient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	thingsauth "github.com/mainflux/mainflux/things/api/grpc"
)

const envAuthGrpcPrefix = "MF_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func Setup(svcName string) (mainflux.AuthServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envAuthGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcclient.Setup(config, svcName)
	if err != nil {
		return nil, nil, err
	}

	return authgrpc.NewClient(c.ClientConn, config.Timeout), ch, nil
}

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func SetupAuthz(svcName string) (mainflux.AuthzServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envAuthGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcclient.Setup(config, svcName)
	if err != nil {
		return nil, nil, err
	}

	return thingsauth.NewClient(c.ClientConn, config.Timeout), ch, nil
}
