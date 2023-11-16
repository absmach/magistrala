// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/absmach/magistrala"
	authgrpc "github.com/absmach/magistrala/auth/api/grpc"
	grpcclient "github.com/absmach/magistrala/internal/clients/grpc"
	"github.com/absmach/magistrala/pkg/errors"
	thingsauth "github.com/absmach/magistrala/things/api/grpc"
	"github.com/caarlos0/env/v10"
)

const (
	envAuthGrpcPrefix  = "MG_AUTH_GRPC_"
	envAuthzGrpcPrefix = "MG_THINGS_AUTH_GRPC_"
)

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func Setup(svcName string) (magistrala.AuthServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.ParseWithOptions(&config, env.Options{Prefix: envAuthGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcclient.Setup(config, svcName)
	if err != nil {
		return nil, nil, err
	}

	return authgrpc.NewClient(c.ClientConn, config.Timeout), ch, nil
}

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func SetupAuthz(svcName string) (magistrala.AuthzServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.ParseWithOptions(&config, env.Options{Prefix: envAuthzGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcclient.Setup(config, svcName)
	if err != nil {
		return nil, nil, err
	}

	return thingsauth.NewClient(c.ClientConn, config.Timeout), ch, nil
}
