// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	grpcClient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	authapi "github.com/mainflux/mainflux/users/policies/api/grpc"
)

const envAuthGrpcPrefix = "MF_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func Setup(envPrefix, jaegerURL, svcName string) (policies.AuthServiceClient, grpcClient.ClientHandler, error) {
	config := grpcClient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envAuthGrpcPrefix, AltPrefix: envPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcClient.Setup(config, svcName, jaegerURL)
	if err != nil {
		return nil, nil, err
	}

	return authapi.NewClient(c.ClientConn, config.Timeout), ch, nil
}
